package types

import (
	"io/fs"
)

type FileInfo struct {
	Path  string
	Size  int64
	Mtime int32
	Mode  fs.FileMode
}

type FileList []FileInfo
