package web

import (
	"encoding/xml"
	"net/http"
)

type URL struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod,omitempty"`
	ChangeFreq string `xml:"changefreq,omitempty"`
	Priority   string `xml:"priority,omitempty"`
}

type URLSet struct {
	XMLName xml.Name `xml:"http://www.sitemaps.org/schemas/sitemap/0.9 urlset"`
	URLs    []URL    `xml:"url"`
}

func (s *Server) Sitemap(w http.ResponseWriter, r *http.Request) {
	baseURL := s.Config.SiteBaseURL
	posts := s.Store.ListPublished()

	var urls []URL

	// Homepage
	urls = append(urls, URL{
		Loc:        baseURL + "/",
		ChangeFreq: "daily",
		Priority:   "1.0",
	})

	// Static pages
	urls = append(urls, URL{Loc: baseURL + "/archive", Priority: "0.5"})
	urls = append(urls, URL{Loc: baseURL + "/posts", Priority: "0.6"})

	// Posts
	for _, post := range posts {
		urls = append(urls, URL{
			Loc:        baseURL + "/posts/" + post.Slug,
			LastMod:    post.UpdatedAt.Format("2006-01-02"),
			ChangeFreq: "weekly",
			Priority:   "0.8",
		})
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(URLSet{URLs: urls}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
