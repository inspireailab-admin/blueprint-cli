// Package ui bundles the dashboard SPA into the Go binary.
//
// During development the SPA is just a single index.html — the real
// React/Vite build pipeline lands once we start needing interactivity
// in PR 2. For now we only need a file the HTTP server can serve.
package ui

import (
	"embed"
	"io/fs"
)

//go:embed dist
var distFS embed.FS

// Assets returns the read-only filesystem rooted at the built dashboard.
// The HTTP server passes this to http.FileServer.
func Assets() fs.FS {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		// Embedded FS can't fail at runtime unless we ship a broken build,
		// in which case panicking here surfaces the bug immediately.
		panic("dashboard ui: failed to open embedded dist: " + err.Error())
	}
	return sub
}
