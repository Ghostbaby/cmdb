#!/bin/bash

# CMDB爬取工具构建脚本

set -e

# 项目信息
PROJECT_NAME="cmdb-crawler"
VERSION=${VERSION:-"1.0.0"}
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# 构建目录
BUILD_DIR="build"
DIST_DIR="dist"

# 清理旧的构建文件
echo "清理构建目录..."
rm -rf ${BUILD_DIR}
rm -rf ${DIST_DIR}
mkdir -p ${BUILD_DIR}
mkdir -p ${DIST_DIR}

# 设置构建参数
LDFLAGS="-s -w"
LDFLAGS="${LDFLAGS} -X main.Version=${VERSION}"
LDFLAGS="${LDFLAGS} -X main.BuildTime=${BUILD_TIME}"
LDFLAGS="${LDFLAGS} -X main.GitCommit=${GIT_COMMIT}"

echo "开始构建 ${PROJECT_NAME} v${VERSION}..."
echo "构建时间: ${BUILD_TIME}"
echo "Git提交: ${GIT_COMMIT}"

# 构建不同平台的二进制文件
platforms=(
    "linux/amd64"
    "linux/arm64" 
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

for platform in "${platforms[@]}"; do
    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}
    
    output_name="${PROJECT_NAME}-${GOOS}-${GOARCH}"
    if [ $GOOS = "windows" ]; then
        output_name+='.exe'
    fi
    
    echo "构建 ${GOOS}/${GOARCH}..."
    
    env GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags="${LDFLAGS}" \
        -o ${BUILD_DIR}/${output_name} \
        main.go
    
    if [ $? -ne 0 ]; then
        echo "构建 ${GOOS}/${GOARCH} 失败"
        exit 1
    fi
    
    # 创建发布包
    archive_name="${PROJECT_NAME}-${VERSION}-${GOOS}-${GOARCH}"
    
    # 复制必要文件
    mkdir -p ${BUILD_DIR}/${archive_name}
    cp ${BUILD_DIR}/${output_name} ${BUILD_DIR}/${archive_name}/
    cp README.md ${BUILD_DIR}/${archive_name}/
    cp -r config ${BUILD_DIR}/${archive_name}/
    cp -r examples ${BUILD_DIR}/${archive_name}/
    
    # 创建启动脚本
    if [ $GOOS = "windows" ]; then
        cat > ${BUILD_DIR}/${archive_name}/run.bat << 'EOF'
@echo off
echo Starting CMDB Crawler...
cmdb-crawler-windows-amd64.exe crawl
pause
EOF
    else
        cat > ${BUILD_DIR}/${archive_name}/run.sh << 'EOF'
#!/bin/bash
echo "Starting CMDB Crawler..."
./cmdb-crawler-* crawl
EOF
        chmod +x ${BUILD_DIR}/${archive_name}/run.sh
    fi
    
    # 打包
    cd ${BUILD_DIR}
    if [ $GOOS = "windows" ]; then
        zip -r ../${DIST_DIR}/${archive_name}.zip ${archive_name}
    else
        tar -czf ../${DIST_DIR}/${archive_name}.tar.gz ${archive_name}
    fi
    cd ..
    
    rm -rf ${BUILD_DIR}/${archive_name}
    
    echo "✓ ${GOOS}/${GOARCH} 构建完成"
done

# 构建Docker镜像
if command -v docker &> /dev/null; then
    echo "构建Docker镜像..."
    docker build -t ${PROJECT_NAME}:${VERSION} .
    docker tag ${PROJECT_NAME}:${VERSION} ${PROJECT_NAME}:latest
    echo "✓ Docker镜像构建完成"
fi

echo ""
echo "🎉 构建完成！"
echo ""
echo "二进制文件:"
ls -la ${BUILD_DIR}/
echo ""
echo "发布包:"
ls -la ${DIST_DIR}/
echo ""
echo "使用方法:"
echo "  Linux/macOS: tar -xzf ${DIST_DIR}/${PROJECT_NAME}-${VERSION}-linux-amd64.tar.gz"
echo "  Windows: 解压 ${DIST_DIR}/${PROJECT_NAME}-${VERSION}-windows-amd64.zip"
echo "  Docker: docker run -v \$(pwd)/config:/app/config ${PROJECT_NAME}:${VERSION} crawl" 