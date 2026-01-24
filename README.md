# Blog Project Template

## Run

```bash
go run ./cmd/server
```

Open:

- Public: http://localhost:8080
- Admin: http://localhost:8081/admin/posts

## Content Storage

Posts are stored in `data/posts.json`.
Site profile is stored in `data/site.json`.

## Routes

- `/` Home
- `/posts` Post list
- `/posts/{slug}` Post detail
- Admin (port 8081):
  - `/admin/posts` Admin list
  - `/admin/posts/new` Create post
  - `/admin/posts/edit?slug=...` Edit post
  - `/admin/settings` Site settings

## Admin Auth

Use basic auth. Defaults:

- user: `admin`
- pass: `admin`

Override with environment variables:

```
ADMIN_USER=youruser
ADMIN_PASS=yourpass
SITE_BASE_URL=http://localhost:8080
ADMIN_BASE_URL=http://localhost:8081
```

## Next Improvements

- Add authentication for admin
- Markdown rendering
- Tag and archive pages
- Image upload and media library
- Pagination and search
- Deploy to VPS or container
