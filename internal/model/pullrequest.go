package model

import (
	"time"
)

type GithubPullRequest struct {
	PrNumber                int
	RepositoryNameWithOwner string
	Title                   string
	AuthorSlug              string
	URL                     string
	CommentsCount           int
	CreatedAt               time.Time
	UpdatedAt               time.Time
}
