package resources

import "io/fs"

type FS interface {
	fs.FS
	fs.ReadDirFS
	fs.ReadFileFS
}
