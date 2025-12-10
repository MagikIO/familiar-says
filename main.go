package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/MagikIO/familiar-says/cmd"
	"github.com/MagikIO/familiar-says/internal/canvas"
)

//go:embed characters/*.json
var characterFS embed.FS

func init() {
	canvas.SetEmbeddedFS(characterFS)
}

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
