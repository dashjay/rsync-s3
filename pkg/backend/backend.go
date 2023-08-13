package backend

import (
	"io"

	"github.com/dashjay/rsync-s3/pkg/types"
)

type Interface interface {
	Stat(path []byte) (types.FileInfo, error)
	GetReader(path []byte) (io.ReadCloser, error)
	GetWriter(path []byte) (io.WriteCloser, error)
	List() ([]types.FileInfo, error)
}
