package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/peter/kcplens/internal/kcp"
	"github.com/peter/kcplens/internal/ui"
)

func main() {
	kubeconfig := flag.String("kubeconfig", "", "path to the kubeconfig file")
	flag.Parse()

	contexts, currentCtx, err := kcp.GetContexts(*kubeconfig)
	if err != nil {
		fmt.Printf("Failed to load kubeconfig contexts: %v\n", err)
		os.Exit(1)
	}

	var appModel *ui.AppModel

	if len(contexts) > 1 {
		cm, err := kcp.NewClientManagerWithContext(*kubeconfig, currentCtx)
		if err != nil {
			fmt.Printf("Failed to initialize KCP client: %v\n", err)
			os.Exit(1)
		}
		appModel = ui.NewAppModelWithContextSelector(cm, *kubeconfig, contexts, currentCtx)
	} else {
		cm, err := kcp.NewClientManager(*kubeconfig)
		if err != nil {
			fmt.Printf("Failed to initialize KCP client: %v\n", err)
			os.Exit(1)
		}
		appModel = ui.NewAppModel(cm)
	}

	p := tea.NewProgram(appModel, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error starting the TUI: %v\n", err)
		os.Exit(1)
	}
}
