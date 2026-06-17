package containerupdate

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/docker/api/types/container"
	"github.com/kernaxis/gmd/tui/controllers/containerupdate"
)

type UpdateFinishedMsg struct {
}

func startUpdate(c *containerupdate.Controller, cont container.InspectResponse) tea.Cmd {
	return func() tea.Msg {
		c.StartUpdate(cont)
		return containerupdate.ControllerUpdateMsg{}
	}
}

func waitUpdateEvent(updatech <-chan containerupdate.ControllerUpdateMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-updatech
		if !ok {
			return UpdateFinishedMsg{}
		}
		return msg
	}
}
