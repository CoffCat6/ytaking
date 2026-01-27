package blog

type Store interface {
	List() []Post
	ListPublished() []Post
	ListPaginated(page, pageSize int) ([]Post, int)
	ListPublishedPaginated(page, pageSize int) ([]Post, int)
	GetBySlug(slug string) (Post, bool)
	GetRelated(slug string, n int) []Post
	Create(post Post) error
	Update(slug string, post Post) error
	Delete(slug string) error
}
