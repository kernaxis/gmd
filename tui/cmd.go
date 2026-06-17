package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kernaxis/gmd/cachalot"
	"github.com/kernaxis/gmd/tui/commands"
)

// StartDockerClient connects to the Docker daemon and takes the initial
// snapshot of containers and images. It runs as a tea.Cmd (in a goroutine)
// so the TUI can render immediately while this (blocking) call completes.
func StartDockerClient() tea.Cmd {
	return func() tea.Msg {
		cli, err := cachalot.NewClient()
		return commands.ClientReadyMsg{Cli: cli, Err: err}
	}
}

func WaitDockerEvent(ch <-chan cachalot.Event) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

func SendResize(width, height int) tea.Cmd {
	return func() tea.Msg {
		return tea.WindowSizeMsg{
			Width:  width,
			Height: height,
		}
	}
}
