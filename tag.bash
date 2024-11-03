#!/bin/bash

SCRIPT_DIR=$(cd $(dirname $0); pwd)

version="$(cat "$SCRIPT_DIR/.version")"

# y/nでversionを確認して、yesならtagをうってpushする

# バージョン確認メッセージ
echo "Current version is: $version"
read -p "Do you want to tag and push this version? (y/n): " confirm

# 入力が 'y' または 'Y' の場合のみタグ付けしてプッシュ
if [[ "$confirm" == "y" || "$confirm" == "Y" ]]; then
    # Gitタグを作成し、リモートにプッシュ
    git tag "$version"
    git push origin "$version"
    echo "Tag v$version has been pushed to remote repository."
else
    echo "Tagging and push canceled."
fi
