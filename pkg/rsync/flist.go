package rsync

import (
	"bytes"
	"encoding/json"
	"os"

	"github.com/dashjay/rsync-s3/pkg/types"
)

type FileMode uint32

type InnerFileInfo struct {
	Path       []byte
	Size       int64
	Mtime      int32
	Mode       FileMode
	TargetLink []byte
}

func (i *InnerFileInfo) String() string {
	bin, _ := json.Marshal(i)
	return string(bin)
}

func (i *InnerFileInfo) ToFileInfo() types.FileInfo {
	return types.FileInfo{
		Path:  string(i.Path),
		Size:  i.Size,
		Mtime: i.Mtime,
		Mode:  i.Mode.Convert(),
	}
}

type InnerFileList []InnerFileInfo

func (l InnerFileList) ToFileList() types.FileList {
	list := make(types.FileList, 0, len(l))
	for i := range l {
		list = append(list, l[i].ToFileInfo())
	}
	return list
}

func (l InnerFileList) Len() int {
	return len(l)
}

func (l InnerFileList) Less(i, j int) bool {
	return bytes.Compare(l[i].Path, l[j].Path) == -1
}

func (l InnerFileList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func (m FileMode) IsREG() bool {
	return (m & SIfmt) == SIfreg
}

func (m FileMode) IsDIR() bool {
	return (m & SIfmt) == SIfdir
}

func (m FileMode) IsBLK() bool {
	return (m & SIfmt) == SIfblk
}

func (m FileMode) IsLNK() bool {
	return (m & SIfmt) == SIflnk
}

func (m FileMode) IsFIFO() bool {
	return (m & SIfmt) == SIfifo
}

func (m FileMode) IsSOCK() bool {
	return (m & SIfmt) == SIfsock
}

// Perm Return only unix permission bits
func (m FileMode) Perm() FileMode {
	return m & 0777
}

// Convert to os.FileMode
func (m FileMode) Convert() os.FileMode {
	mode := os.FileMode(m & 0777)
	switch m & SIfmt {
	case SIfreg:
		// For regular files, none will be set.
	case SIfdir:
		mode |= os.ModeDir
	case SIflnk:
		mode |= os.ModeSymlink
	case SIfblk:
		mode |= os.ModeDevice
	case SIfsock:
		mode |= os.ModeSocket
	case SIfifo:
		mode |= os.ModeNamedPipe
	case SIfchr:
		mode |= os.ModeCharDevice
	default:
		mode |= os.ModeIrregular
	}
	return mode
}

// strmode
func (m FileMode) String() string {
	chars := []byte("-rwxrwxrwx")
	switch m & SIfmt {
	case SIfreg:
	case SIfdir:
		chars[0] = 'd'
	case SIflnk:
		chars[0] = 'l'
	case SIfblk:
		chars[0] = 'b'
	case SIfsock:
		chars[0] = 's'
	case SIfifo:
		chars[0] = 'p'
	case SIfchr:
		chars[0] = 'c'
	default:
		chars[0] = '?'
	}
	for i := 0; i < 9; i++ {
		if m&(1<<i) == 0 {
			chars[9-i] = '-'
		}
	}
	return string(chars)
}
