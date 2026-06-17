package containers

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kernaxis/gmd/cachalot"
	"github.com/kernaxis/gmd/tui/controllers/containerstats"
)

type ContainerUpdateMsg struct {
	ContainerID string
	Update      bool
	Err         error
}

func CheckContainerUpdate(cli *cachalot.Client, id string) tea.Cmd {
	return func() tea.Msg {
		update, err := cli.CheckUpdate(id)
		return ContainerUpdateMsg{ContainerID: id, Update: update, Err: err}
	}
}

func WaitStatsEvent(ch <-chan containerstats.StatsMsg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}
