package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type ListModel struct {
	Categories    []string
	CategoryIndex int
	Entries       [][]Entry
	List          list.Model
	clone         bool
	quit          bool
}

func (m *ListModel) Init() tea.Cmd { return nil }

func (m *ListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left":
			m.CategoryIndex = (m.CategoryIndex + len(m.Categories) - 1) % len(m.Categories)
			m.List.SetItems(ItemsFromEntries(m.Entries[m.CategoryIndex]))
		case "right":
			m.CategoryIndex = (m.CategoryIndex + 1) % len(m.Categories)
			m.List.SetItems(ItemsFromEntries(m.Entries[m.CategoryIndex]))
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
		m.List.SetSize(msg.Width-h, msg.Height-v)
	}
	m.List.Title = m.Categories[m.CategoryIndex]

	var cmd tea.Cmd
	m.List, cmd = m.List.Update(msg)

	return m, cmd
}

func (m *ListModel) View() string {
	sb := strings.Builder{}

	for _, cat := range m.Categories {
		if cat == m.Categories[m.CategoryIndex] {
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
	sb.WriteString(m.List.View())

	return docStyle.Render(sb.String())
}

func (m *ListModel) IsQuit() bool {
	return m.quit
}

func (m *ListModel) IsClone() bool {
	return m.clone
}
