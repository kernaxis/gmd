package tui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/kernaxis/gmd/cachalot"
	"github.com/kernaxis/gmd/tui/commands"
	"github.com/kernaxis/gmd/tui/componants"
	"github.com/kernaxis/gmd/tui/models/containers"
	"github.com/kernaxis/gmd/tui/models/images"
	"github.com/kernaxis/gmd/tui/models/maintab"
)

// ---------------------------------------------------
// Model Root
// ---------------------------------------------------

type Model struct {
	cli    *cachalot.Client
	stack  []tea.Model
	toasts []componants.Toast

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

	case images.DeleteImageMsg:
		var toast componants.Toast
		if msg.Err != nil {
			toast.Type = componants.ToastError
			toast.Message = "Unable to delete image " + msg.ID + ": " + msg.Err.Error()
		} else {
			toast.Type = componants.ToastInfo
			toast.Message = "Image " + msg.ID + " deleted"
		}
		m.toasts = append(m.toasts, toast)
		var cmd tea.Cmd
		m.stack[0], cmd = m.stack[0].Update(msg)
		return m, tea.Batch(tea.Tick(10*time.Second, func(t time.Time) tea.Msg {
			return componants.ToastExpiredMessage{}
		}), cmd)

	case commands.ContainerActionMsg:
		var toast componants.Toast
		if msg.Err != nil {
			toast.Type = componants.ToastError
			toast.Message = fmt.Sprintf("Unable to %s container %s: %s", msg.Action, msg.ContainerID, msg.Err)
		} else {
			toast.Type = componants.ToastInfo
			toast.Message = fmt.Sprintf("Container %s %s", msg.ContainerID, msg.Action.DoneString())
		}
		m.toasts = append(m.toasts, toast)
		var cmd tea.Cmd
		m.stack[0], cmd = m.stack[0].Update(msg)
		return m, tea.Batch(tea.Tick(10*time.Second, func(t time.Time) tea.Msg {
			return componants.ToastExpiredMessage{}
		}), cmd)

	case componants.ToastExpiredMessage:
		if len(m.toasts) > 0 {
			m.toasts = m.toasts[1:]
		}
		return m, nil

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

func mergeOverlay(base, overlay string) string {
	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")

	h := max(len(overlayLines), len(baseLines))

	result := make([]string, h)

	for i := 0; i < h; i++ {
		var bLine, oLine string

		if i < len(baseLines) {
			bLine = baseLines[i]
		}
		if i < len(overlayLines) {
			oLine = overlayLines[i]
		}

		// Si l'overlay contient quelque chose, il passe au-dessus
		if strings.TrimSpace(oLine) != "" {
			result[i] = overlayLineMerge(bLine, oLine)
		} else {
			result[i] = bLine
		}
	}

	return strings.Join(result, "\n")
}

// sgrSequence matches a single SGR escape code (colors, bold, reset, ...),
// the only kind of ANSI sequence lipgloss emits for the strings this app
// renders (no cursor movement, no OSC links).
var sgrSequence = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// isResetSGR reports whether code (e.g. "\x1b[0m" or "\x1b[m") clears all
// attributes rather than setting one.
func isResetSGR(code string) bool {
	params := code[2 : len(code)-1] // strip the leading "\x1b[" and trailing "m"
	return params == "" || params == "0"
}

type columnRun struct{ start, end int }

// opaqueRuns walks s and returns the column ranges that should be treated
// as opaque when overlaid on another line: any column where an SGR style is
// actively set when the character prints, or any column showing a
// non-space character. This is deliberately NOT the same as "every
// non-space character" — a space *inside* an active style (e.g. the spaces
// between words of a background-colored toast) is opaque too, otherwise the
// background would only paint the words and leave gaps at every space,
// which is exactly the "haché" look this replaces.
func opaqueRuns(s string) []columnRun {
	var runs []columnRun
	active := false
	openRun := -1
	col := 0

	pos := 0
	matches := sgrSequence.FindAllStringIndex(s, -1)
	mi := 0
	for pos < len(s) {
		if mi < len(matches) && matches[mi][0] == pos {
			active = !isResetSGR(s[matches[mi][0]:matches[mi][1]])
			pos = matches[mi][1]
			mi++
			continue
		}

		end := len(s)
		if mi < len(matches) {
			end = matches[mi][0]
		}

		for _, r := range s[pos:end] {
			if active || r != ' ' {
				if openRun == -1 {
					openRun = col
				}
			} else if openRun != -1 {
				runs = append(runs, columnRun{start: openRun, end: col})
				openRun = -1
			}
			col++
		}
		pos = end
	}
	if openRun != -1 {
		runs = append(runs, columnRun{start: openRun, end: col})
	}

	return runs
}

// overlayLineMerge fusionne overlay par-dessus base en raisonnant en colonnes
// d'écran réelles (ansi.StringWidth), en classant chaque colonne via
// opaqueRuns (donc en suivant l'état de style actif, pas juste "est-ce un
// espace"), et en découpant les chaînes avec ansi.Cut, qui ne coupe jamais
// une séquence ANSI en plein milieu et réinjecte le style actif à la
// frontière de coupe.
func overlayLineMerge(base, overlay string) string {
	baseWidth := ansi.StringWidth(base)
	overlayWidth := ansi.StringWidth(overlay)

	width := max(baseWidth, overlayWidth)
	if baseWidth < width {
		base += strings.Repeat(" ", width-baseWidth)
	}

	runs := opaqueRuns(overlay)

	var b strings.Builder
	pos := 0
	for _, r := range runs {
		if r.start > pos {
			b.WriteString(ansi.Cut(base, pos, r.start))
		}
		b.WriteString(ansi.Cut(overlay, r.start, r.end))
		pos = r.end
	}
	if pos < width {
		b.WriteString(ansi.Cut(base, pos, width))
	}

	return b.String()
}

func (m Model) renderToasts() string {
	if len(m.toasts) == 0 {
		return ""
	}

	var out []string
	for i := len(m.toasts) - 1; i >= 0; i-- {
		out = append(out, m.toasts[i].View())
	}

	return lipgloss.JoinVertical(lipgloss.Right, out...)
}

func (m Model) View() string {
	top := m.stack[len(m.stack)-1]
	baseView := top.View()
	//return top.View()

	toasts := m.renderToasts()
	if toasts == "" {
		return baseView
	}

	overlay := lipgloss.Place(
		m.screeWidth-10,
		m.screenHeight-10,
		lipgloss.Right,
		lipgloss.Bottom,
		toasts,
	)

	return mergeOverlay(baseView, overlay)
}
