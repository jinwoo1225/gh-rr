package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/go-github/v71/github"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

// Entry represents a pull request in the UI.
type Entry struct {
	Repo   string
	Title  string
	URL    string
	AgeStr string
}

// categoryItem wraps a category name for the category list.
// itemEntry wraps an Entry for the PR list.
type itemEntry struct{ entry Entry }

func (i itemEntry) Title() string       { return fmt.Sprintf("%s — %s", i.entry.Repo, i.entry.Title) }
func (i itemEntry) Description() string { return fmt.Sprintf("Age: %s", i.entry.AgeStr) }
func (i itemEntry) FilterValue() string { return i.entry.Repo + " " + i.entry.Title }

// itemsFromEntries converts []Entry to []list.Item.
func itemsFromEntries(entries []Entry) []list.Item {
	items := make([]list.Item, len(entries))
	for i, e := range entries {
		items[i] = itemEntry{entry: e}
	}
	return items
}

// model is the main Bubble Tea model.
type model struct {
	categories []string
	catIndex   int
	entries    [][]Entry
	list       list.Model
	clone      bool
	quit       bool
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left":
			m.catIndex = (m.catIndex + len(m.categories) - 1) % len(m.categories)
			m.list.SetItems(itemsFromEntries(m.entries[m.catIndex]))
		case "right":
			m.catIndex = (m.catIndex + 1) % len(m.categories)
			m.list.SetItems(itemsFromEntries(m.entries[m.catIndex]))
		case "enter":
			m.clone = false
			return m, tea.Quit
		case "c":
			m.clone = true
			return m, tea.Quit
		case "q", "ctrl+c":
			m.quit = true
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}
	m.list.Title = m.categories[m.catIndex]

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)

	return m, cmd
}

func (m model) View() string {
	sb := strings.Builder{}

	for _, cat := range m.categories {
		if cat == m.categories[m.catIndex] {
			sb.WriteString(" [ ")
			sb.WriteString(cat)
			sb.WriteString(" ]")
		} else {
			sb.WriteString("   ")
			sb.WriteString(cat)
			sb.WriteString("  ")
		}
	}

	sb.WriteString("\n")
	sb.WriteString(m.list.View())

	return docStyle.Render(sb.String())
}

func main() {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "GITHUB_TOKEN must be set")
		os.Exit(1)
	}
	// Initialize GitHub client
	ctx := context.Background()
	client := github.NewClient(http.DefaultClient).WithAuthToken(token)

	wg := sync.WaitGroup{}

	var (
		reviewRequested  []*github.Issue
		myPullRequest    []*github.Issue
		draftPullRequest []*github.Issue
	)
	start := time.Now()

	wg.Add(1)
	go func() {
		defer wg.Done()
		rev, err := fetchIssues(ctx, client, "is:pr is:open review-requested:@me draft:false archived:false")
		if err != nil {
			log.Println("Error While Fetching ReviewRequest", errors.WithStack(err))
			return
		}
		log.Printf("Fetched review requested PRs in %1.4fs\n", time.Since(start).Seconds())

		reviewRequested = rev
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		mine, err := fetchIssues(ctx, client, "is:pr is:open author:@me draft:false archived:false")
		if err != nil {
			log.Println("Error While Fetching My PRs", errors.WithStack(err))
			return
		}
		log.Printf("Fetched my PRs in %1.4fs\n", time.Since(start).Seconds())

		myPullRequest = mine
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		draft, err := fetchIssues(ctx, client, "is:pr is:open author:@me draft:true archived:false")
		if err != nil {
			log.Println("Error While Fetching Draft PRs", errors.WithStack(err))
			return
		}
		log.Printf("Fetched draft PRs in %1.4fs\n", time.Since(start).Seconds())

		draftPullRequest = draft
	}()

	wg.Wait()
	log.Println("Fetching completed.")

	now := time.Now()
	revEntries := buildEntries(reviewRequested, now)
	myEntries := buildEntries(myPullRequest, now)
	draftEntries := buildEntries(draftPullRequest, now)
	// Initialize TUI with categories and PR entries
	categories := []string{"Review Requests", "My PRs", "Draft PRs"}
	entries2d := [][]Entry{revEntries, myEntries, draftEntries}
	// Prepare list Model with initial category
	initialItems := itemsFromEntries(entries2d[0])
	// Set dimensions: width 80, height max 20 or len(entries)+4
	h := len(initialItems) + 4
	if h > 20 {
		h = 20
	}
	// Customize list delegate for clearer selection
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(lipgloss.Color("205")).Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(lipgloss.Color("240"))
	delegate.ShortHelpFunc = func() []key.Binding {
		return []key.Binding{
			keys.Left, keys.Right, keys.Enter, keys.Checkout, keys.Quit,
		}
	}
	delegate.FullHelpFunc = func() [][]key.Binding {
		return [][]key.Binding{
			{keys.Left, keys.Right},
			{keys.Enter, keys.Checkout},
			{keys.Quit},
		}
	}

	l := list.New(initialItems, delegate, 0, 0)
	l.SetShowHelp(true)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)

	// Run Bubble Tea program
	m0 := model{
		categories: categories,
		catIndex:   0,
		entries:    entries2d,
		list:       l,
	}
	finalModel, err := tea.NewProgram(m0, tea.WithAltScreen()).Run()
	if err != nil {
		fatal(err)
	}
	m := finalModel.(model)

	if m.quit {
		return
	}

	selEntry := m.entries[m.catIndex][m.list.Index()]
	baseDir := getBaseDir()
	if m.clone {
		cloneAndCheckout(selEntry.Repo, selEntry.URL, baseDir, token)
	} else {
		openURL(selEntry.URL)
	}
}

// fetchIssues uses go-github to search PRs.
func fetchIssues(ctx context.Context, client *github.Client, query string) ([]*github.Issue, error) {
	opts := &github.SearchOptions{Sort: "created", Order: "asc", ListOptions: github.ListOptions{PerPage: 50}}
	res, _, err := client.Search.Issues(ctx, query, opts)
	if err != nil {
		return nil, err
	}
	return res.Issues, nil
}

// buildEntries converts GitHub issues to UI entries.
func buildEntries(items []*github.Issue, now time.Time) []Entry {
	var es []Entry
	for _, it := range items {
		age := now.Sub(it.GetCreatedAt().Time)
		es = append(es, Entry{
			Repo:   strings.TrimPrefix(it.GetRepositoryURL(), "https://api.github.com/repos/"),
			Title:  it.GetTitle(),
			URL:    it.GetHTMLURL(),
			AgeStr: humanize(int(age.Seconds())),
		})
	}
	return es
}

// humanize formats seconds to a human-friendly string.
func humanize(s int) string {
	switch {
	case s < 60:
		return fmt.Sprintf("%ds", s)
	case s < 3600:
		return fmt.Sprintf("%dm", s/60)
	case s < 86400:
		return fmt.Sprintf("%dh", s/3600)
	case s < 604800:
		return fmt.Sprintf("%dd", s/86400)
	default:
		return fmt.Sprintf("%dw", s/604800)
	}
}

// getBaseDir returns the directory to clone PRs into.
func getBaseDir() string {
	if b := os.Getenv("BASE_DIR"); b != "" {
		return expandHome(b)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "workspace")
}

// expandHome expands a leading '~' in a path to the user home directory.
func expandHome(p string) string {
	if strings.HasPrefix(p, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return p
		}
		return filepath.Join(home, p[1:])
	}
	return p
}

// cloneAndCheckout clones the repo and checks out the PR branch.
func cloneAndCheckout(repo, urlStr, baseDir, token string) {
	pr := path.Base(strings.TrimRight(urlStr, "/"))
	dir := filepath.Join(baseDir, repo)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("Clone %s into %s? [Y/n]: ", repo, dir)
		resp, _ := reader.ReadString('\n')
		resp = strings.TrimSpace(resp)
		if resp != "" && strings.ToLower(resp) == "n" {
			fmt.Println("Skipping clone.")
			return
		}
		fmt.Printf("Cloning %s into %s\n", repo, dir)
		if err := os.MkdirAll(filepath.Dir(dir), 0755); err != nil {
			fatal(err)
		}
		cmd := exec.Command("git", "clone", fmt.Sprintf("https://github.com/%s.git", repo), dir)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fatal(err)
		}
	}
	// Checkout PR
	if ghPath, err := exec.LookPath("gh"); err == nil {
		cmd := exec.Command(ghPath, "pr", "checkout", pr)
		cmd.Dir = dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fatal(err)
		}
	} else {
		// fallback: fetch and checkout
		api := fmt.Sprintf("https://api.github.com/repos/%s/pulls/%s", repo, pr)
		req, _ := http.NewRequest("GET", api, nil)
		req.Header.Set("Authorization", "token "+token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil || resp.StatusCode != 200 {
			fatal(fmt.Errorf("failed to fetch PR metadata"))
		}
		defer resp.Body.Close()
		var prInfo struct {
			Head struct {
				Ref  string `json:"ref"`
				Repo struct {
					CloneURL string `json:"clone_url"`
					FullName string `json:"full_name"`
				} `json:"repo"`
			} `json:"head"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&prInfo); err != nil {
			fatal(err)
		}
		head := prInfo.Head
		if head.Repo.FullName == repo {
			exec.Command("git", "fetch", "origin", fmt.Sprintf("pull/%s/head", pr)).Run()
			exec.Command("git", "checkout", "-b", head.Ref, "FETCH_HEAD").Run()
		} else {
			remote := fmt.Sprintf("pr-%s", pr)
			exec.Command("git", "remote", "add", remote, head.Repo.CloneURL).Run()
			exec.Command("git", "fetch", remote, head.Ref).Run()
			exec.Command("git", "checkout", "-b", head.Ref, fmt.Sprintf("%s/%s", remote, head.Ref)).Run()
		}
	}
	fmt.Printf("Checked out PR #%s in %s\n", pr, dir)
	// change working directory and launch a shell in the repo
	if err := os.Chdir(dir); err != nil {
		fatal(err)
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	// replace current process with a shell in the checked-out repo
	if err := syscall.Exec(shell, []string{shell}, os.Environ()); err != nil {
		fatal(err)
	}
}

// openURL opens the given URL in the default browser.
func openURL(u string) {
	if cmd, err := exec.LookPath("xdg-open"); err == nil {
		exec.Command(cmd, u).Start()
	} else if cmd, err := exec.LookPath("open"); err == nil {
		exec.Command(cmd, u).Start()
	} else {
		fmt.Printf("Please open this URL manually: %s\n", u)
	}
}

// fatal prints the error and exits.
func fatal(err error) {
	fmt.Fprintln(os.Stderr, "Error:", err)
	os.Exit(1)
}

type keyMap struct {
	Left     key.Binding
	Right    key.Binding
	Enter    key.Binding
	Checkout key.Binding
	Quit     key.Binding
}

var keys = keyMap{
	Left: key.NewBinding(
		key.WithKeys("left"),
		key.WithHelp("←", "prev category"),
	),
	Right: key.NewBinding(
		key.WithKeys("right"),
		key.WithHelp("→", "next category"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("↵", "open PR in browser"),
	),
	Checkout: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "checkout PR"),
	),
}
