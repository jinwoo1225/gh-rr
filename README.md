# gh-rr: GitHub CLI extension for PR review requests

> Open your GitHub review-requested pull requests in a neat fzf-powered menu.

## Installation

Using the official GH CLI extension mechanism:

```bash
gh extension install jinwoo1225/gh-rr
```

Or install manually by cloning this repo and copying the script:

```bash
git clone https://github.com/jinwoo1225/gh-rr.git
cd gh-rr
mkdir -p ~/.local/share/gh/extensions/rr
cp bin/gh-rr ~/.local/share/gh/extensions/rr/gh-rr
chmod +x ~/.local/share/gh/extensions/rr/gh-rr
```

Ensure you have `jq`, `fzf`, `curl` installed, and the `GITHUB_TOKEN` env var set.

## Usage

```bash
# List and open PRs requesting your review
gh rr
```

For help:

```bash
gh rr --help
```
