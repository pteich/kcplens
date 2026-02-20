package views

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/peter/kcplens/internal/kcp"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type AvailableResourceItem struct {
	res kcp.AvailableResource
}

func (i AvailableResourceItem) Title() string {
	group := i.res.GVR.Group
	if group == "" {
		group = "core"
	}
	return fmt.Sprintf("%s (%s.%s)", i.res.Kind, i.res.GVR.Resource, group)
}

func (i AvailableResourceItem) Description() string {
	scope := "cluster-scoped"
	if i.res.Namespaced {
		scope = "namespaced"
	}
	return fmt.Sprintf("GVR: %s/%s | %s", i.res.GVR.GroupVersion(), i.res.GVR.Resource, scope)
}

func (i AvailableResourceItem) FilterValue() string {
	return i.res.Kind + " " + i.res.GVR.Resource + " " + i.res.GVR.Group
}

type AvailableResourceList struct {
	list list.Model
}

func NewAvailableResourceList() *AvailableResourceList {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Available Resources"
	return &AvailableResourceList{list: l}
}

func (a *AvailableResourceList) SetItems(resources []kcp.AvailableResource) tea.Cmd {
	items := make([]list.Item, len(resources))
	for i, r := range resources {
		items[i] = AvailableResourceItem{res: r}
	}
	return a.list.SetItems(items)
}

func (a *AvailableResourceList) Init() tea.Cmd {
	return nil
}

func (a *AvailableResourceList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		a.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	a.list, cmd = a.list.Update(msg)
	return a, cmd
}

func (a *AvailableResourceList) View() string {
	help := helpStyle.Render("[enter] List instances  [backspace/esc] Back  [q] Quit")
	return docStyle.Render(a.list.View()) + "\n" + help
}

func (a *AvailableResourceList) SelectedResource() *kcp.AvailableResource {
	i := a.list.SelectedItem()
	if i == nil {
		return nil
	}
	if ri, ok := i.(AvailableResourceItem); ok {
		return &ri.res
	}
	return nil
}

func (a *AvailableResourceList) Title() string {
	return a.list.Title
}

func (a *AvailableResourceList) SetTitle(title string) {
	a.list.Title = title
}

type ResourceListItem struct {
	res kcp.GenericResource
}

func (i ResourceListItem) Title() string {
	return i.res.Name
}

func (i ResourceListItem) Description() string {
	ns := i.res.Namespace
	if ns == "" {
		ns = "-"
	}
	return fmt.Sprintf("Namespace: %s | Workspace: %s", ns, i.res.Workspace)
}

func (i ResourceListItem) FilterValue() string {
	return i.res.Name + " " + i.res.Namespace
}

type ResourceInstanceList struct {
	list list.Model
	gvr  schema.GroupVersionResource
}

func NewResourceInstanceList() *ResourceInstanceList {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Resources"
	return &ResourceInstanceList{list: l}
}

func (r *ResourceInstanceList) SetItems(resources []kcp.GenericResource) tea.Cmd {
	items := make([]list.Item, len(resources))
	for i, res := range resources {
		items[i] = ResourceListItem{res: res}
	}
	return r.list.SetItems(items)
}

func (r *ResourceInstanceList) SetGVR(gvr schema.GroupVersionResource) {
	r.gvr = gvr
	group := gvr.Group
	if group == "" {
		group = "core"
	}
	r.list.Title = fmt.Sprintf("%s (%s.%s)", gvr.Resource, gvr.Resource, group)
}

func (r *ResourceInstanceList) GVR() schema.GroupVersionResource {
	return r.gvr
}

func (r *ResourceInstanceList) Init() tea.Cmd {
	return nil
}

func (r *ResourceInstanceList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		r.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	r.list, cmd = r.list.Update(msg)
	return r, cmd
}

func (r *ResourceInstanceList) View() string {
	help := helpStyle.Render("[backspace/esc] Back to resource types  [q] Quit")
	return docStyle.Render(r.list.View()) + "\n" + help
}

func (r *ResourceInstanceList) SetWorkspacePath(path string) {
}
