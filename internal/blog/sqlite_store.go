package blog

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dsn string) (*SQLiteStore, error) {
	// 确保数据库文件所在的目录存在
	dir := filepath.Dir(dsn)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	// 开启 WAL 模式以提高并发性能和数据安全性
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}
	// 开启外键约束
	if _, err := db.Exec("PRAGMA foreign_keys=ON;"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}
	// 设置繁忙超时，防止 locked 错误
	if _, err := db.Exec("PRAGMA busy_timeout=5000;"); err != nil {
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	s := &SQLiteStore{db: db}
	if err := s.init(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *SQLiteStore) init() error {
	query := `
	CREATE TABLE IF NOT EXISTS posts (
		slug TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		summary TEXT,
		content TEXT,
		category TEXT,
		tags TEXT,
		cover_image TEXT,
		featured BOOLEAN,
		is_draft BOOLEAN,
		created_at DATETIME,
		updated_at DATETIME
	);
	CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts(created_at DESC);
	CREATE TABLE IF NOT EXISTS subscribers (
		email TEXT PRIMARY KEY,
		active BOOLEAN DEFAULT 1,
		created_at DATETIME
	);
	`
	_, err := s.db.Exec(query)
	return err
}

func (s *SQLiteStore) List() []Post {
	// List usually implies all posts, ordered by created_at desc
	return s.queryPosts("SELECT * FROM posts ORDER BY created_at DESC")
}

func (s *SQLiteStore) ListPublished() []Post {
	return s.queryPosts("SELECT * FROM posts WHERE is_draft = 0 ORDER BY created_at DESC")
}

func (s *SQLiteStore) ListPaginated(page, pageSize int) ([]Post, int) {
	total := s.count("SELECT COUNT(*) FROM posts")
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	query := fmt.Sprintf("SELECT * FROM posts ORDER BY created_at DESC LIMIT %d OFFSET %d", pageSize, offset)
	return s.queryPosts(query), total
}

func (s *SQLiteStore) ListPublishedPaginated(page, pageSize int) ([]Post, int) {
	total := s.count("SELECT COUNT(*) FROM posts WHERE is_draft = 0")
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	query := fmt.Sprintf("SELECT * FROM posts WHERE is_draft = 0 ORDER BY created_at DESC LIMIT %d OFFSET %d", pageSize, offset)
	return s.queryPosts(query), total
}

func (s *SQLiteStore) GetBySlug(slug string) (Post, bool) {
	posts := s.queryPosts("SELECT * FROM posts WHERE slug = ?", slug)
	if len(posts) == 0 {
		return Post{}, false
	}
	return posts[0], true
}

func (s *SQLiteStore) GetRelated(slug string, n int) []Post {
	// 1. Get current post tags
	current, ok := s.GetBySlug(slug)
	if !ok || len(current.Tags) == 0 {
		return []Post{}
	}

	// 2. Ideally we use full text search or a junction table for tags, but for simple port:
	// We will just fetch all published posts and do the scoring in memory (hybrid approach)
	// or use LIKE queries. Given SQLite and small dataset, hybrid is fine.
	// But let's try to be a bit smarter with SQL if possible.
	// Actually, just fetching all published is fine for < 1000 posts.
	// For "Modern & Intelligent", let's stick to the memory logic we just wrote,
	// but fetch from DB.

	all := s.ListPublished()
	// Re-use the scoring logic
	type scoredPost struct {
		post  Post
		score int
	}
	var candidates []scoredPost
	currentTags := make(map[string]bool)
	for _, t := range current.Tags {
		currentTags[t] = true
	}

	for _, p := range all {
		if p.Slug == slug {
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

	// Sort
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

func (s *SQLiteStore) Create(post Post) error {
	now := time.Now()
	if post.CreatedAt.IsZero() {
		post.CreatedAt = now
	}
	post.UpdatedAt = now

	tagsJSON, _ := json.Marshal(post.Tags)

	query := `
	INSERT INTO posts (slug, title, summary, content, category, tags, cover_image, featured, is_draft, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.Exec(query, post.Slug, post.Title, post.Summary, post.Content, post.Category, string(tagsJSON), post.CoverImage, post.Featured, post.IsDraft, post.CreatedAt, post.UpdatedAt)
	return err
}

func (s *SQLiteStore) Update(slug string, post Post) error {
	// Check if exists
	// We need original ID or just update by slug.
	// If slug changes, we need to handle that.
	// The interface implies "Update the post identified by 'slug' with new data 'post'".
	// If 'post.Slug' is different, we update the slug too.

	post.UpdatedAt = time.Now()
	tagsJSON, _ := json.Marshal(post.Tags)

	if post.Slug == "" {
		post.Slug = slug
	}

	query := `
	UPDATE posts SET 
		slug = ?, title = ?, summary = ?, content = ?, category = ?, tags = ?, 
		cover_image = ?, featured = ?, is_draft = ?, updated_at = ?
	WHERE slug = ?
	`
	_, err := s.db.Exec(query, post.Slug, post.Title, post.Summary, post.Content, post.Category, string(tagsJSON), post.CoverImage, post.Featured, post.IsDraft, post.UpdatedAt, slug)
	return err
}

func (s *SQLiteStore) Delete(slug string) error {
	_, err := s.db.Exec("DELETE FROM posts WHERE slug = ?", slug)
	return err
}

// Subscriber methods

func (s *SQLiteStore) AddSubscriber(email string) error {
	_, err := s.db.Exec(`
		INSERT INTO subscribers (email, active, created_at) 
		VALUES (?, ?, ?)
		ON CONFLICT(email) DO UPDATE SET active = 1
	`, email, true, time.Now())
	return err
}

func (s *SQLiteStore) RemoveSubscriber(email string) error {
	_, err := s.db.Exec("UPDATE subscribers SET active = 0 WHERE email = ?", email)
	return err
}

func (s *SQLiteStore) ListSubscribers() []Subscriber {
	rows, err := s.db.Query("SELECT email, active, created_at FROM subscribers WHERE active = 1 ORDER BY created_at DESC")
	if err != nil {
		return []Subscriber{}
	}
	defer rows.Close()

	var subs []Subscriber
	for rows.Next() {
		var s Subscriber
		var active bool
		if err := rows.Scan(&s.Email, &active, &s.CreatedAt); err != nil {
			continue
		}
		s.Active = active
		subs = append(subs, s)
	}
	return subs
}

// Helpers

func (s *SQLiteStore) queryPosts(query string, args ...any) []Post {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return []Post{}
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var p Post
		var tagsRaw string
		var featured, isDraft bool // SQLite stores boolean as 0/1, driver handles it usually?
		// modernc.org/sqlite handles bool as int64 usually. Let's be safe.
		// actually modernc sqlite driver maps BOOL to bool automatically if defined in table?
		// No, SQLite types are dynamic.
		// Let's scan into bool, standard `database/sql` converts 1/0 to bool.

		err := rows.Scan(
			&p.Slug, &p.Title, &p.Summary, &p.Content, &p.Category, &tagsRaw,
			&p.CoverImage, &featured, &isDraft, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			continue
		}
		p.Featured = featured
		p.IsDraft = isDraft
		_ = json.Unmarshal([]byte(tagsRaw), &p.Tags)
		posts = append(posts, p)
	}
	return posts
}

func (s *SQLiteStore) count(query string) int {
	var n int
	_ = s.db.QueryRow(query).Scan(&n)
	return n
}
