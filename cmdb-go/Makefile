# CMDB服务树数据爬取工具 Makefile

# 项目信息
PROJECT_NAME := cmdb-crawler
VERSION ?= 1.0.0
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# 构建目录
BUILD_DIR := build
DIST_DIR := dist
OUTPUT_DIR := output
LOG_DIR := logs

# Go 构建参数
LDFLAGS := -s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)
GOBUILD := go build -ldflags="$(LDFLAGS)"
GOBUILD_DEBUG := go build -gcflags="all=-N -l"

# 调试参数
DEBUG_PORT ?= 2345
DEBUG_HOST ?= localhost

# 默认目标
.PHONY: all
all: clean deps build

# 清理构建文件
.PHONY: clean
clean:
	@echo "清理构建文件..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR)
	@rm -f $(PROJECT_NAME) $(PROJECT_NAME).exe $(PROJECT_NAME)-debug
	@echo "清理完成"

# 安装依赖
.PHONY: deps
deps:
	@echo "安装依赖..."
	@go mod tidy
	@go mod download
	@echo "依赖安装完成"

# 构建程序（当前平台）
.PHONY: build
build: deps
	@echo "构建程序..."
	@mkdir -p $(OUTPUT_DIR) $(LOG_DIR)
	@$(GOBUILD) -o $(PROJECT_NAME) main.go
	@echo "构建完成: $(PROJECT_NAME)"

# 构建调试版本
.PHONY: build-debug
build-debug: deps
	@echo "构建调试版本..."
	@mkdir -p $(OUTPUT_DIR) $(LOG_DIR)
	@$(GOBUILD_DEBUG) -o $(PROJECT_NAME)-debug main.go
	@echo "调试版本构建完成: $(PROJECT_NAME)-debug"

# 使用dlv调试程序
.PHONY: debug
debug: build-debug
	@echo "启动调试器..."
	@echo "调试端口: $(DEBUG_HOST):$(DEBUG_PORT)"
	@echo "使用以下命令连接:"
	@echo "  dlv connect $(DEBUG_HOST):$(DEBUG_PORT)"
	@echo "或者直接在代码中设置断点后运行:"
	@echo "注意: 程序执行完成后，需要手动按 Ctrl+C 退出 dlv 服务器"
	@echo "如果要自动退出，请使用: make debug-direct"
	@dlv exec ./$(PROJECT_NAME)-debug crawl --headless --listen=$(DEBUG_HOST):$(DEBUG_PORT) --api-version=2 --accept-multiclient

# 直接调试（不启用headless模式）
.PHONY: debug-direct
debug-direct: build-debug
	@echo "直接调试模式..."
	@dlv exec ./$(PROJECT_NAME)-debug -- crawl

# 自动退出的调试模式
.PHONY: debug-auto
debug-auto: build-debug
	@echo "自动退出调试模式..."
	@echo "程序执行完成后将自动退出"
	@timeout 300 dlv exec ./$(PROJECT_NAME)-debug crawl --headless --listen=$(DEBUG_HOST):$(DEBUG_PORT) --api-version=2 || true

# 调试特定命令
.PHONY: debug-cmd
debug-cmd: build-debug
	@echo "调试特定命令: $(CMD)"
	@dlv exec ./$(PROJECT_NAME)-debug -- $(CMD)

# 调试测试
.PHONY: debug-test
debug-test:
	@echo "调试测试..."
	@dlv test ./... --headless --listen=$(DEBUG_HOST):$(DEBUG_PORT) --api-version=2

# 附加到运行中的进程调试
.PHONY: debug-attach
debug-attach:
	@echo "附加到进程调试..."
	@echo "请输入要附加的进程ID:"
	@read pid && dlv attach $$pid --headless --listen=$(DEBUG_HOST):$(DEBUG_PORT) --api-version=2

# 构建所有平台
.PHONY: build-all
build-all: clean deps
	@echo "构建所有平台..."
	@chmod +x scripts/build.sh
	@./scripts/build.sh
	@echo "所有平台构建完成"

# 运行程序
.PHONY: run
run: build
	@echo "运行爬取工具..."
	@./$(PROJECT_NAME) crawl

# 运行指定服务树
.PHONY: run-views
run-views: build
	@echo "运行爬取工具（指定视图）..."
	@./$(PROJECT_NAME) crawl --views "$(VIEWS)"

# 快速运行（使用脚本）
.PHONY: quick-run
quick-run:
	@chmod +x scripts/run.sh
	@./scripts/run.sh $(ARGS)

# 运行测试
.PHONY: test
test:
	@echo "运行测试..."
	@go test -v ./...

# 代码格式化
.PHONY: fmt
fmt:
	@echo "格式化代码..."
	@go fmt ./...
	@goimports -w .

# 代码检查
.PHONY: lint
lint:
	@echo "代码检查..."
	@golangci-lint run

# 生成文档
.PHONY: docs
docs:
	@echo "生成文档..."
	@godoc -http=:6060 &
	@echo "文档服务器启动在 http://localhost:6060"

# 安装开发工具
.PHONY: install-tools
install-tools:
	@echo "安装开发工具..."
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "工具安装完成"

# 安装dlv调试器
.PHONY: install-dlv
install-dlv:
	@echo "安装dlv调试器..."
	@go install github.com/go-delve/delve/cmd/dlv@latest
	@echo "dlv调试器安装完成"
	@echo "验证安装:"
	@dlv version || echo "dlv安装可能失败，请检查GOPATH和PATH设置"

# 创建发布版本
.PHONY: release
release: clean build-all
	@echo "创建发布版本 $(VERSION)..."
	@echo "版本: $(VERSION)" > $(DIST_DIR)/VERSION
	@echo "构建时间: $(BUILD_TIME)" >> $(DIST_DIR)/VERSION
	@echo "Git提交: $(GIT_COMMIT)" >> $(DIST_DIR)/VERSION
	@echo "发布版本创建完成"

# 安装到系统
.PHONY: install
install: build
	@echo "安装到系统..."
	@sudo cp $(PROJECT_NAME) /usr/local/bin/
	@echo "安装完成: /usr/local/bin/$(PROJECT_NAME)"

# 卸载
.PHONY: uninstall
uninstall:
	@echo "从系统卸载..."
	@sudo rm -f /usr/local/bin/$(PROJECT_NAME)
	@echo "卸载完成"

# Docker构建
.PHONY: docker-build
docker-build:
	@echo "构建Docker镜像..."
	@docker build -t $(PROJECT_NAME):$(VERSION) .
	@docker tag $(PROJECT_NAME):$(VERSION) $(PROJECT_NAME):latest
	@echo "Docker镜像构建完成"

# Docker运行
.PHONY: docker-run
docker-run: docker-build
	@echo "运行Docker容器..."
	@docker run -v $(PWD)/config:/app/config -v $(PWD)/output:/app/output $(PROJECT_NAME):$(VERSION) crawl

# 显示版本信息
.PHONY: version
version:
	@echo "项目: $(PROJECT_NAME)"
	@echo "版本: $(VERSION)"
	@echo "构建时间: $(BUILD_TIME)"
	@echo "Git提交: $(GIT_COMMIT)"

# 初始化项目
.PHONY: init
init:
	@echo "初始化项目..."
	@mkdir -p $(OUTPUT_DIR) $(LOG_DIR) config examples scripts
	@if [ ! -f config/config.yaml ]; then \
		echo "创建默认配置文件..."; \
		cp config/config.yaml.example config/config.yaml 2>/dev/null || true; \
	fi
	@chmod +x scripts/*.sh 2>/dev/null || true
	@echo "项目初始化完成"

# 显示帮助
.PHONY: help
help:
	@echo "CMDB服务树数据爬取工具 - Makefile"
	@echo ""
	@echo "构建相关:"
	@echo "  all          - 清理、安装依赖、构建"
	@echo "  build        - 构建当前平台的程序"
	@echo "  build-debug  - 构建调试版本（禁用优化）"
	@echo "  build-all    - 构建所有平台的程序"
	@echo "  clean        - 清理构建文件"
	@echo "  deps         - 安装依赖"
	@echo ""
	@echo "运行相关:"
	@echo "  run          - 构建并运行程序"
	@echo "  run-views    - 运行指定视图 (使用 VIEWS=视图名称)"
	@echo "  quick-run    - 使用脚本快速运行 (使用 ARGS=参数)"
	@echo ""
	@echo "调试相关:"
	@echo "  debug        - 使用dlv调试程序 (headless模式，需手动退出)"
	@echo "  debug-direct - 直接调试模式 (交互式，自动退出)"
	@echo "  debug-auto   - 自动退出调试模式 (headless模式，程序完成后自动退出)"
	@echo "  debug-cmd    - 调试特定命令 (使用 CMD=命令参数)"
	@echo "  debug-test   - 调试测试"
	@echo "  debug-attach - 附加到运行中的进程调试"
	@echo "  install-dlv  - 安装dlv调试器"
	@echo ""
	@echo "开发相关:"
	@echo "  test         - 运行测试"
	@echo "  fmt          - 格式化代码"
	@echo "  lint         - 代码检查"
	@echo "  docs         - 启动文档服务器"
	@echo "  install-tools- 安装开发工具"
	@echo ""
	@echo "部署相关:"
	@echo "  release      - 创建发布版本"
	@echo "  install      - 安装到系统"
	@echo "  uninstall    - 从系统卸载"
	@echo "  docker-build - 构建Docker镜像"
	@echo "  docker-run   - 运行Docker容器"
	@echo ""
	@echo "其他:"
	@echo "  version      - 显示版本信息"
	@echo "  init         - 初始化项目"
	@echo "  help         - 显示此帮助信息"
	@echo ""
	@echo "调试使用示例:"
	@echo "  make debug                               # 启动调试服务器（需手动退出）"
	@echo "  make debug-direct                       # 直接调试（推荐）"
	@echo "  make debug-auto                         # 自动退出调试"
	@echo "  make debug-cmd CMD=\"crawl --verbose\"   # 调试特定命令"
	@echo "  make debug DEBUG_PORT=3456               # 使用指定端口调试"
	@echo ""
	@echo "其他示例:"
	@echo "  make run                                 # 构建并运行"
	@echo "  make run-views VIEWS=\"产品服务树\"      # 运行指定视图"
	@echo "  make quick-run ARGS=\"crawl --format csv\" # 快速运行"
	@echo "  make release VERSION=1.1.0              # 创建指定版本"

# 设置默认目标
.DEFAULT_GOAL := help 