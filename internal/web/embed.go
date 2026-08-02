package web

import "embed"

// dist 为前端构建产物（由 web/ 目录构建后输出到此），由 Dockerfile/构建脚本负责生成。
//go:embed all:dist
var distFS embed.FS
