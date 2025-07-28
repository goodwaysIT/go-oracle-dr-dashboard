package main

import (
	"embed"
	"github.com/goodwaysIT/go-oracle-dr-dashboard/server"
	"io/fs"
)

//go:embed all:static
var staticFiles embed.FS

//go:embed all:locales
var localeFiles embed.FS

func main() {
	// Create sub-filesystems to avoid path issues in the server package
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err) // Or handle more gracefully
	}

	localeFS, err := fs.Sub(localeFiles, "locales")
	if err != nil {
		panic(err) // Or handle more gracefully
	}

	server.Run(staticFS, localeFS)
}
