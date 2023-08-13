package rsync_test

import (
	"testing"

	"github.com/dashjay/rsync-s3/pkg/rsync"
	"github.com/stretchr/testify/assert"
)

func TestClient(t *testing.T) {
	client, err := rsync.NewClient(&rsync.ClientConfig{RsyncEndpoint: "rsync://mirrors.ustc.edu.cn/gentoo-portage"})
	assert.Nil(t, err)
	defer func() {
		assert.Nil(t, client.Shutdown())
	}()
	assert.Equal(t, "gentoo-portage", client.ModuleName())
	_, err = client.ListFiles()
	assert.Nil(t, err)
}
