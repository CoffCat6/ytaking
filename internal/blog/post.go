package blog

import (
	"fmt"
	"time"
)

type Post struct {
	Title      string    `json:"title"`
	Slug       string    `json:"slug"`
	Summary    string    `json:"summary"`
	Content    string    `json:"content"`
	Category   string    `json:"category"`
	Tags       []string  `json:"tags"`
	CoverImage string    `json:"cover_image"`
	Featured   bool      `json:"featured"`
	IsDraft    bool      `json:"is_draft"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (p Post) ReadTime() string {
	runes := []rune(p.Content)
	count := len(runes)
	minutes := count / 400
	if minutes < 1 {
		return "1 分钟"
	}
	return fmt.Sprintf("%d 分钟", minutes)
}
