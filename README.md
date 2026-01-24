# Go Blog

这是一个简单、轻量级的个人博客系统，使用 Go 语言编写，支持 Markdown 文章编辑和 SQLite 数据存储。

## 快速开始

1.  **运行项目**

    ```bash
    go run ./cmd/server
    ```

2.  **访问项目**

    *   前台首页：<http://localhost:8080>
    *   后台管理：<http://localhost:8080/admin/posts>

## 部署文档

### 方式一：GitHub Pages 静态部署（推荐）

本项目支持一键生成静态网站，利用 GitHub Pages 免费托管。

#### 1. 准备工作

*   在 GitHub 上创建一个新仓库，例如 `myblog`。
*   在仓库设置 (Settings) -> Pages 中，将 Source 设置为 `GitHub Actions`。

#### 2. 本地写作流程

1.  **启动后台**：运行 `go run ./cmd/server`，进入后台写文章。
2.  **生成静态文件**：
    ```bash
    # 注意：如果不是发布到根域名，请设置 BASE_URL，例如：
    # export SITE_BASE_URL=https://yourname.github.io/myblog
    go run ./cmd/generator
    ```
    执行后会在 `dist/` 目录下生成所有 HTML 文件。
3.  **推送到 GitHub**：将代码提交并推送到 GitHub。

#### 3. 配置 GitHub Actions

在项目中创建 `.github/workflows/deploy.yml` 文件：

```yaml
name: Deploy to GitHub Pages

on:
  push:
    branches: ["main"]
  workflow_dispatch:

permissions:
  contents: read
  pages: write
  id-token: write

concurrency:
  group: "pages"
  cancel-in-progress: false

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Build and Generate
        run: |
          # 替换为你的 GitHub Pages URL
          export SITE_BASE_URL=https://${{ github.repository_owner }}.github.io/${{ github.event.repository.name }}
          go run ./cmd/generator

      - name: Upload artifact
        uses: actions/upload-pages-artifact@v3
        with:
          path: ./dist

  deploy:
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    runs-on: ubuntu-latest
    needs: build
    steps:
      - name: Deploy to GitHub Pages
        id: deployment
        uses: actions/deploy-pages@v4
```

### 方式二：Docker 动态部署

参考 [deployment_guide.md](deployment_guide.md) 获取详细说明。

## 配置说明

*   **数据存储**：默认存储在 `data/` 目录下的 SQLite 数据库。
*   **登录密码**：
    *   默认账号：`admin`
    *   默认密码：`admin`
    *   *建议在环境变量中设置 `ADMIN_PASS` 修改密码*。
