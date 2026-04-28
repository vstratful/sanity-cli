package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vstratful/sanity-cli/internal/update"
)

var (
	updateCheckOnly bool
	updateForce     bool
	updateTimeout   time.Duration
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update the CLI to the latest GitHub Release",
	RunE:  runUpdate,
}

func init() {
	updateCmd.Flags().BoolVarP(&updateCheckOnly, "check", "c", false, "Only check for updates, don't install")
	updateCmd.Flags().BoolVarP(&updateForce, "force", "f", false, "Update without confirmation")
	updateCmd.Flags().DurationVar(&updateTimeout, "update-timeout", 30*time.Second, "Timeout for network operations")
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), updateTimeout)
	defer cancel()

	currentVersion := version
	fmt.Println("Checking for updates...")
	fmt.Printf("Current version: %s\n", currentVersion)

	release, err := update.CheckForUpdate(ctx, currentVersion)
	if err != nil {
		if errors.Is(err, update.ErrDevVersion) {
			fmt.Println("\nYou are running a development build.")
			fmt.Println("Auto-update is only available for released versions.")
			fmt.Println("Install a release from: https://github.com/vstratful/sanity-cli/releases")
			return nil
		}
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if release == nil {
		fmt.Println("\nYou are running the latest version.")
		return nil
	}

	fmt.Printf("Latest version:  %s\n", release.Version)
	if release.Description != "" {
		fmt.Println("\nRelease notes:")
		for _, line := range strings.Split(release.Description, "\n") {
			fmt.Printf("  %s\n", line)
		}
	}

	if updateCheckOnly {
		fmt.Println("\nRun 'sanity-cli update' to install.")
		return nil
	}

	if !updateForce {
		fmt.Print("\nUpdate? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Update cancelled.")
			return nil
		}
	}

	fmt.Printf("\nDownloading %s...\n", release.AssetName)
	cancel()
	downloadCtx, downloadCancel := context.WithTimeout(context.Background(), updateTimeout*2)
	defer downloadCancel()

	if err := update.ApplyUpdate(downloadCtx, release); err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "permission denied") || strings.Contains(errMsg, "access is denied") {
			fmt.Println("\nPermission denied. Try running with elevated privileges:")
			osName, _ := update.GetPlatformInfo()
			if osName == "windows" {
				fmt.Println("  Run as Administrator")
			} else {
				fmt.Println("  sudo sanity-cli update")
			}
			return err
		}
		if strings.Contains(errMsg, "checksum") {
			fmt.Println("\nSecurity warning: checksum verification failed!")
			fmt.Println("Download manually from: https://github.com/vstratful/sanity-cli/releases")
			return err
		}
		return err
	}

	fmt.Printf("\nSuccessfully updated to v%s!\n", release.Version)
	return nil
}
