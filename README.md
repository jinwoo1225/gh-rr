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

## Clone & Checkout

By default, pressing 'Enter' on a selection opens the PR in your browser. You can also press the 'c' key to clone the repository and checkout the pull request branch locally.

Before cloning, you will be prompted to confirm the clone directory, which defaults to `~/workspace` unless overridden in a config file.

### Configuration

Create a config file at `~/.gh-rr` to customize settings:

```bash
# Example ~/.gh-rr
BASE_DIR="${HOME}/projects/github"  # Directory where repos will be cloned
```
