package commands

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/docker/api/types/container"
	"github.com/kernaxis/gmd/cachalot"
)

func SwitchPageCmd(modelCreate func() tea.Model) tea.Cmd {
	return func() tea.Msg {
		var model tea.Model = nil
		if modelCreate != nil {
			model = modelCreate()
		}
		return SwitchPageMsg{Model: model}
	}
}

func ContainerCmd(cli *cachalot.Client, action Action, id string) tea.Cmd {
	return func() tea.Msg {
		msg := ContainerActionMsg{ContainerID: id, Action: action}
		ctx := context.Background()

		switch action {
		case StartContainerAction:
			msg.Err = cli.ContainerStart(ctx, id, container.StartOptions{})
		case StopContainerAction:
			msg.Err = cli.ContainerStop(ctx, id, container.StopOptions{})
		case RestartContainerAction:
			msg.Err = cli.ContainerRestart(ctx, id, container.StopOptions{})
		case RecreateContainerAction:
			msg.ContainerID, msg.Err = cli.RecreateContainer(id)
		}
		return msg
	}
}
