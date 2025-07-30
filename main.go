package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"os"

	"github.com/goodwaysIT/go-oracle-dr-dashboard/server"
)

//go:embed all:static
var staticFiles embed.FS

//go:embed all:locales
var localeFiles embed.FS

const version = "1.0.0"

func main() {
	// Define command-line flags
	configFile := flag.String("f", "config.yaml", "Path to the configuration file")
	showVersion := flag.Bool("v", false, "Display version information")
	showHelp := flag.Bool("h", false, "Display help information")

	// Custom usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Go Oracle DR Dashboard - A web-based monitoring tool for Oracle Data Guard.\n\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nFor more information, visit: https://github.com/goodwaysIT/go-oracle-dr-dashboard\n")
	}

	flag.Parse()

	if *showHelp {
		flag.Usage()
		return
	}

	if *showVersion {
		fmt.Println("Version:", version)
		return
	}

	// Create sub-filesystems to avoid path issues in the server package
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err) // Or handle more gracefully
	}

	localeFS, err := fs.Sub(localeFiles, "locales")
	if err != nil {
		panic(err) // Or handle more gracefully
	}

	server.Run(staticFS, localeFS, *configFile)
}
