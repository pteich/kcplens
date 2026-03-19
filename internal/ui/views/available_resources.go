package views

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/peter/kcplens/internal/kcp"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
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
	list     list.Model
	gvr      schema.GroupVersionResource
	viewport viewport.Model
	state    APIListViewState
}

func NewResourceInstanceList() *ResourceInstanceList {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Resources"
	return &ResourceInstanceList{
		list:  l,
		state: APIListStateList,
	}
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
	case tea.KeyMsg:
		switch msg.String() {
		case "y":
			if r.state == APIListStateList {
				if item, ok := r.list.SelectedItem().(ResourceListItem); ok {
					yamlBytes, err := yaml.Marshal(item.res.Raw)
					if err != nil {
						r.viewport.SetContent(fmt.Sprintf("Error: %v", err))
					} else {
						r.viewport.SetContent(string(yamlBytes))
					}
					r.state = APIListStateDetail
					return r, nil
				}
			}
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		r.list.SetSize(msg.Width-h, msg.Height-v)
		r.viewport = viewport.New(msg.Width-h, msg.Height-v-2)
	}

	switch r.state {
	case APIListStateList:
		var cmd tea.Cmd
		r.list, cmd = r.list.Update(msg)
		return r, cmd
	case APIListStateDetail:
		var cmd tea.Cmd
		r.viewport, cmd = r.viewport.Update(msg)
		return r, cmd
	default:
		return r, nil
	}
}

func (r *ResourceInstanceList) View() string {
	if r.state == APIListStateDetail {
		title := lipgloss.NewStyle().Bold(true).Margin(1, 2, 0, 2).Render("YAML (press backspace/esc to go back)")
		help := helpStyle.Render("[backspace/esc] Back  [q] Quit")
		return title + "\n" + docStyle.Render(r.viewport.View()) + "\n" + help
	}

	help := helpStyle.Render("[y] Show YAML  [backspace/esc] Back to resource types  [q] Quit")
	return docStyle.Render(r.list.View()) + "\n" + help
}

func (r *ResourceInstanceList) SetWorkspacePath(path string) {
}

func (r *ResourceInstanceList) InDetailView() bool {
	return r.state == APIListStateDetail
}

func (r *ResourceInstanceList) ExitDetailView() {
	r.state = APIListStateList
}
