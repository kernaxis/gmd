package updater

import (
	"context"
	"fmt"
	"runtime"

	"github.com/creativeprojects/go-selfupdate"
)

const slug = "kernaxis/gmd"

type Checker struct {
	updater *selfupdate.Updater
}

type Result struct {
	Current string
	Latest  string
}

func NewChecker() (*Checker, error) {
	updaterConfig := selfupdate.Config{
		Validator: &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"},
	}

	up, err := selfupdate.NewUpdater(updaterConfig)
	if err != nil {
		return nil, fmt.Errorf("error occurred while creating updater: %w", err)
	}

	return &Checker{updater: up}, nil
}

func (c *Checker) DetectLatest(ctx context.Context, version string) (Result, bool, error) {
	latest, found, err := c.updater.DetectLatest(ctx, selfupdate.ParseSlug(slug))
	if err != nil {
		return Result{}, false, fmt.Errorf("an error occurred while detecting version: %w", err)
	}
	if !found {
		return Result{}, false, fmt.Errorf("latest version for %s/%s could not be found", runtime.GOOS, runtime.GOARCH)
	}

	return Result{
		Current: version,
		Latest:  latest.Version(),
	}, latest.LessOrEqual(version), nil
}

func ExecutablePath() (string, error) {
	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		return "", fmt.Errorf("error occurred while getting path to executable: %w", err)
	}

	return exe, nil
}

func (c *Checker) UpdateToLatest(ctx context.Context, exe string) (string, error) {
	latest, found, err := c.updater.DetectLatest(ctx, selfupdate.ParseSlug(slug))
	if err != nil {
		return "", fmt.Errorf("an error occurred while detecting version: %w", err)
	}
	if !found {
		return "", fmt.Errorf("latest version for %s/%s could not be found", runtime.GOOS, runtime.GOARCH)
	}

	if err := c.updater.UpdateTo(ctx, latest, exe); err != nil {
		return "", fmt.Errorf("error occurred while updating binary: %w", err)
	}

	return latest.Version(), nil
}
