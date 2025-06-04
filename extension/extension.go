package extension

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var embedFs embed.FS

var FS fs.FS

func init() {
	subFS, err := fs.Sub(embedFs, "dist")
	if err != nil {
		panic("failed to create sub filesystem for extension: " + err.Error())
	}

	FS = subFS
}
