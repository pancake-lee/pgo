//go:build !windows

package pclient

// RunApp is the platform-aware entrypoint
// - No args on Windows: runs UI mode
// - No args on non-Windows: runs CLI (interactive menu)
// - With args: always runs CLI mode
func RunApp(cli func(), ui func()) {
	cli()
}
