#!/usr/bin/env bash
# 吃心前端构建脚本 —— 纯静态资源原样复制到 internal/web/dist 供 go:embed 内嵌。
#
# 产物：internal/web/dist（无需 node / 打包器，保持轻量）
# 前置：无
#
# 用法：
#   bash web/build.sh
#
# 约定见 Felix2yu/.github/docs/build-script.md（本脚本无 Go/Node，适用第二、四节）
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

step() { printf '\033[1;34m==>\033[0m %s\n' "$1"; }
die() { printf '\033[1;31m错误：\033[0m %s\n' "$1" >&2; exit 1; }

WEB="$ROOT/web"
DEST="$ROOT/internal/web/dist"

step "[1/3] 清理产物目录"
rm -rf "$DEST"
mkdir -p "$DEST"

step "[2/3] 复制静态资源"
cp "$WEB/index.html" "$DEST/index.html"
cp "$WEB/offline.html" "$DEST/offline.html"
cp "$WEB/styles.css" "$DEST/styles.css"
cp "$WEB/manifest.webmanifest" "$DEST/manifest.webmanifest"
cp -r "$WEB/js" "$DEST/js"

# 图标契约（见 PWA.md）：统一放在站点根，URL 为 /icon-192.png 等
cp "$WEB"/icons/*.png "$DEST"/
cp "$WEB"/icons/*.ico "$DEST"/
cp "$WEB"/icons/*.svg "$DEST"/

# go:embed 要求目录存在，占位文件保证「没跑过本脚本的 clone」也能编译。
touch "$DEST/.gitkeep"

step "[3/3] 注入版本号到 Service Worker"
# 版本号进缓存名，版本一变即触发旧缓存淘汰
VERSION=$(git -C "$ROOT" rev-parse --short HEAD 2>/dev/null || date +%Y%m%d%H%M%S)
sed "s/__BUILD_VERSION__/${VERSION}/g" "$WEB/sw.js" > "$DEST/sw.js"

printf '\n完成：%s（版本 %s）\n' "$DEST" "$VERSION"
