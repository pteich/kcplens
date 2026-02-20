package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/peter/kcplens/internal/kcp"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

var helpStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("241")).
	Margin(1, 2)

var emptyStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("241")).
	Italic(true).
	Margin(1, 2)

type WorkspaceItem struct {
	node *kcp.WorkspaceNode
}

func (i WorkspaceItem) Title() string       { return i.node.Name }
func (i WorkspaceItem) Description() string { return "Path: " + i.node.Path }
func (i WorkspaceItem) FilterValue() string { return i.node.Name + " " + i.node.Path }

type WorkspaceList struct {
	list             list.Model
	currentPath      string
	hasSubWorkspaces bool
}

func NewWorkspaceList() *WorkspaceList {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "KCP Workspaces"
	l.SetShowTitle(true)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)

	return &WorkspaceList{
		list:        l,
		currentPath: "root",
	}
}

func (w *WorkspaceList) SetItems(nodes []*kcp.WorkspaceNode) tea.Cmd {
	items := make([]list.Item, len(nodes))
	for i, n := range nodes {
		items[i] = WorkspaceItem{node: n}
	}
	w.hasSubWorkspaces = len(nodes) > 0
	return w.list.SetItems(items)
}

func (w *WorkspaceList) SetCurrentPath(path string) {
	w.currentPath = path
	w.list.Title = fmt.Sprintf("Workspace: %s", path)
}

func (w *WorkspaceList) Init() tea.Cmd {
	return nil
}

func (w *WorkspaceList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return w, tea.Quit
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		w.list.SetSize(msg.Width-h, msg.Height-v-4)
	}

	var cmd tea.Cmd
	w.list, cmd = w.list.Update(msg)
	return w, cmd
}

func (w *WorkspaceList) View() string {
	var b strings.Builder

	if w.hasSubWorkspaces {
		b.WriteString(docStyle.Render(w.list.View()))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render(
			fmt.Sprintf("Current: %s | [a] APIs  [s] SyncTargets  [r] Resources  [enter] Navigate  [backspace] Back  [q] Quit", w.currentPath),
		))
	} else {
		b.WriteString(docStyle.Render(w.list.Title))
		b.WriteString("\n\n")
		b.WriteString(emptyStyle.Render("No sub-workspaces. Use the commands below to explore this workspace."))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render(
			fmt.Sprintf("Current: %s | [a] APIs  [s] SyncTargets  [r] Resources  [backspace] Back  [q] Quit", w.currentPath),
		))
	}

	return b.String()
}

func (w *WorkspaceList) SelectedNode() *kcp.WorkspaceNode {
	i := w.list.SelectedItem()
	if i == nil {
		return nil
	}
	if wi, ok := i.(WorkspaceItem); ok {
		return wi.node
	}
	return nil
}
