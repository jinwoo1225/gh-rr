package ui

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jinwoo1225/gh-rr/internal/model"
	"github.com/jinwoo1225/gh-rr/internal/pullrequest"
	"github.com/jinwoo1225/gh-rr/internal/utils"
)

var (
	docStyle = lipgloss.NewStyle().Margin(1, 2)

	tabStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color("240"))

	selectedTabStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Bold(true).
				Foreground(lipgloss.Color("205")).                          // 더 밝은 색상
				Background(lipgloss.Color("236")).                          // 배경색 추가
				Border(lipgloss.NormalBorder(), false, false, true, false). // 하단 테두리 추가
				BorderForeground(lipgloss.Color("205"))                     // 테두리 색상

)

type ListModel struct {
	Categories    []string
	CategoryIndex int
	Entries       [][]Entry
	List          list.Model
	clone         bool
	quit          bool
	nextRefresh   time.Time
}

type refreshedMsg struct {
	entries [][]Entry
}

// tickMsg signals the passing of time for auto-refresh countdown.
type tickMsg time.Time

// tickCmd schedules the next tickMsg after one second.
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *ListModel) refreshCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var (
			reviewRequested []*model.GithubPullRequest
			myPullRequests  []*model.GithubPullRequest
			draftPulls      []*model.GithubPullRequest
			involvedPulls   []*model.GithubPullRequest
			err             error
			wg              sync.WaitGroup
		)
		wg.Add(1)
		go func() {
			defer wg.Done()
			reviewRequested, err = pullrequest.Fetch(ctx,
				pullrequest.StateOpen, pullrequest.ReviewRequestedMe,
				pullrequest.DraftFalse, pullrequest.ArchivedFalse,
				pullrequest.SortCreated)
			if err != nil {
				log.Println("refresh: review requested error:", err)
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			myPullRequests, err = pullrequest.Fetch(ctx,
				pullrequest.StateOpen, pullrequest.AuthorMe,
				pullrequest.DraftFalse, pullrequest.ArchivedFalse,
				pullrequest.SortCreated)
			if err != nil {
				log.Println("refresh: my pull requests error:", err)
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			draftPulls, err = pullrequest.Fetch(ctx,
				pullrequest.StateOpen, pullrequest.DraftTrue,
				pullrequest.InvolvesMe, pullrequest.ArchivedFalse,
				pullrequest.SortCreated)
			if err != nil {
				log.Println("refresh: draft pull requests error:", err)
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			involvedPulls, err = pullrequest.Fetch(ctx,
				pullrequest.StateOpen, pullrequest.InvolvesMe,
				pullrequest.DraftFalse, pullrequest.ArchivedFalse,
				pullrequest.SortCreated)
			if err != nil {
				log.Println("refresh: involved pull requests error:", err)
			}
		}()
		wg.Wait()

		now := time.Now()
		return refreshedMsg{
			entries: [][]Entry{
				BuildEntries(reviewRequested, now),
				BuildEntries(myPullRequests, now),
				BuildEntries(draftPulls, now),
				BuildEntries(involvedPulls, now),
			},
		}
	}
}

func (m *ListModel) Init() tea.Cmd {
	m.nextRefresh = time.Now().Add(time.Minute)
	return tickCmd()
}

func (m *ListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		now := time.Time(msg)
		if now.After(m.nextRefresh) || now.Equal(m.nextRefresh) {
			m.nextRefresh = now.Add(time.Minute)
			return m, tea.Batch(m.refreshCmd(), tickCmd())
		}
		return m, tickCmd()
	case refreshedMsg:
		m.Entries = msg.entries
		m.List.SetItems(ItemsFromEntries(m.Entries[m.CategoryIndex]))
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "left":
			m.CategoryIndex = (m.CategoryIndex + len(m.Categories) - 1) % len(m.Categories)
			m.List.SetItems(ItemsFromEntries(m.Entries[m.CategoryIndex]))
		case "right":
			m.CategoryIndex = (m.CategoryIndex + 1) % len(m.Categories)
			m.List.SetItems(ItemsFromEntries(m.Entries[m.CategoryIndex]))
		case "enter":
			entry := m.Entries[m.CategoryIndex][m.List.Index()]
			utils.OpenURL(entry.URL)
			m.clone = false
			return m, nil
		case "c":
			m.clone = true
			return m, tea.Quit
		case "r":
			now := time.Now()
			m.nextRefresh = now.Add(time.Minute)
			return m, tea.Batch(m.refreshCmd(), tickCmd())
		case "q", "ctrl+c":
			m.quit = true
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.List.SetSize(msg.Width-h, msg.Height-v-3)
	}
	m.List.Title = ""

	var cmd tea.Cmd
	m.List, cmd = m.List.Update(msg)

	return m, cmd
}

func (m *ListModel) View() string {
	sb := strings.Builder{}

	// 탭 렌더링 개선
	var tabsView []string
	for i, cat := range m.Categories {
		if i == m.CategoryIndex {
			tabsView = append(tabsView, selectedTabStyle.Render(cat))
		} else {
			tabsView = append(tabsView, tabStyle.Render(cat))
		}
	}

	// 탭 간 간격을 조정하고 모든 탭을 연결
	sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, tabsView...))
	sb.WriteString("\n\n") // 아래 콘텐츠와의 여백 추가

	// 목록이 비어있는 경우 메시지 표시
	if len(m.Entries[m.CategoryIndex]) == 0 {
		emptyMsg := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true).
			Render("🥳 Nothing to see here 🎊")
		sb.WriteString(emptyMsg)
	} else {
		sb.WriteString(m.List.View())
	}

	return docStyle.Render(sb.String())
}

func (m *ListModel) IsQuit() bool {
	return m.quit
}

func (m *ListModel) IsClone() bool {
	return m.clone
}

// NextRefresh returns the scheduled time for the next automatic refresh.
func (m *ListModel) NextRefresh() time.Time {
	return m.nextRefresh
}
