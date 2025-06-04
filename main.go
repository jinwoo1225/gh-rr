package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jinwoo1225/gh-rr/internal/model"
	"github.com/jinwoo1225/gh-rr/internal/pullrequest"
	"github.com/jinwoo1225/gh-rr/internal/ui"
	"github.com/jinwoo1225/gh-rr/internal/utils"
)

const clearConsoleANSIEscapeCode = "\033c"

func main() {
	ctx := context.Background()

	var (
		reviewRequestedPullRequests []*model.GithubPullRequest
		myPullRequests              []*model.GithubPullRequest
		draftPullRequests           []*model.GithubPullRequest
		involvedPullRequest         []*model.GithubPullRequest

		err error
	)

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		reviewRequestedPullRequests, err = pullrequest.Fetch(
			ctx,
			pullrequest.StateOpen,
			pullrequest.ReviewRequestedMe,
			pullrequest.DraftFalse,
			pullrequest.ArchivedFalse,
			pullrequest.SortCreated,
		)
		if err != nil {
			log.Println(errors.Wrap(err, "fetching review requested"))
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		myPullRequests, err = pullrequest.Fetch(
			ctx,
			pullrequest.StateOpen,
			pullrequest.AuthorMe,
			pullrequest.DraftFalse,
			pullrequest.ArchivedFalse,
			pullrequest.SortCreated,
		)
		if err != nil {
			log.Println(errors.Wrap(err, "fetching my pull requests"))
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		draftPullRequests, err = pullrequest.Fetch(
			ctx,
			pullrequest.StateOpen,
			pullrequest.DraftTrue,
			pullrequest.InvolvesMe,
			pullrequest.ArchivedFalse,
			pullrequest.SortCreated,
		)
		if err != nil {
			log.Println(errors.Wrap(err, "fetching draft pull requests"))
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		involvedPullRequest, err = pullrequest.Fetch(
			ctx,
			pullrequest.StateOpen,
			pullrequest.InvolvesMe,
			pullrequest.DraftFalse,
			pullrequest.ArchivedFalse,
			pullrequest.SortCreated,
		)
		if err != nil {
			log.Println(errors.Wrap(err, "fetching involved pull requests"))
		}
	}()

	wg.Wait()

	now := time.Now()
	revEntries := ui.BuildEntries(reviewRequestedPullRequests, now)
	myEntries := ui.BuildEntries(myPullRequests, now)
	draftEntries := ui.BuildEntries(draftPullRequests, now)
	involvedEntries := ui.BuildEntries(involvedPullRequest, now)

	// Initialize TUI with categories and PR entries
	categories := []string{"Review Requests", "My PRs", "Draft PRs", "Involved Open PRs"}
	entries2d := [][]ui.Entry{revEntries, myEntries, draftEntries, involvedEntries}

	// Prepare list Model with initial category
	initialItems := ui.ItemsFromEntries(entries2d[0])

	// Customize list delegate for clearer selection
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(lipgloss.Color("205")).Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(lipgloss.Color("240"))

	l := list.New(initialItems, &delegate, 0, 0)
	l.SetShowHelp(true)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)

	// Run Bubble Tea program
	listModel := &ui.ListModel{
		Categories: categories,
		Entries:    entries2d,
		List:       l,
	}
	delegate.ShortHelpFunc = func() []key.Binding {
		now := time.Now()
		remaining := int(listModel.NextRefresh().Sub(now).Seconds())
		if remaining < 0 {
			remaining = 0
		}
		timer := utils.HumanizeDuration(remaining)
		return []key.Binding{
			ui.Keys.Left, ui.Keys.Right, ui.Keys.Enter,
			key.NewBinding(
				key.WithKeys("r"),
				key.WithHelp("r", fmt.Sprintf("refresh (in %s)", timer)),
			),
			ui.Keys.Checkout, ui.Keys.Quit,
		}
	}
	delegate.FullHelpFunc = func() [][]key.Binding {
		now := time.Now()
		remaining := int(listModel.NextRefresh().Sub(now).Seconds())
		if remaining < 0 {
			remaining = 0
		}
		timer := utils.HumanizeDuration(remaining)
		rBinding := key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", fmt.Sprintf("refresh (in %s)", timer)),
		)
		return [][]key.Binding{
			{ui.Keys.Left, ui.Keys.Right},
			{ui.Keys.Enter, rBinding, ui.Keys.Checkout},
			{ui.Keys.Quit},
		}
	}

	finalModel, err := tea.NewProgram(listModel, tea.WithAltScreen()).Run()
	if err != nil {
		log.Panicln(errors.Wrap(err, "running tea program"))
	}
	m := finalModel.(*ui.ListModel)

	if m.IsQuit() {
		return
	}

	fmt.Print(clearConsoleANSIEscapeCode)

	selectedEntry := m.Entries[m.CategoryIndex][m.List.Index()]
	baseDir := utils.GetBaseDir()
	if m.IsClone() {
		utils.CloneAndCheckout(ctx, selectedEntry.RepositoryNameWithOwner, selectedEntry.PrNumber, baseDir)
	} else {
		utils.OpenURL(selectedEntry.URL)
	}
}
