package maintab

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kernaxis/gmd/cachalot"
	"github.com/kernaxis/gmd/tui/componants"
	"github.com/kernaxis/gmd/tui/models/containers"
	"github.com/kernaxis/gmd/tui/models/images"
	style "github.com/kernaxis/gmd/tui/styles"
)

// ---------------------------------------------------
// Messages
// ---------------------------------------------------

type SwitchTabMsg int

// ---------------------------------------------------
// Model Root
// ---------------------------------------------------

const (
	imagesTabIndex     = 0
	containersTabIndex = 1
)

type Model struct {
	lists     []componants.ListModel
	activeTab int
}

func New() Model {

	m := Model{
		lists: make([]componants.ListModel, 2),
	}

	m.lists[imagesTabIndex] = images.New()
	m.lists[containersTabIndex] = containers.New()
	return m
}

func (m Model) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0, 2)
	for i := range m.lists {
		cmds = append(cmds, m.lists[i].Init())
	}
	return tea.Batch(cmds...)
}

func (m Model) IsSearching() bool {
	return m.lists[m.activeTab].(componants.Searchable).IsSearching()
}

// ---------------------------------------------------
// Update
// ---------------------------------------------------

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {

		case "tab", "ctrl+tab":
			// On avance d’un onglet, circulation circulaire
			m.activeTab = (m.activeTab + 1) % len(m.lists)
			return m, nil

		case "shift+tab":
			m.activeTab--
			if m.activeTab < 0 {
				m.activeTab = len(m.lists) - 1
			}
			return m, nil

		}

		// Pass key stroke to active tab
		l, cmd := m.lists[m.activeTab].Update(msg)
		m.lists[m.activeTab] = l
		return m, cmd

	case cachalot.Event:
		switch msg.Type {
		case cachalot.ImagesLoaded, cachalot.ImageUpdated:
			l, cmd := m.lists[imagesTabIndex].Update(msg)
			m.lists[imagesTabIndex] = l
			return m, cmd
		case cachalot.ContainersLoaded, cachalot.ContainerUpdated:
			l, cmd := m.lists[containersTabIndex].Update(msg)
			m.lists[containersTabIndex] = l
			return m, cmd
		}
	}

	// pass all events to all lists
	var cmds []tea.Cmd
	for i := range m.lists {
		l, cmd := m.lists[i].Update(msg)
		cmds = append(cmds, cmd)
		m.lists[i] = l
	}

	return m, tea.Batch(cmds...)
}

// ---------------------------------------------------
// View
// ---------------------------------------------------

func (m Model) View() string {
	return lipgloss.JoinVertical(lipgloss.Left,
		m.viewTabs(),
		style.Normal().Bold(true).Render("────────────────────────────────────────────────────────────"),
		m.viewContent(),
	)
}

func (m Model) viewTabs() string {

	var (
		tabImages     = style.Inactive().Render(" Images ")
		tabContainers = style.Inactive().Render(" Containers ")
	)

	switch m.activeTab {
	case imagesTabIndex:
		tabImages = style.Success().Render(" Images ")
	case containersTabIndex:
		tabContainers = style.Success().Render(" Containers ")
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, tabImages, tabContainers)
}

func (m Model) viewContent() string {
	return m.lists[m.activeTab].View()
}
