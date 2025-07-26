#!/bin/bash

# CMDB爬取工具快速运行脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_header() {
    echo -e "${BLUE}$1${NC}"
}

# 检查Go环境
check_go() {
    if ! command -v go &> /dev/null; then
        log_error "Go未安装，请先安装Go 1.21+版本"
        log_info "下载地址: https://golang.org/dl/"
        exit 1
    fi
    
    go_version=$(go version | awk '{print $3}' | sed 's/go//')
    log_info "Go版本: $go_version"
}

# 检查配置文件
check_config() {
    if [ ! -f "config/config.yaml" ]; then
        log_warn "配置文件不存在，创建默认配置..."
        mkdir -p config
        cat > config/config.yaml << 'EOF'
cmdb:
  base_url: "http://localhost:8080"
  api_version: "v0.1"
  auth:
    username: "admin"
    password: "admin"
  request:
    timeout: 30s
    retry_count: 3
    retry_wait_time: 1s

crawler:
  service_tree:
    target_views: []
    max_depth: -1
    page_size: 1000
    include_statistics: true
  concurrency:
    max_workers: 10
    request_interval: 100ms

output:
  format: "json"
  file_path: "./output/service_tree_data.json"
  pretty_print: true

logging:
  level: "info"
  output: "console"
  file_path: "./logs/cmdb-crawler.log"
EOF
        log_info "默认配置已创建: config/config.yaml"
        log_warn "请修改配置文件中的CMDB连接信息后重新运行"
        exit 1
    fi
}

# 安装依赖
install_deps() {
    log_info "安装依赖包..."
    if [ ! -f "go.mod" ]; then
        log_error "go.mod文件不存在，请确保在项目根目录运行"
        exit 1
    fi
    
    go mod tidy
    if [ $? -eq 0 ]; then
        log_info "依赖安装完成"
    else
        log_error "依赖安装失败"
        exit 1
    fi
}

# 构建程序
build_app() {
    log_info "构建程序..."
    
    # 创建输出和日志目录
    mkdir -p output logs
    
    # 构建
    go build -o cmdb-crawler main.go
    if [ $? -eq 0 ]; then
        log_info "构建完成: cmdb-crawler"
    else
        log_error "构建失败"
        exit 1
    fi
}

# 运行程序
run_crawler() {
    log_header "=== CMDB服务树数据爬取工具 ==="
    echo
    
    # 解析命令行参数
    args="$@"
    if [ -z "$args" ]; then
        args="crawl"
    fi
    
    log_info "执行命令: ./cmdb-crawler $args"
    echo
    
    # 运行程序
    ./cmdb-crawler $args
    
    result=$?
    echo
    
    if [ $result -eq 0 ]; then
        log_info "爬取完成！"
        
        # 显示输出文件
        if [ -d "output" ] && [ "$(ls -A output)" ]; then
            log_info "输出文件:"
            ls -la output/
        fi
    else
        log_error "爬取失败，退出码: $result"
        exit $result
    fi
}

# 显示帮助信息
show_help() {
    cat << EOF
CMDB服务树数据爬取工具 - 快速运行脚本

用法:
  ./scripts/run.sh [选项]

选项:
  help                  显示此帮助信息
  build                 只构建程序，不运行
  crawl                 爬取所有服务树（默认）
  crawl --views "名称"  爬取指定服务树
  crawl --format csv    输出为CSV格式
  crawl --summary-only  只输出摘要信息
  crawl --help          显示爬取命令帮助

示例:
  ./scripts/run.sh                                    # 爬取所有服务树
  ./scripts/run.sh crawl --format yaml               # 输出YAML格式
  ./scripts/run.sh crawl --views "产品服务树"        # 爬取指定服务树
  ./scripts/run.sh crawl --max-depth 3 --verbose     # 限制深度并显示详细日志

配置文件: config/config.yaml
输出目录: output/
日志目录: logs/
EOF
}

# 主函数
main() {
    # 检查是否在项目根目录
    if [ ! -f "main.go" ]; then
        log_error "请在项目根目录运行此脚本"
        exit 1
    fi
    
    # 处理帮助命令
    if [ "$1" = "help" ] || [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
        show_help
        exit 0
    fi
    
    # 处理只构建命令
    if [ "$1" = "build" ]; then
        check_go
        install_deps
        build_app
        log_info "构建完成，可以直接运行: ./cmdb-crawler crawl"
        exit 0
    fi
    
    # 完整流程
    log_header "初始化环境..."
    check_go
    check_config
    install_deps
    build_app
    
    echo
    log_header "开始爬取..."
    run_crawler "$@"
}

# 执行主函数
main "$@" 