package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/kernaxis/gmd/cachalot"
	"github.com/kernaxis/gmd/tui/commands"
	"github.com/kernaxis/gmd/tui/componants"
	"github.com/kernaxis/gmd/tui/models/containers"
	"github.com/kernaxis/gmd/tui/models/maintab"
)

// ---------------------------------------------------
// Model Root
// ---------------------------------------------------

type Model struct {
	cli    *cachalot.Client
	stack  []tea.Model


	screeWidth   int
	screenHeight int
}

func NewModel() (Model, error) {
	m := Model{
		stack: []tea.Model{maintab.New()},
	}

	return m, nil
}

func (m Model) Init() tea.Cmd {
	top := m.stack[len(m.stack)-1]
	return tea.Batch(
		StartDockerClient(),
		top.Init(),
	)
}

// ---------------------------------------------------
// Update
// ---------------------------------------------------

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.screeWidth = msg.Width
		m.screenHeight = msg.Height

	case tea.KeyMsg:
		if searchable, ok := m.stack[len(m.stack)-1].(componants.Searchable); ok && searchable.IsSearching() {
			var cmd tea.Cmd
			m.stack[len(m.stack)-1], cmd = m.stack[len(m.stack)-1].Update(msg)
			return m, cmd
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case commands.ClientReadyMsg:
		var cmd tea.Cmd
		m.stack[0], cmd = m.stack[0].Update(msg)
		if msg.Err != nil {
			toast := componants.Toast{Type: componants.ToastError, Message: "Unable to connect to Docker: " + msg.Err.Error()}
			m.toasts = append(m.toasts, toast)
			return m, cmd
		}
		m.cli = msg.Cli
		return m, tea.Batch(WaitDockerEvent(m.cli.Updates()), cmd)

	case cachalot.Event:
		var cmd tea.Cmd
		m.stack[0], cmd = m.stack[0].Update(msg)
		return m, tea.Batch(WaitDockerEvent(m.cli.Updates()), cmd)

	case containers.ContainerUpdateMsg:
		var cmd tea.Cmd
		m.stack[0], cmd = m.stack[0].Update(msg)
		return m, cmd

	case commands.SwitchPageMsg:
		model := msg.Model
		if model == nil {
			m.stack = m.stack[:len(m.stack)-1] // pop
			return m, nil
		}
		cmd := model.Init()
		m.stack = append(m.stack, model)
		return m, tea.Batch( /*tea.ExitAltScreen,*/ cmd, SendResize(m.screeWidth, m.screenHeight))
	}

	top := m.stack[len(m.stack)-1]
	newTop, cmd := top.Update(msg)
	m.stack[len(m.stack)-1] = newTop

	return m, cmd
}

// ---------------------------------------------------
// View
// ---------------------------------------------------

func (m Model) View() string {
	top := m.stack[len(m.stack)-1]
	return top.View()
}
