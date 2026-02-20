package views

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/peter/kcplens/internal/kcp"
	"sigs.k8s.io/yaml"
)

type APIItem struct {
	rel kcp.APIRelationship
}

func (i APIItem) Title() string {
	if i.rel.Type == "Binding" && i.rel.ExportName != "" {
		return fmt.Sprintf("Binding: %s", i.rel.ExportName)
	}
	if i.rel.Type == "Export" && i.rel.ResourceName != "" {
		return fmt.Sprintf("Export: %s/%s", i.rel.ResourceGroup, i.rel.ResourceName)
	}
	return i.rel.Name
}

func (i APIItem) Description() string {
	if i.rel.Type == "Binding" {
		path := i.rel.ExportPath
		if path == "" {
			path = "unknown"
		}
		return fmt.Sprintf("from: %s | status: %s", path, i.rel.Status)
	}
	if i.rel.Type == "Export" {
		return fmt.Sprintf("provides API to consumers | status: %s", i.rel.Status)
	}
	return fmt.Sprintf("status: %s", i.rel.Status)
}

func (i APIItem) FilterValue() string {
	return i.rel.Name + " " + i.rel.Type + " " + i.rel.ExportName + " " + i.rel.ResourceName
}

type APIListViewState int

const (
	APIListStateList APIListViewState = iota
	APIListStateDetail
)

type APIList struct {
	list     list.Model
	viewport viewport.Model
	state    APIListViewState
	ready    bool
}

func NewAPIList() *APIList {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "API Relationships"
	l.SetShowStatusBar(false)
	return &APIList{
		list:  l,
		state: APIListStateList,
	}
}

func (a *APIList) SetWorkspacePath(path string) {
	a.list.Title = fmt.Sprintf("API Relationships in %s", path)
}

func (a *APIList) SetItems(rels []kcp.APIRelationship) tea.Cmd {
	items := make([]list.Item, len(rels))
	for i, r := range rels {
		items[i] = APIItem{rel: r}
	}
	return a.list.SetItems(items)
}

func (a *APIList) Init() tea.Cmd {
	return nil
}

type ShowYAMLToggleMsg struct{}

func (a *APIList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y":
			if a.state == APIListStateList {
				if item, ok := a.list.SelectedItem().(APIItem); ok {
					yamlBytes, err := yaml.Marshal(item.rel.Raw)
					if err != nil {
						a.viewport.SetContent(fmt.Sprintf("Error: %v", err))
					} else {
						a.viewport.SetContent(string(yamlBytes))
					}
					a.state = APIListStateDetail
					return a, nil
				}
			}
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		a.list.SetSize(msg.Width-h, msg.Height-v)
		a.viewport = viewport.New(msg.Width-h, msg.Height-v-2)
		a.ready = true
	}

	switch a.state {
	case APIListStateList:
		var cmd tea.Cmd
		a.list, cmd = a.list.Update(msg)
		cmds = append(cmds, cmd)
	case APIListStateDetail:
		var cmd tea.Cmd
		a.viewport, cmd = a.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return a, tea.Batch(cmds...)
}

func (a *APIList) View() string {
	if a.state == APIListStateDetail {
		title := lipgloss.NewStyle().Bold(true).Margin(1, 2, 0, 2).Render("YAML (press backspace/esc to go back)")
		help := helpStyle.Render("[backspace/esc] Back  [q] Quit")
		return title + "\n" + docStyle.Render(a.viewport.View()) + "\n" + help
	}

	help := helpStyle.Render("[y] Show YAML  [backspace/esc] Back  [q] Quit")
	return docStyle.Render(a.list.View()) + "\n" + help
}

func (a *APIList) SelectedRelationship() *kcp.APIRelationship {
	i := a.list.SelectedItem()
	if i == nil {
		return nil
	}
	if apiItem, ok := i.(APIItem); ok {
		return &apiItem.rel
	}
	return nil
}

func (a *APIList) InDetailView() bool {
	return a.state == APIListStateDetail
}

func (a *APIList) ExitDetailView() {
	a.state = APIListStateList
}
