package dashboard

import (
	"os/exec"
	"runtime"
)

// openBrowser fires the OS's "open this URL" handler. Best-effort — if
// the user is on a headless box or the command isn't found, we return
// an error and the caller logs the listen URL for manual paste.
//
// Lives in its own file because the platform fan-out is uninteresting
// noise inside server.go.
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		// `rundll32 url.dll,FileProtocolHandler <url>` is the most reliable
		// way to open a URL on Windows without going through cmd.exe (which
		// would need extra quoting). It's been the recommended pattern since
		// Windows XP and still works on Win 11.
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
