package ui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/peter/kcplens/internal/kcp"
	"github.com/peter/kcplens/internal/ui/views"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type AppState int

const (
	StateWorkspaces AppState = iota
	StateAPIs
	StateSyncTargets
	StateAvailableResources
	StateResourceInstances
)

type AppModel struct {
	clientMgr             *kcp.ClientManager
	workspaceList         *views.WorkspaceList
	apiList               *views.APIList
	syncTargetList        *views.SyncTargetList
	availableResourceList *views.AvailableResourceList
	resourceInstanceList  *views.ResourceInstanceList
	state                 AppState
	err                   error
	loading               bool
	history               []string
}

func NewAppModel(cm *kcp.ClientManager) *AppModel {
	return &AppModel{
		clientMgr:             cm,
		loading:               true,
		workspaceList:         views.NewWorkspaceList(),
		apiList:               views.NewAPIList(),
		syncTargetList:        views.NewSyncTargetList(),
		availableResourceList: views.NewAvailableResourceList(),
		resourceInstanceList:  views.NewResourceInstanceList(),
		state:                 StateWorkspaces,
		history:               []string{},
	}
}

type workspacesLoadedMsg struct {
	workspaces []*kcp.WorkspaceNode
}

type errorMsg struct {
	err error
}

type apisLoadedMsg struct {
	apis []kcp.APIRelationship
}

type syncTargetsLoadedMsg struct {
	targets []kcp.SyncTarget
}

type availableResourcesLoadedMsg struct {
	resources []kcp.AvailableResource
}

type resourceInstancesLoadedMsg struct {
	resources []kcp.GenericResource
}

func fetchWorkspacesCmd(cm *kcp.ClientManager, path string) tea.Cmd {
	return func() tea.Msg {
		ws, err := cm.DiscoverWorkspaces(context.Background(), path)
		if err != nil {
			return errorMsg{err}
		}
		return workspacesLoadedMsg{ws}
	}
}

func fetchAPIsCmd(cm *kcp.ClientManager, path string) tea.Cmd {
	return func() tea.Msg {
		apis, err := cm.DiscoverAPIRelationships(context.Background(), path)
		if err != nil {
			return errorMsg{err}
		}
		return apisLoadedMsg{apis}
	}
}

func fetchSyncTargetsCmd(cm *kcp.ClientManager, path string) tea.Cmd {
	return func() tea.Msg {
		targets, err := cm.DiscoverSyncTargets(context.Background(), path)
		if err != nil {
			return errorMsg{err}
		}
		return syncTargetsLoadedMsg{targets}
	}
}

func fetchAvailableResourcesCmd(cm *kcp.ClientManager, path string) tea.Cmd {
	return func() tea.Msg {
		res, err := cm.DiscoverAvailableResources(context.Background(), path)
		if err != nil {
			return errorMsg{err}
		}
		return availableResourcesLoadedMsg{res}
	}
}

func fetchResourceInstancesCmd(cm *kcp.ClientManager, path string, gvr schema.GroupVersionResource) tea.Cmd {
	return func() tea.Msg {
		res, err := cm.DiscoverResourcesInWorkspace(context.Background(), path, gvr, "")
		if err != nil {
			return errorMsg{err}
		}
		return resourceInstancesLoadedMsg{res}
	}
}

func (m *AppModel) Init() tea.Cmd {
	return tea.Batch(
		fetchWorkspacesCmd(m.clientMgr, "root"),
		m.workspaceList.Init(),
	)
}

func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}

		if !m.loading && m.err == nil {
			cmds = append(cmds, m.handleKey(msg))
		}

	case tea.WindowSizeMsg:
		m.workspaceList.Update(msg)
		m.apiList.Update(msg)
		m.syncTargetList.Update(msg)
		m.availableResourceList.Update(msg)
		m.resourceInstanceList.Update(msg)

	case workspacesLoadedMsg:
		m.loading = false
		m.err = nil
		m.workspaceList.SetCurrentPath(m.clientMgr.CurrentWorkspace())
		cmds = append(cmds, m.workspaceList.SetItems(msg.workspaces))

	case apisLoadedMsg:
		m.loading = false
		m.err = nil
		cmds = append(cmds, m.apiList.SetItems(msg.apis))

	case syncTargetsLoadedMsg:
		m.loading = false
		m.err = nil
		cmds = append(cmds, m.syncTargetList.SetItems(msg.targets))

	case availableResourcesLoadedMsg:
		m.loading = false
		m.err = nil
		cmds = append(cmds, m.availableResourceList.SetItems(msg.resources))

	case resourceInstancesLoadedMsg:
		m.loading = false
		m.err = nil
		cmds = append(cmds, m.resourceInstanceList.SetItems(msg.resources))

	case errorMsg:
		m.err = msg.err
		m.loading = false
	}

	if !m.loading && m.err == nil {
		cmds = append(cmds, m.updateCurrentView(msg))
	}

	return m, tea.Batch(cmds...)
}

func (m *AppModel) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		return m.handleEnter()
	case "a":
		return m.handleAPIKey()
	case "s":
		return m.handleSyncTargetsKey()
	case "r":
		return m.handleResourcesKey()
	case "backspace", "esc":
		return m.handleBackspace()
	}
	return nil
}

func (m *AppModel) handleEnter() tea.Cmd {
	switch m.state {
	case StateWorkspaces:
		selected := m.workspaceList.SelectedNode()
		if selected != nil {
			m.loading = true
			m.history = append(m.history, m.clientMgr.CurrentWorkspace())
			m.clientMgr.SetWorkspace(selected.Path)
			return fetchWorkspacesCmd(m.clientMgr, selected.Path)
		}
	case StateAvailableResources:
		selected := m.availableResourceList.SelectedResource()
		if selected != nil {
			m.state = StateResourceInstances
			m.loading = true
			m.resourceInstanceList.SetGVR(selected.GVR)
			return fetchResourceInstancesCmd(m.clientMgr, m.clientMgr.CurrentWorkspace(), selected.GVR)
		}
	}
	return nil
}

func (m *AppModel) handleAPIKey() tea.Cmd {
	if m.state == StateWorkspaces {
		m.state = StateAPIs
		m.loading = true
		m.apiList.SetWorkspacePath(m.clientMgr.CurrentWorkspace())
		return fetchAPIsCmd(m.clientMgr, m.clientMgr.CurrentWorkspace())
	}
	return nil
}

func (m *AppModel) handleSyncTargetsKey() tea.Cmd {
	if m.state == StateWorkspaces {
		m.state = StateSyncTargets
		m.loading = true
		return fetchSyncTargetsCmd(m.clientMgr, m.clientMgr.CurrentWorkspace())
	}
	return nil
}

func (m *AppModel) handleResourcesKey() tea.Cmd {
	if m.state == StateWorkspaces {
		m.state = StateAvailableResources
		m.loading = true
		m.availableResourceList.SetTitle("Available Resources in " + m.clientMgr.CurrentWorkspace())
		return fetchAvailableResourcesCmd(m.clientMgr, m.clientMgr.CurrentWorkspace())
	}
	return nil
}

func (m *AppModel) handleBackspace() tea.Cmd {
	switch m.state {
	case StateAPIs:
		if m.apiList.InDetailView() {
			m.apiList.ExitDetailView()
			return nil
		}
		m.state = StateWorkspaces
		return nil
	case StateSyncTargets:
		m.state = StateWorkspaces
		return nil
	case StateAvailableResources:
		m.state = StateWorkspaces
		return nil
	case StateResourceInstances:
		m.state = StateAvailableResources
		return nil
	case StateWorkspaces:
		if len(m.history) > 0 {
			m.loading = true
			prev := m.history[len(m.history)-1]
			m.history = m.history[:len(m.history)-1]
			m.clientMgr.SetWorkspace(prev)
			return fetchWorkspacesCmd(m.clientMgr, prev)
		}
	}
	return nil
}

func (m *AppModel) updateCurrentView(msg tea.Msg) tea.Cmd {
	switch m.state {
	case StateWorkspaces:
		_, cmd := m.workspaceList.Update(msg)
		return cmd
	case StateAPIs:
		_, cmd := m.apiList.Update(msg)
		return cmd
	case StateSyncTargets:
		_, cmd := m.syncTargetList.Update(msg)
		return cmd
	case StateAvailableResources:
		_, cmd := m.availableResourceList.Update(msg)
		return cmd
	case StateResourceInstances:
		_, cmd := m.resourceInstanceList.Update(msg)
		return cmd
	}
	return nil
}

func (m *AppModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error in %s: %v\n\nPress backspace to go back or q to quit.", m.clientMgr.CurrentWorkspace(), m.err)
	}
	if m.loading {
		return fmt.Sprintf("Loading for %s...\n", m.clientMgr.CurrentWorkspace())
	}

	switch m.state {
	case StateWorkspaces:
		return m.workspaceList.View()
	case StateAPIs:
		return m.apiList.View()
	case StateSyncTargets:
		return m.syncTargetList.View()
	case StateAvailableResources:
		return m.availableResourceList.View()
	case StateResourceInstances:
		return m.resourceInstanceList.View()
	default:
		return m.workspaceList.View()
	}
}
