#!/bin/sh
# 构建脚本：将纯静态前端（HTML/CSS/JS/图标/清单/SW）复制到 internal/web/dist，
# 供 Go 通过 go:embed 内嵌。无需 node / 打包器，保持轻量。
set -e

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
DEST="$SCRIPT_DIR/../internal/web/dist"

rm -rf "$DEST"
mkdir -p "$DEST"

cp "$SCRIPT_DIR/index.html" "$DEST/index.html"
cp "$SCRIPT_DIR/offline.html" "$DEST/offline.html"
cp "$SCRIPT_DIR/styles.css" "$DEST/styles.css"
cp "$SCRIPT_DIR/manifest.webmanifest" "$DEST/manifest.webmanifest"
cp -r "$SCRIPT_DIR/js" "$DEST/js"

# 图标契约（见 ../../PWA规范.md）：统一放在站点根，URL 为 /icon-192.png 等
cp "$SCRIPT_DIR"/icons/*.png "$DEST"/
cp "$SCRIPT_DIR"/icons/*.ico "$DEST"/
cp "$SCRIPT_DIR"/icons/*.svg "$DEST"/

# Service Worker：把版本号写进缓存名，版本一变即触发旧缓存淘汰
VERSION=$(git -C "$SCRIPT_DIR" rev-parse --short HEAD 2>/dev/null || date +%Y%m%d%H%M%S)
sed "s/__BUILD_VERSION__/${VERSION}/g" "$SCRIPT_DIR/sw.js" > "$DEST/sw.js"

echo "前端已构建到 $DEST"
