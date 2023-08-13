package types

import (
	"strings"
)

/*
Diff two sorted rsync file list, return their difference
list NEW: only R has.
list OLD: only L has.
*/
func (l FileList) Diff(r FileList) (newItems []int, oldItems []int) {
	newItems = make([]int, 0)
	oldItems = make([]int, 0)
	i := 0 // index of l
	j := 0 // index of r

	for i < len(l) && j < len(r) {
		// The result will be 0 if a==b, -1 if a < b, and +1 if a > b.
		// Compare their paths by bytes.Compare
		// The result will be 0 if a==b, -1 if a < b, and +1 if a > b
		// If 1, B doesn't have
		// If 0, A & B have
		// If -1, A doesn't have
		switch strings.Compare(l[i].Path, r[j].Path) {
		case 0:
			// nolint:gocritic // may be useful in future
			/*
				// because our backend will not save the mtime.
				fmt.Printf("mtime old: %s, mtime: new: %s\n",
					time.Unix(int64(l[i].Mtime), 0).String(), time.Unix(int64(r[i].Mtime), 0).String())
				if l[i].Mtime != r[j].Mtime || l[i].Size != r[j].Size {
					newItems = append(newItems, j)
				}
			*/
			if l[i].Size != r[j].Size {
				newItems = append(newItems, j)
			}
			i++
			j++
		case 1:
			newItems = append(newItems, j)
			j++
		case -1:
			oldItems = append(oldItems, i)
			i++
		}
	}

	// Handle remains
	for ; i < len(l); i++ {
		oldItems = append(oldItems, i)
	}
	for ; j < len(r); j++ {
		newItems = append(newItems, j)
	}

	return
}
