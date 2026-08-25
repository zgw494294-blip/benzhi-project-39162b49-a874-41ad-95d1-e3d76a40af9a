# BENZHI_README

基于 Go 实现的舞台吊挂安全启用工作台 Web 项目，一款后端服务，舞台吊挂安全启用工作台已完整实现：剧场技术团队可在浏览器中建立方案、录入载荷与动作、自动校核风险、记录联排、提交整改、执行独立评审，并验证冻结后的演出启用单。服务默认安全监听 `127.0.0.1:19081`，支持 `-addr` 和合法 `PORT` 配置。

## 项目说明
- 项目：benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a
- 项目用途：舞台吊挂安全启用工作台已完整实现：剧场技术团队可在浏览器中建立方案、录入载荷与动作、自动校核风险、记录联排、提交整改、执行独立评审，并验证冻结后的演出启用单。服务默认安全监听 `127.0.0.1:19081`，支持 `-addr` 和合法 `PORT` 配置。
- Go 工具链：`golang:1.24.0`
- 前端工具链：原生 HTML、CSS 和 JavaScript，由 Go 服务直接提供

## 标准构建、运行和测试命令
进入容器后执行：
cd '/app' && GOTOOLCHAIN=local go build ./...
cd /app && GOTOOLCHAIN=local go run ./cmd/server -selfcheck -selfcheck-timeout=20s -addr=127.0.0.1:19081
cd '/app' && GOTOOLCHAIN=local go test ./...

## Docker 构建和进入容器
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a-amd64 linux/amd64
./build_benzhi_docker.sh benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a-arm64 linux/arm64
docker run -it benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a-amd64:latest

## 题目验证命令
1. 预期退出码 0：`go test ./...`
2. 预期退出码 0：`go run ./cmd/server -selfcheck -selfcheck-timeout=20s -addr=127.0.0.1:19081`
