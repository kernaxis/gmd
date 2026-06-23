package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/kernaxis/gmd/updater"
)

type versionStatus int

const (
	versionStatusChecking versionStatus = iota
	versionStatusUpToDate
	versionStatusUpdateAvailable
	versionStatusError
)

type versionCheckMsg struct {
	latest   string
	upToDate bool
	err      error
}

type versionSpinnerTickMsg struct{}

func CheckLatestVersion(version string) tea.Cmd {
	return func() tea.Msg {
		checker, err := updater.NewChecker()
		if err != nil {
			return versionCheckMsg{err: err}
		}

		result, upToDate, err := checker.DetectLatest(context.Background(), version)
		if err != nil {
			return versionCheckMsg{err: err}
		}

		return versionCheckMsg{
			latest:   result.Latest,
			upToDate: upToDate,
		}
	}
}

func TickVersionSpinner() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg {
		return versionSpinnerTickMsg{}
	})
}
