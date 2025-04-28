# gh-rr: GitHub CLI extension for PR review requests

> Open your GitHub review-requested pull requests in a neat terminal UI.

## Installation

Using the official GH CLI extension mechanism:

```bash
gh extension install jinwoo1225/gh-rr
```

To build the Go binary, run:

```bash
go mod tidy
go build -o bin/gh-rr main.go
chmod +x bin/gh-rr
```

To install as a GitHub CLI extension:

```bash
mkdir -p ~/.local/share/gh/extensions/rr
cp bin/gh-rr ~/.local/share/gh/extensions/rr/gh-rr
```

Ensure you have the [GitHub CLI (`gh`)](https://cli.github.com) installed, and the `GITHUB_TOKEN` env var set.

## Usage

```bash
# List and open PRs requesting your review
gh rr
```

Controls:
  - ←/→: switch between Review Requests / My PRs / Draft PRs
  - ↑/↓: navigate PR list
  - Enter: open selected PR in browser
  - c: clone & checkout selected PR locally
  - q: quit TUI

## Clone & Checkout

By default, pressing 'Enter' on a selection opens the PR in your browser. You can also press the 'c' key to clone the repository and checkout the pull request branch locally.

When checking out a PR locally with 'c', the script will use GitHub CLI's `gh pr checkout <pr_number>` if available. Otherwise, it will fetch the PR metadata and check out the actual head branch name:
- If the PR is from the same repository, it fetches `pull/<pr_number>/head` and checks out a local branch named after the head branch.
- If the PR is from a fork, it adds a remote for the forked repository, fetches the head branch, and checks out a local branch tracking the fork's branch.

Before cloning, you will be prompted to confirm the clone directory, which defaults to `~/workspace` unless overridden by setting the `BASE_DIR` environment variable.

### Configuration

You can customize the clone directory by setting the `BASE_DIR` environment variable:

```bash
export BASE_DIR="${HOME}/projects/github"  # Directory where repos will be cloned
```

If `BASE_DIR` is not set, the default is `~/workspace`.

## Contribution

Requirements:

  - Go 1.24+ installed in your PATH
  - Go modules enabled (GO111MODULE=on)
  - git


