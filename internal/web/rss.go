package web

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/feeds"
	"myblog/internal/blog"
)

func (s *Server) RSS(w http.ResponseWriter, r *http.Request) {
	profile := s.SiteStore.Get()
	siteURL := s.Config.SiteBaseURL

	feed := &feeds.Feed{
		Title:       profile.Title,
		Link:        &feeds.Link{Href: siteURL},
		Description: profile.Intro,
		Author:      &feeds.Author{Name: "Author", Email: profile.Email},
		Created:     time.Now(),
	}

	posts := s.Store.List()
	// Filter drafts and limit to recent 20 posts
	var validPosts []blog.Post
	for _, p := range posts {
		if !p.IsDraft {
			validPosts = append(validPosts, p)
		}
	}
	if len(validPosts) > 20 {
		validPosts = validPosts[:20]
	}

	for _, post := range validPosts {
		feed.Items = append(feed.Items, &feeds.Item{
			Title:       post.Title,
			Link:        &feeds.Link{Href: siteURL + "/posts/" + post.Slug},
			Description: post.Summary,
			Created:     post.CreatedAt,
			Content:     renderMarkdown(post.Content), // Render full content
		})
	}

	w.Header().Set("Content-Type", "application/xml")
	if err := feed.WriteRss(w); err != nil {
		log.Printf("RSS error: %v", err)
		http.Error(w, "Failed to generate RSS", http.StatusInternalServerError)
	}
}
