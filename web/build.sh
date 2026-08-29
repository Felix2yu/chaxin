#!/bin/sh
# 构建脚本：将纯静态前端（HTML/CSS/JS/图标）原样复制到 internal/web/dist，
# 供 Go 通过 go:embed 内嵌。无需 node / 打包器，保持轻量。
set -e

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
DEST="$SCRIPT_DIR/../internal/web/dist"

rm -rf "$DEST"
mkdir -p "$DEST"

cp "$SCRIPT_DIR/index.html" "$DEST/index.html"
cp "$SCRIPT_DIR/styles.css" "$DEST/styles.css"
cp -r "$SCRIPT_DIR/js" "$DEST/js"
cp -r "$SCRIPT_DIR/icons" "$DEST/icons"

echo "前端已构建到 $DEST"
