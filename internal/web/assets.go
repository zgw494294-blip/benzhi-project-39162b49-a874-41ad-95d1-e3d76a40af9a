package web

import (
	"embed"
	"io/fs"
)

//go:embed assets/*
var assetFiles embed.FS

func embeddedAssets() fs.FS {
	sub, err := fs.Sub(assetFiles, "assets")
	if err != nil {
		panic(err)
	}
	return sub
}
