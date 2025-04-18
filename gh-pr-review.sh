#!/usr/bin/env bash
set -euo pipefail

: "${GITHUB_TOKEN:?Environment variable GITHUB_TOKEN must be set}"

if ! command -v jq &>/dev/null; then
  echo "Error: 'jq' is required. Please install jq." >&2
  exit 1
fi
if ! command -v fzf &>/dev/null; then
  echo "Error: 'fzf' is required. Please install fzf." >&2
  exit 1
fi

# Fetch pull requests requesting your review
response=$(curl -sS \
  -H "Accept: application/vnd.github.v3+json" \
  -H "Authorization: token $GITHUB_TOKEN" \
  "https://api.github.com/search/issues?q=is:pr+is:open+review-requested:@me&per_page=100")

# Extract repository, title, html_url
entries=$(echo "$response" | jq -r '.items[] | (.repository_url | sub("https://api.github.com/repos/"; "")) + "\t" + .title + "\t" + .html_url')

if [[ -z "$entries" ]]; then
  echo "No open pull requests requesting your review." >&2
  exit 0
fi

# Let user select with fzf
selected=$(printf "%s\n" "$entries" \
  | fzf --header='Select a pull request to open' \
        --delimiter=$'\t' \
        --with-nth=1,2 \
        --ansi)

if [[ -z "$selected" ]]; then
  exit 0
fi

# Extract URL and open
url=$(echo "$selected" | cut -f3)
if command -v xdg-open &>/dev/null; then
  xdg-open "$url"
elif command -v open &>/dev/null; then
  open "$url"
else
  echo "Please open this URL manually: $url"
fi