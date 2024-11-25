# 第一阶段：构建阶段
FROM golang:1.23.1-alpine AS build
ARG GO_OS="linux"
ARG GO_ARCH="amd64"
WORKDIR /usr/local/build/
COPY go.mod go.sum ./

# 下载依赖并验证模块
RUN GOPROXY=https://goproxy.cn,direct GOOS=${GO_OS} GOARCH=${GO_ARCH} go mod download && go mod verify

# 复制所有源代码到工作目录
COPY . .

# 构建二进制文件
RUN GOPROXY=https://goproxy.cn,direct GOOS=${GO_OS} GOARCH=${GO_ARCH} go build -v -o ndiff -ldflags '-s -w' cmd/main.go

# 第二阶段：生成运行时镜像
FROM alpine:3.18.4
RUN apk add --no-cache tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

# 添加用户并设置工作目录
RUN adduser -u 1000 -D sato -h /data
USER sato
WORKDIR /data
RUN chmod -R 755 /data

# 从构建阶段复制二进制文件和配置文件到运行时镜像
COPY --chown=sato --from=build /usr/local/build/ndiff /data/ndiff
COPY --chown=sato --from=build /usr/local/build/config/config.yaml /data/config.yaml

# 设置环境变量和暴露端口
# ENV LISTEN="0.0.0.0:8000"
# EXPOSE 8000

# 启动命令
CMD ["./ndiff", "-config", "config.yaml"]