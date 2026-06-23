package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"github.com/kernaxis/gmd/updater"
)

func init() {
	rootCmd.AddCommand(updateCmd)
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update gmd",
	Run: func(cmd *cobra.Command, args []string) {

		err := update(version)
		if err != nil {
			log.Fatal(err)
		}

	},
}

func update(version string) error {
	checker, err := updater.NewChecker()
	if err != nil {
		return err
	}

	exe, err := updater.ExecutablePath()
	if err != nil {
		return err
	}

	fmt.Println("→ Checking for latest version...")
	result, upToDate, err := checker.DetectLatest(context.Background(), version)
	if err != nil {
		return err
	}

	if upToDate {
		fmt.Println("✔ Already up to date:", result.Latest)
		return nil
	}

	fmt.Println("→ New version available:", result.Latest)

	updatedVersion, err := checker.UpdateToLatest(context.Background(), exe)
	if err != nil {
		return err
	}
	fmt.Printf("✔ Successfully updated to version %s\n", updatedVersion)
	return nil
}
