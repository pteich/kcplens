package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var contextDocStyle = lipgloss.NewStyle().Margin(1, 2)

var contextHelpStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("241")).
	Margin(1, 2)

type ContextItem struct {
	name string
}

func (i ContextItem) Title() string       { return i.name }
func (i ContextItem) Description() string { return "Context" }
func (i ContextItem) FilterValue() string { return i.name }

type ContextSelector struct {
	list           list.Model
	hasSelected    bool
	kubeconfigPath string
	contexts       []string
	currentCtx     string
}

func NewContextSelector(kubeconfigPath string, contexts []string, currentCtx string) *ContextSelector {
	items := make([]list.Item, len(contexts))
	for i, ctx := range contexts {
		items[i] = ContextItem{name: ctx}
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select a kubeconfig context"
	l.SetShowTitle(true)
	l.SetShowStatusBar(true)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)

	cs := &ContextSelector{
		list:           l,
		hasSelected:    false,
		kubeconfigPath: kubeconfigPath,
		contexts:       contexts,
		currentCtx:     currentCtx,
	}
	return cs
}

func (c *ContextSelector) SelectedContext() string {
	if c.hasSelected {
		i := c.list.SelectedItem()
		if i != nil {
			return i.FilterValue()
		}
	}
	return ""
}

func (c *ContextSelector) KubeconfigPath() string {
	return c.kubeconfigPath
}

func (c *ContextSelector) Init() tea.Cmd {
	return nil
}

func (c *ContextSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if c.list.FilterState() != list.Filtering {
			switch msg.String() {
			case "enter":
				c.hasSelected = true
				return c, nil
			case "ctrl+c", "q":
				return c, tea.Quit
			}
		}
	case tea.WindowSizeMsg:
		h, v := contextDocStyle.GetFrameSize()
		c.list.SetSize(msg.Width-h, msg.Height-v-4)
	}

	var cmd tea.Cmd
	c.list, cmd = c.list.Update(msg)
	return c, cmd
}

func (c *ContextSelector) View() string {
	var b strings.Builder

	b.WriteString(contextDocStyle.Render(c.list.View()))
	b.WriteString("\n")
	b.WriteString(contextHelpStyle.Render(
		fmt.Sprintf("Current context: %s | [enter] Select  [q] Quit", c.currentCtx),
	))

	return b.String()
}
