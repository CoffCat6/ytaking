package web

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"myblog/internal/blog"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

func (s *Server) Index(w http.ResponseWriter, r *http.Request) {
	page := 1
	pageSize := 6
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}

	// Use ListPublishedPaginated to hide drafts
	posts, total := s.Store.ListPublishedPaginated(page, pageSize)

	var featured blog.Post
	var featuredPosts []blog.Post

	if page == 1 && len(posts) > 0 {
		featured = posts[0]
		for _, post := range posts {
			if post.Featured {
				featuredPosts = append(featuredPosts, post)
			}
		}
		if len(featuredPosts) == 0 {
			featuredPosts = append(featuredPosts, posts[0])
		}
	}

	tags, categories := collectFilters(s.Store.ListPublished()) // Only collect tags from published
	data := s.baseData(r)

	// SEO for Index
	if intro, ok := data["Intro"].(string); ok && intro != "" {
		data["Description"] = intro
		// log.Printf("Description set to: %s", intro)
	} else {
		data["Description"] = data["Tagline"]
		// log.Printf("Description set to tagline: %s", data["Tagline"])
	}
	data["CurrentPath"] = r.URL.Path

	data["Posts"] = posts
	data["Featured"] = featured
	data["FeaturedPosts"] = featuredPosts
	data["TagFilters"] = tags
	data["CategoryFilters"] = categories

	totalPages := (total + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	data["CurrentPage"] = page
	data["TotalPages"] = totalPages
	data["HasPrev"] = page > 1
	data["HasNext"] = page < totalPages
	data["PrevPage"] = page - 1
	data["NextPage"] = page + 1

	s.render(w, "index.html", data)
}

func (s *Server) PostsList(w http.ResponseWriter, r *http.Request) {
	data := s.baseData(r)
	data["Posts"] = s.Store.ListPublished()
	s.render(w, "posts.html", data)
}

func (s *Server) PostDetail(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/posts/")
	if slug == "" || slug == "/" {
		http.NotFound(w, r)
		return
	}

	post, ok := s.Store.GetBySlug(slug)
	if !ok {
		http.NotFound(w, r)
		return
	}

	// We allow viewing drafts via direct URL for preview purposes,
	// or we could check s.isAuthenticated(r) if we wanted strict privacy.
	// For now, let's allow it so the author can preview without logging in on a different browser,
	// provided the slug is guessed.

	related := s.Store.GetRelated(slug, 3)

	data := s.baseData(r)
	data["Post"] = post
	data["PostHTML"] = template.HTML(renderMarkdown(post.Content))
	data["RelatedPosts"] = related

	// SEO Data
	data["Title"] = post.Title + " - " + data["Title"].(string)
	if post.Summary != "" {
		data["Description"] = post.Summary
	}
	if len(post.Tags) > 0 {
		data["Keywords"] = strings.Join(post.Tags, ", ")
	}
	if post.CoverImage != "" {
		data["CoverImage"] = post.CoverImage
	}
	data["IsPost"] = true
	data["CurrentPath"] = r.URL.Path

	s.render(w, "post.html", data)
}

func (s *Server) ArchivePage(w http.ResponseWriter, r *http.Request) {
	type archiveGroup struct {
		Title string
		Posts []blog.Post
	}
	grouped := map[string][]blog.Post{}
	for _, post := range s.Store.ListPublished() {
		key := post.CreatedAt.Format("2006-01")
		grouped[key] = append(grouped[key], post)
	}

	var groups []archiveGroup
	for key, posts := range grouped {
		groups = append(groups, archiveGroup{Title: key, Posts: posts})
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Title > groups[j].Title
	})

	data := s.baseData(r)
	data["Archives"] = groups
	s.render(w, "archive.html", data)
}

func (s *Server) SearchPage(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	var results []blog.Post
	if query != "" {
		lower := strings.ToLower(query)
		for _, post := range s.Store.ListPublished() {
			if strings.Contains(strings.ToLower(post.Title), lower) ||
				strings.Contains(strings.ToLower(post.Summary), lower) ||
				strings.Contains(strings.ToLower(post.Content), lower) ||
				strings.Contains(strings.ToLower(post.Category), lower) ||
				containsTagLower(post.Tags, lower) {
				results = append(results, post)
			}
		}
	}
	data := s.baseData(r)
	data["Query"] = query
	data["Results"] = results

	if r.Header.Get("HX-Request") == "true" {
		s.renderPartial(w, "search_results.html", data)
		return
	}

	s.render(w, "search.html", data)
}

func collectFilters(posts []blog.Post) ([]string, []string) {
	tagSet := map[string]struct{}{}
	categorySet := map[string]struct{}{}
	for _, post := range posts {
		if post.Category != "" {
			categorySet[post.Category] = struct{}{}
		}
		for _, tag := range post.Tags {
			if tag == "" {
				continue
			}
			tagSet[tag] = struct{}{}
		}
	}

	var tags []string
	for tag := range tagSet {
		tags = append(tags, tag)
	}
	sort.Strings(tags)

	var categories []string
	for category := range categorySet {
		categories = append(categories, category)
	}
	sort.Strings(categories)
	return tags, categories
}

func (s *Server) AdminPosts(w http.ResponseWriter, r *http.Request) {
	page := 1
	pageSize := 20
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}

	// Admin sees ALL posts (including drafts)
	posts, total := s.Store.ListPaginated(page, pageSize)

	data := s.baseData(r)
	data["Posts"] = posts

	totalPages := (total + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	data["CurrentPage"] = page
	data["TotalPages"] = totalPages
	data["HasPrev"] = page > 1
	data["HasNext"] = page < totalPages
	data["PrevPage"] = page - 1
	data["NextPage"] = page + 1

	s.render(w, "admin_list.html", data)
}

func (s *Server) AdminSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		data := s.baseData(r)
		data["PageTitle"] = "站点设置"
		profile := s.SiteStore.Get()
		data["Profile"] = profile
		data["FocusText"] = strings.Join(profile.CurrentFocus, "\n")
		s.render(w, "admin_settings.html", data)
	case http.MethodPost:
		// TODO: Validate CSRF
		profile := parseSiteForm(r)
		if err := s.SiteStore.Update(profile); err != nil {
			data := s.baseData(r)
			data["PageTitle"] = "站点设置"
			data["Error"] = err.Error()
			data["Profile"] = profile
			data["FocusText"] = strings.Join(profile.CurrentFocus, "\n")
			s.render(w, "admin_settings.html", data)
			return
		}
		http.Redirect(w, r, "/admin/settings", http.StatusSeeOther)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) AdminPostNew(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		data := s.baseData(r)
		data["PageTitle"] = "新建文章"
		data["Post"] = blog.Post{}
		data["Action"] = "/admin/posts/new"
		s.render(w, "admin_form.html", data)
	case http.MethodPost:
		post := parsePostForm(r)
		if post.Slug == "" {
			post.Slug = slugify(post.Title)
		}
		if post.Slug == "" {
			s.renderAdminFormError(w, r, "新建文章", "需要填写 slug 或者标题包含英文/数字。", post, "/admin/posts/new")
			return
		}
		if err := s.Store.Create(post); err != nil {
			s.renderAdminFormError(w, r, "新建文章", err.Error(), post, "/admin/posts/new")
			return
		}
		http.Redirect(w, r, "/admin/posts", http.StatusSeeOther)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) AdminPostEdit(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		slug := r.URL.Query().Get("slug")
		post, ok := s.Store.GetBySlug(slug)
		if !ok {
			http.NotFound(w, r)
			return
		}
		data := s.baseData(r)
		data["PageTitle"] = "编辑文章"
		data["Post"] = post
		data["Action"] = "/admin/posts/edit?slug=" + slug
		s.render(w, "admin_form.html", data)
	case http.MethodPost:
		slug := r.URL.Query().Get("slug")
		post := parsePostForm(r)
		if post.Slug == "" {
			post.Slug = slugify(post.Title)
		}
		if post.Slug == "" {
			s.renderAdminFormError(w, r, "编辑文章", "需要填写 slug 或者标题包含英文/数字。", post, "/admin/posts/edit?slug="+slug)
			return
		}
		if err := s.Store.Update(slug, post); err != nil {
			s.renderAdminFormError(w, r, "编辑文章", err.Error(), post, "/admin/posts/edit?slug="+slug)
			return
		}
		http.Redirect(w, r, "/admin/posts", http.StatusSeeOther)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) AdminPostDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	slug := r.FormValue("slug")
	_ = s.Store.Delete(slug)
	http.Redirect(w, r, "/admin/posts", http.StatusSeeOther)
}

func (s *Server) AdminLogin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		data := s.baseData(r)
		data["PageTitle"] = "后台登录"
		s.render(w, "admin_login.html", data)
	case http.MethodPost:
		user := strings.TrimSpace(r.FormValue("username"))
		pass := strings.TrimSpace(r.FormValue("password"))
		if !s.validAdminCredentials(user, pass) {
			data := s.baseData(r)
			data["PageTitle"] = "后台登录"
			data["Error"] = "账号或密码错误"
			s.render(w, "admin_login.html", data)
			return
		}

		token, err := createSession()
		if err != nil {
			http.Error(w, "session error", http.StatusInternalServerError)
			return
		}
		setSessionCookie(w, token)
		http.Redirect(w, r, "/admin/posts", http.StatusSeeOther)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) AdminLogout(w http.ResponseWriter, r *http.Request) {
	clearSessionCookie(w)
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

func parsePostForm(r *http.Request) blog.Post {
	_ = r.ParseForm()
	return blog.Post{
		Title:      strings.TrimSpace(r.FormValue("title")),
		Slug:       strings.TrimSpace(r.FormValue("slug")),
		Summary:    strings.TrimSpace(r.FormValue("summary")),
		Content:    strings.TrimSpace(r.FormValue("content")),
		Category:   strings.TrimSpace(r.FormValue("category")),
		Tags:       splitComma(r.FormValue("tags")),
		CoverImage: strings.TrimSpace(r.FormValue("cover_image")),
		Featured:   r.FormValue("featured") == "on",
		IsDraft:    r.FormValue("is_draft") == "on",
	}
}

func parseSiteForm(r *http.Request) blog.SiteProfile {
	_ = r.ParseForm()
	return blog.SiteProfile{
		Title:        strings.TrimSpace(r.FormValue("title")),
		Tagline:      strings.TrimSpace(r.FormValue("tagline")),
		Intro:        strings.TrimSpace(r.FormValue("intro")),
		Location:     strings.TrimSpace(r.FormValue("location")),
		Email:        strings.TrimSpace(r.FormValue("email")),
		Newsletter:   strings.TrimSpace(r.FormValue("newsletter")),
		CurrentFocus: splitLines(strings.TrimSpace(r.FormValue("current_focus"))),
	}
}

func (s *Server) renderAdminFormError(w http.ResponseWriter, r *http.Request, pageTitle, msg string, post blog.Post, action string) {
	data := s.baseData(r)
	data["PageTitle"] = pageTitle
	data["Error"] = msg
	data["Post"] = post
	data["Action"] = action
	s.render(w, "admin_form.html", data)
}

func (s *Server) render(w http.ResponseWriter, page string, data map[string]any) {
	t, err := s.templateFor(page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := t.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) templateFor(page string) (*template.Template, error) {
	s.TemplateCache = ensureCache(s.TemplateCache)
	if t, ok := s.TemplateCache[page]; ok {
		return t, nil
	}

	files := []string{
		"internal/web/templates/base.html",
		"internal/web/templates/" + page,
	}
	if page == "search.html" {
		files = append(files, "internal/web/templates/search_results.html")
	}

	t, err := template.New("").Funcs(template.FuncMap{
		"formatDate": func(t time.Time) string {
			if t.IsZero() {
				return ""
			}
			return t.Format("2006-01-02")
		},
		"lower": func(input string) string {
			return strings.ToLower(input)
		},
		"joinTags": func(tags []string) string {
			return strings.Join(tags, ",")
		},
	}).ParseFiles(files...)
	if err != nil {
		return nil, err
	}
	s.TemplateCache[page] = t
	return t, nil
}

func (s *Server) renderPartial(w http.ResponseWriter, page string, data map[string]any) {
	t, err := template.New(filepath.Base(page)).Funcs(template.FuncMap{
		"formatDate": func(t time.Time) string {
			if t.IsZero() {
				return ""
			}
			return t.Format("2006-01-02")
		},
		"lower": func(input string) string {
			return strings.ToLower(input)
		},
		"joinTags": func(tags []string) string {
			return strings.Join(tags, ",")
		},
	}).ParseFiles("internal/web/templates/" + page)
	if err != nil {
		log.Printf("Template parse error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := t.Execute(w, data); err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func ensureCache(cache map[string]*template.Template) map[string]*template.Template {
	if cache == nil {
		return make(map[string]*template.Template)
	}
	return cache
}

func slugify(input string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(input) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if r == ' ' || r == '-' || r == '_' {
			if !lastDash && b.Len() > 0 {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	slug := strings.Trim(b.String(), "-")
	return slug
}

func (s *Server) baseData(r *http.Request) map[string]any {
	profile := s.SiteStore.Get()
	return map[string]any{
		"Title":        profile.Title,
		"Tagline":      profile.Tagline,
		"Intro":        profile.Intro,
		"Location":     profile.Location,
		"Email":        profile.Email,
		"Newsletter":   profile.Newsletter,
		"CurrentFocus": profile.CurrentFocus,
		"SiteURL":      s.Config.SiteBaseURL,
		"AdminURL":     s.Config.AdminBaseURL,
		"CSRFToken":    getCsrfToken(r),
	}
}

func splitLines(input string) []string {
	if input == "" {
		return nil
	}
	lines := strings.Split(input, "\n")
	var result []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		result = append(result, line)
	}
	return result
}

func renderMarkdown(input string) string {
	if strings.TrimSpace(input) == "" {
		return ""
	}
	var b strings.Builder
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(html.WithUnsafe()),
	)
	if err := md.Convert([]byte(input), &b); err != nil {
		return input
	}
	return b.String()
}

func splitComma(input string) []string {
	parts := strings.Split(input, ",")
	var result []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		result = append(result, part)
	}
	return result
}

func containsTag(tags []string, target string) bool {
	for _, tag := range tags {
		if tag == target {
			return true
		}
	}
	return false
}

func containsTagLower(tags []string, lower string) bool {
	for _, tag := range tags {
		if strings.Contains(strings.ToLower(tag), lower) {
			return true
		}
	}
	return false
}

func (s *Server) validAdminCredentials(user, pass string) bool {
	return user == s.Config.AdminUser && pass == s.Config.AdminPass
}
