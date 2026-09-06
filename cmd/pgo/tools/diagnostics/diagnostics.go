package diagnostics

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const defaultAddress = "http://127.0.0.1:19090"

// NewCommand creates read-only operational commands for a papp diagnostics port.
func NewCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "diagnostics",
		Short: "Query papp health, metrics, and runtime profiles",
	}
	command.AddCommand(newHealthCommand(), newMetricsURLCommand(), newProfileCommand())
	return command
}

func newHealthCommand() *cobra.Command {
	var address string
	command := &cobra.Command{
		Use:   "health",
		Short: "Print the diagnostics health report",
		RunE: func(cmd *cobra.Command, args []string) error {
			return printResponse(address, "/readyz")
		},
	}
	command.Flags().StringVar(&address, "addr", defaultAddress, "diagnostics base address")
	return command
}

func newMetricsURLCommand() *cobra.Command {
	var address string
	command := &cobra.Command{
		Use:   "metrics-url",
		Short: "Print the Prometheus metrics URL",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), endpoint(address, "/metrics"))
			return nil
		},
	}
	command.Flags().StringVar(&address, "addr", defaultAddress, "diagnostics base address")
	return command
}

func newProfileCommand() *cobra.Command {
	var address, output string
	var seconds int
	var open bool
	command := &cobra.Command{
		Use:   "profile {cpu|heap|goroutine}",
		Short: "Download a pprof profile and optionally open it with go tool pprof",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profileType := args[0]
			if output == "" {
				output = fmt.Sprintf("%s.pprof", profileType)
			}
			if err := downloadProfile(address, profileType, seconds, output); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "profile saved to %s\n", output)
			if !open {
				return nil
			}
			pprofCommand := exec.Command("go", "tool", "pprof", "-http=:0", output)
			pprofCommand.Stdout = cmd.OutOrStdout()
			pprofCommand.Stderr = cmd.ErrOrStderr()
			return pprofCommand.Run()
		},
	}
	command.Flags().StringVar(&address, "addr", defaultAddress, "diagnostics base address")
	command.Flags().StringVarP(&output, "output", "o", "", "output profile path")
	command.Flags().IntVar(&seconds, "seconds", 30, "CPU profile duration in seconds")
	command.Flags().BoolVar(&open, "open", false, "open the downloaded profile in go tool pprof")
	return command
}

func printResponse(address, path string) error {
	response, err := httpClient().Get(endpoint(address, path))
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if response.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("diagnostics returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	fmt.Println(string(body))
	return nil
}

func downloadProfile(address, profileType string, seconds int, output string) error {
	path := ""
	switch profileType {
	case "cpu":
		if seconds <= 0 {
			return fmt.Errorf("seconds must be positive")
		}
		path = fmt.Sprintf("/debug/pprof/profile?seconds=%d", seconds)
	case "heap", "goroutine":
		path = "/debug/pprof/" + profileType
	default:
		return fmt.Errorf("unsupported profile type %q", profileType)
	}

	response, err := httpClient().Get(endpoint(address, path))
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("profile request returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil && filepath.Dir(output) != "." {
		return err
	}
	file, err := os.Create(output)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, response.Body)
	return err
}

func endpoint(address, path string) string {
	if !strings.Contains(address, "://") {
		address = "http://" + address
	}
	parsed, err := url.Parse(address)
	if err != nil {
		return address + path
	}
	path, rawQuery, _ := strings.Cut(path, "?")
	parsed.Path = strings.TrimSuffix(parsed.Path, "/") + path
	parsed.RawQuery = rawQuery
	return parsed.String()
}

func httpClient() *http.Client {
	return &http.Client{Timeout: 35 * time.Second}
}
