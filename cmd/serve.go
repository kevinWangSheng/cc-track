package cmd

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/shenghuikevin/cc-track/internal/web"
	"github.com/spf13/cobra"
)

var servePort int

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start web dashboard",
	RunE:  runServe,
}

func init() {
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8080, "port to listen on")
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	url := fmt.Sprintf("http://localhost:%d", servePort)

	// Try to open browser
	go func() {
		var openCmd string
		switch runtime.GOOS {
		case "darwin":
			openCmd = "open"
		case "linux":
			openCmd = "xdg-open"
		case "windows":
			openCmd = "start"
		}
		if openCmd != "" {
			exec.Command(openCmd, url).Start()
		}
	}()

	return web.Serve(servePort)
}
