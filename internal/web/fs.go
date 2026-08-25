package web

import "io/fs"

func fsReadFile(files fs.FS, name string) ([]byte, error) {
	return fs.ReadFile(files, name)
}
