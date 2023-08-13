package types_test

import (
	"testing"

	"github.com/dashjay/rsync-s3/pkg/types"
)

func TestDiffList(t *testing.T) {
	originList := make(types.FileList, 0)
	originList = append(originList, types.FileInfo{
		Path: "test",
		Size: 0,
	})
	originList = append(originList, types.FileInfo{
		Path: "new",
		Size: 0,
	})
	originList = append(originList, types.FileInfo{
		Path: "a",
		Size: 100,
	})
	newList := make(types.FileList, 0)
	newList = append(newList, types.FileInfo{
		Path: "a",
		Size: 100,
	})
	newItems, oldItems := originList.Diff(newList)
	for i := range oldItems {
		t.Logf("old: %d, %v", i, originList[oldItems[i]])
	}
	for i := range newItems {
		t.Logf("new: %d, %v", i, newList[newItems[i]])
	}
}
