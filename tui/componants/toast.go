package componants

import (
	"github.com/charmbracelet/lipgloss"
	style "github.com/kernaxis/gmd/tui/styles"
)

type ToastType int

const (
	ToastInfo ToastType = iota
	ToastWarning
	ToastError
)

type Toast struct {
	Message string
	Type    ToastType
}

type ToastExpiredMessage struct {
	ID int
}

func (t Toast) icon() string {
	switch t.Type {
	case ToastInfo:
		return "ℹ"
	case ToastWarning:
		return "⚠"
	case ToastError:
		return "✖"
	default:
		return ""
	}
}

func (t Toast) style() lipgloss.Style {
	base := lipgloss.NewStyle().
		Padding(0, 1).
		Margin(0, 1).
		Border(lipgloss.RoundedBorder()).
		Bold(true)

	switch t.Type {
	case ToastInfo:
		return base.
			Foreground(style.Default.Fg).
			Background(style.ColorSuccess()).
			BorderForeground(style.ColorSuccess())

	case ToastWarning:
		return base.
			Foreground(style.Default.Fg).
			Background(style.ColorWarning()).
			BorderForeground(style.ColorWarning())

	case ToastError:
		return base.
			Foreground(style.Default.Fg).
			Background(style.ColorDanger()).
			BorderForeground(style.ColorDanger())

	default:
		return base
	}
}

func (t Toast) View() string {
	icon := t.icon()

	maxContentWidth := 50

	message := lipgloss.NewStyle().
		Width(maxContentWidth).
		Render(icon + " " + t.Message)

	return t.style().
		MaxWidth(60).
		Render(message)
}
