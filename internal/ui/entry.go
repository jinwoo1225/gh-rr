package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/list"

	"github.com/jinwoo1225/gh-rr/internal/model"
	"github.com/jinwoo1225/gh-rr/internal/utils"
)

type Entry struct {
	RepositoryNameWithOwner string
	Title                   string
	URL                     string
	AgeStr                  string
	LastUpdatedSinceStr     string
	Author                  string
	PrNumber                int
	CommentsCount           int
}

// categoryItem wraps a category name for the category List.
// itemEntry wraps an Entry for the PR List.
type itemEntry struct{ entry Entry }

func (i itemEntry) Title() string {
	return fmt.Sprintf("%s — %s - %d", i.entry.RepositoryNameWithOwner, i.entry.Title, i.entry.PrNumber)
}
func (i itemEntry) Description() string {
	return fmt.Sprintf("Age: %s, LastUpdatedSince: %s, Author: %s, CommentCount: %d", i.entry.AgeStr, i.entry.LastUpdatedSinceStr, i.entry.Author, i.entry.CommentsCount)
}
func (i itemEntry) FilterValue() string { return i.entry.RepositoryNameWithOwner + " " + i.entry.Title }

// ItemsFromEntries converts []Entry to []list.Item.
func ItemsFromEntries(entries []Entry) []list.Item {
	items := make([]list.Item, len(entries))
	for i, e := range entries {
		items[i] = itemEntry{entry: e}
	}
	return items
}

func BuildEntries(pullRequests []*model.GithubPullRequest, now time.Time) []Entry {
	entries := make([]Entry, 0, len(pullRequests))
	for _, pullRequest := range pullRequests {
		age := now.Sub(pullRequest.CreatedAt)
		lastUpdatedSince := now.Sub(pullRequest.UpdatedAt)
		entries = append(entries, Entry{
			RepositoryNameWithOwner: pullRequest.RepositoryNameWithOwner,
			Title:                   pullRequest.Title,
			URL:                     pullRequest.URL,
			AgeStr:                  utils.HumanizeDuration(int(age.Seconds())),
			LastUpdatedSinceStr:     utils.HumanizeDuration(int(lastUpdatedSince.Seconds())),
			Author:                  pullRequest.AuthorSlug,
			PrNumber:                pullRequest.PrNumber,
			CommentsCount:           pullRequest.CommentsCount,
		})
	}
	return entries
}
