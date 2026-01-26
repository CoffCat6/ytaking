package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/smtp"
	"os"
	"path/filepath"

	"myblog/internal/blog"
	"myblog/internal/config"
)

func main() {
	slug := flag.String("slug", "", "Slug of the post to notify about (default: latest published)")
	dryRun := flag.Bool("dry-run", false, "Dry run: print email instead of sending")
	flag.Parse()

	// Load Config
	cfg := config.Load()
	dbPath := filepath.Join(cfg.DataDir, "blog.db")

	// Init Store
	store, err := blog.NewSQLiteStore(dbPath)
	if err != nil {
		log.Fatalf("Failed to open store: %v", err)
	}

	// Init SiteStore for email config
	sitePath := filepath.Join(cfg.DataDir, "site.json")
	siteStore, err := blog.NewSiteStore(sitePath)
	if err != nil {
		log.Printf("Warning: failed to open site store: %v", err)
	}
	siteProfile := siteStore.Get()

	// 1. Get Post
	var post blog.Post
	if *slug != "" {
		var ok bool
		post, ok = store.GetBySlug(*slug)
		if !ok {
			log.Fatalf("Post not found: %s", *slug)
		}
	} else {
		posts := store.ListPublished()
		if len(posts) == 0 {
			log.Fatal("No published posts found")
		}
		post = posts[0]
	}

	// 2. Get Subscribers
	subs := store.ListSubscribers()
	if len(subs) == 0 {
		log.Println("No active subscribers.")
		return
	}

	log.Printf("Preparing to notify %d subscribers about '%s'...", len(subs), post.Title)

	// 3. Send Emails
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")

	if !*dryRun {
		if smtpHost == "" || smtpUser == "" {
			log.Fatal("SMTP settings missings (SMTP_HOST, SMTP_USER, etc)")
		}
	}

	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)

	// Simple email template
	tmpl := `Subject: {{.Title}}
MIME-Version: 1.0
Content-Type: text/html; charset=UTF-8

<!DOCTYPE html>
<html>
<head>
	<style>
		body { font-family: sans-serif; line-height: 1.6; color: #333; }
		.container { max-width: 600px; margin: 0 auto; padding: 20px; }
		.btn { background: #e35a3d; color: #fff; text-decoration: none; padding: 10px 20px; border-radius: 5px; display: inline-block; }
		.footer { font-size: 12px; color: #888; margin-top: 40px; border-top: 1px solid #eee; padding-top: 10px; }
	</style>
</head>
<body>
	<div class="container">
		<h2>新文章发布: {{.Title}}</h2>
		<p>{{.Summary}}</p>
		<p>
			<a href="{{.Link}}" class="btn">阅读全文</a>
		</p>
		<p class="footer">
			这是订阅通知。不想再收到邮件? <a href="mailto:{{.UnsubEmail}}?subject=Unsubscribe">回复退订</a>
		</p>
	</div>
</body>
</html>
`
	t := template.Must(template.New("email").Parse(tmpl))

	type EmailData struct {
		Title      string
		Summary    string
		Link       string
		UnsubEmail string
	}

	for _, sub := range subs {
		var body bytes.Buffer
		data := EmailData{
			Title:      post.Title,
			Summary:    post.Summary,
			Link:       fmt.Sprintf("%s/posts/%s", cfg.SiteBaseURL, post.Slug),
			UnsubEmail: siteProfile.Email,
		}
		if err := t.Execute(&body, data); err != nil {
			log.Printf("Template error: %v", err)
			continue
		}

		msg := []byte("To: " + sub.Email + "\r\n" + body.String())

		if *dryRun {
			log.Printf("[Dry Run] Sending to %s:\n%s\n", sub.Email, body.String())
		} else {
			addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)
			if err := smtp.SendMail(addr, auth, smtpUser, []string{sub.Email}, msg); err != nil {
				log.Printf("Failed to send to %s: %v", sub.Email, err)
			} else {
				log.Printf("Sent to %s", sub.Email)
			}
		}
	}
}
