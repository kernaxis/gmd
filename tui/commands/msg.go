package commands

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kernaxis/gmd/cachalot"
)

type SwitchPageMsg struct {
	Model tea.Model
}

// ClientReadyMsg signals that the cachalot.Client has finished connecting
// and taking its initial snapshot of containers and images.
type ClientReadyMsg struct {
	Cli *cachalot.Client
	Err error
}

type Action string

const (
	StartContainerAction    Action = "start"
	StopContainerAction     Action = "stop"
	RestartContainerAction  Action = "restart"
	RecreateContainerAction Action = "recreate"
)

func (a Action) DoneString() string {
	switch a {
	case StartContainerAction:
		return "started"
	case StopContainerAction:
		return "stopped"
	case RestartContainerAction:
		return "restarted"
	case RecreateContainerAction:
		return "recreated"
	default:
		return ""
	}
}

type ContainerActionMsg struct {
	ContainerID string
	Action      Action
	Update      bool
	Err         error
}

// type PullStartedMsg struct {
// 	Channel chan PullProgressMsg
// }

// type PullProgressMsg struct {
// 	LayerID         string
// 	Status          string
// 	Progress        string
// 	ProgressCurrent float64
// 	ProgressTotal   float64
// 	ProgressPct     float64
// 	Err             error
// }

// type PullCompleteMsg struct {
// }

// type StoppedContainerMsg struct {
// 	Err error
// }
