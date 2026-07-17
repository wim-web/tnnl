#!/bin/bash

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
version="${1:-}"

if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    printf 'usage: ./tag.bash vX.Y.Z\n' >&2
    exit 2
fi

printf 'Current version is: %s\n' "$version"
confirm=""
read -r -p "Do you want to tag and push this version? (y/n): " confirm || true

if [[ "$confirm" == "y" || "$confirm" == "Y" ]]; then
    git -C "$script_dir" tag "$version"
    git -C "$script_dir" push origin "$version"
    printf 'Tag %s has been pushed to remote repository.\n' "$version"
else
    printf 'Tagging and push canceled.\n'
fi
