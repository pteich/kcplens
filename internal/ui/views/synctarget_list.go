package views

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/peter/kcplens/internal/kcp"
)

type SyncTargetItem struct {
	target kcp.SyncTarget
}

func (i SyncTargetItem) Title() string { return i.target.Name }
func (i SyncTargetItem) Description() string {
	return fmt.Sprintf("Status: %s", i.target.Status)
}
func (i SyncTargetItem) FilterValue() string { return i.target.Name }

type SyncTargetList struct {
	list list.Model
}

func NewSyncTargetList() *SyncTargetList {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Sync Targets (Physical Clusters)"
	return &SyncTargetList{list: l}
}

func (s *SyncTargetList) SetItems(targets []kcp.SyncTarget) tea.Cmd {
	items := make([]list.Item, len(targets))
	for i, t := range targets {
		items[i] = SyncTargetItem{target: t}
	}
	return s.list.SetItems(items)
}

func (s *SyncTargetList) Init() tea.Cmd {
	return nil
}

func (s *SyncTargetList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := lipgloss.NewStyle().Margin(1, 2).GetFrameSize()
		s.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	s.list, cmd = s.list.Update(msg)
	return s, cmd
}

func (s *SyncTargetList) View() string {
	help := helpStyle.Render("[backspace/esc] Back  [q] Quit")
	return docStyle.Render(s.list.View()) + "\n" + help
}
