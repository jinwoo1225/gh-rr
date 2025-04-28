package pullrequest

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cli/go-gh/v2"

	"github.com/jinwoo1225/gh-rr/internal/model"
)

type FetchOption string

const (
	StateOpen         FetchOption = "--state=open"
	AuthorMe          FetchOption = "--author=@me"
	ReviewRequestedMe FetchOption = "--review-requested=@me"
	DraftFalse        FetchOption = "--draft=false"
	DraftTrue         FetchOption = "--draft=true"
	ArchivedFalse     FetchOption = "--archived=false"
	SortCreated       FetchOption = "--sort=created"
	InvolvesMe        FetchOption = "--involves=@me"
)

const (
	outputJSONFormat = "--json=author,title,url,repository,createdAt,updatedAt,commentsCount,number"
)

type rawGithubPullRequestIssueResponse struct {
	Author struct {
		Slug string `json:"login"`
	} `json:"author"`
	CommentsCount int       `json:"commentsCount"`
	CreatedAt     time.Time `json:"createdAt"`
	Number        int       `json:"number"`
	Repository    struct {
		NameWithOwner string `json:"nameWithOwner"`
	} `json:"repository"`
	Title     string    `json:"title"`
	UpdatedAt time.Time `json:"updatedAt"`
	Url       string    `json:"url"`
}

func Fetch(ctx context.Context, options ...FetchOption) ([]*model.GithubPullRequest, error) {
	optionsStr := make([]string, 0, len(options)+3)
	optionsStr = append(optionsStr, "search", "prs", outputJSONFormat)

	for _, option := range options {
		optionsStr = append(optionsStr, string(option))
	}

	stdout, stderr, err := gh.ExecContext(ctx, optionsStr...)
	if err != nil {
		var errMsg string
		if stderr.Len() > 0 {
			errMsg = stderr.String()
		}
		return nil, fmt.Errorf("fetching pull requests: %w: %s", err, errMsg)
	}

	decoder := json.NewDecoder(&stdout)
	var rawGithubPullRequestIssueResponses []*rawGithubPullRequestIssueResponse
	if err := decoder.Decode(&rawGithubPullRequestIssueResponses); err != nil {
		return nil, fmt.Errorf("parsing pull requests: %w", err)
	}

	pullRequests := make([]*model.GithubPullRequest, 0, len(rawGithubPullRequestIssueResponses))
	for _, rawPullRequestIssueResponse := range rawGithubPullRequestIssueResponses {
		pullRequests = append(pullRequests, &model.GithubPullRequest{
			PrNumber:                rawPullRequestIssueResponse.Number,
			RepositoryNameWithOwner: rawPullRequestIssueResponse.Repository.NameWithOwner,
			Title:                   rawPullRequestIssueResponse.Title,
			AuthorSlug:              rawPullRequestIssueResponse.Author.Slug,
			URL:                     rawPullRequestIssueResponse.Url,
			CommentsCount:           rawPullRequestIssueResponse.CommentsCount,
			CreatedAt:               rawPullRequestIssueResponse.CreatedAt,
			UpdatedAt:               rawPullRequestIssueResponse.UpdatedAt,
		})
	}

	return pullRequests, nil
}
