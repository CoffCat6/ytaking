# 构建阶段
FROM golang:1.24-alpine AS builder

# 不需要 gcc，因为 modernc.org/sqlite 是纯 Go 实现
# RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源码
COPY . .

# 编译应用
# 使用 CGO_ENABLED=0，因为 modernc.org/sqlite 不需要 CGO，且这样构建速度更快
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o blog-server ./cmd/server/main.go

# 运行阶段
FROM alpine:latest

WORKDIR /app

# 安装基础证书和时区数据
RUN apk add --no-cache ca-certificates tzdata

# 复制二进制文件
COPY --from=builder /app/blog-server .

# 复制静态资源和模板
COPY static ./static
COPY internal/web/templates ./internal/web/templates

# 创建数据目录
RUN mkdir -p data uploads/img

# 暴露端口
EXPOSE 8080 8081

# 启动命令
CMD ["./blog-server"]
