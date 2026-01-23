package blog

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

var ErrNotFound = errors.New("post not found")
var ErrDuplicateSlug = errors.New("post slug already exists")
var ErrInvalidSlug = errors.New("post slug is required")

type FileStore struct {
	path  string
	mu    sync.RWMutex
	posts []Post
}

func NewFileStore(path string) (*FileStore, error) {
	store := &FileStore{path: path}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *FileStore) List() []Post {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 返回副本，避免外部修改内部切片
	posts := make([]Post, len(s.posts))
	copy(posts, s.posts)
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].CreatedAt.After(posts[j].CreatedAt)
	})
	return posts
}

func (s *FileStore) GetBySlug(slug string) (Post, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, post := range s.posts {
		if post.Slug == slug {
			return post, true
		}
	}
	return Post{}, false
}

func (s *FileStore) Create(post Post) error {
	if post.Slug == "" {
		return ErrInvalidSlug
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, existing := range s.posts {
		if existing.Slug == post.Slug {
			return ErrDuplicateSlug
		}
	}

	now := time.Now()
	if post.CreatedAt.IsZero() {
		post.CreatedAt = now
	}
	post.UpdatedAt = now

	s.posts = append(s.posts, post)
	return s.save()
}

func (s *FileStore) Update(slug string, updated Post) error {
	if slug == "" {
		return ErrInvalidSlug
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	index := -1
	for i, post := range s.posts {
		if post.Slug == slug {
			index = i
			break
		}
	}
	if index == -1 {
		return ErrNotFound
	}

	if updated.Slug == "" {
		updated.Slug = s.posts[index].Slug
	} else if updated.Slug != slug {
		for _, existing := range s.posts {
			if existing.Slug == updated.Slug {
				return ErrDuplicateSlug
			}
		}
	}

	updated.CreatedAt = s.posts[index].CreatedAt
	updated.UpdatedAt = time.Now()

	s.posts[index] = updated
	return s.save()
}

func (s *FileStore) Delete(slug string) error {
	if slug == "" {
		return ErrInvalidSlug
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	index := -1
	for i, post := range s.posts {
		if post.Slug == slug {
			index = i
			break
		}
	}
	if index == -1 {
		return ErrNotFound
	}

	s.posts = append(s.posts[:index], s.posts[index+1:]...)
	return s.save()
}

func (s *FileStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := os.Stat(s.path); err != nil {
		if os.IsNotExist(err) {
			s.posts = []Post{}
			return nil
		}
		return err
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}

	if len(data) == 0 {
		s.posts = []Post{}
		return nil
	}

	var posts []Post
	if err := json.Unmarshal(data, &posts); err != nil {
		return err
	}

	s.posts = posts
	return nil
}

func (s *FileStore) save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s.posts, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(s.path, data, 0o644)
}

func (s *FileStore) ListPaginated(page, pageSize int) ([]Post, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	posts := make([]Post, len(s.posts))
	copy(posts, s.posts)
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].CreatedAt.After(posts[j].CreatedAt)
	})

	total := len(posts)
	if page < 1 {
		page = 1
	}
	start := (page - 1) * pageSize
	if start >= total {
		return []Post{}, total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return posts[start:end], total
}

func (s *FileStore) ListPublished() []Post {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var published []Post
	for _, p := range s.posts {
		if !p.IsDraft {
			published = append(published, p)
		}
	}
	// Sort desc
	sort.Slice(published, func(i, j int) bool {
		return published[i].CreatedAt.After(published[j].CreatedAt)
	})
	return published
}

func (s *FileStore) ListPublishedPaginated(page, pageSize int) ([]Post, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var published []Post
	for _, p := range s.posts {
		if !p.IsDraft {
			published = append(published, p)
		}
	}
	sort.Slice(published, func(i, j int) bool {
		return published[i].CreatedAt.After(published[j].CreatedAt)
	})

	total := len(published)
	if page < 1 {
		page = 1
	}
	start := (page - 1) * pageSize
	if start >= total {
		return []Post{}, total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return published[start:end], total
}

func (s *FileStore) GetRelated(slug string, n int) []Post {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var current Post
	found := false
	for _, p := range s.posts {
		if p.Slug == slug {
			current = p
			found = true
			break
		}
	}
	if !found || len(current.Tags) == 0 {
		return []Post{}
	}

	type scoredPost struct {
		post  Post
		score int
	}

	var candidates []scoredPost
	currentTags := make(map[string]bool)
	for _, t := range current.Tags {
		currentTags[t] = true
	}

	for _, p := range s.posts {
		if p.Slug == slug || p.IsDraft {
			continue
		}

		score := 0
		for _, t := range p.Tags {
			if currentTags[t] {
				score++
			}
		}

		if score > 0 {
			candidates = append(candidates, scoredPost{post: p, score: score})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return candidates[i].post.CreatedAt.After(candidates[j].post.CreatedAt)
	})

	if len(candidates) > n {
		candidates = candidates[:n]
	}

	var result []Post
	for _, c := range candidates {
		result = append(result, c.post)
	}
	return result
}
