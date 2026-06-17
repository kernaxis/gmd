package images

import (
	"fmt"

	"github.com/dustin/go-humanize"
	"github.com/kernaxis/gmd/cachalot"
)

type ImageItem cachalot.Image

func (i ImageItem) Title() string { return cachalot.Image(i).Tag() }
func (i ImageItem) Description() string {
	return fmt.Sprintf("%s - %s", i.ID, humanize.Bytes(uint64(i.Size)))
}
func (i ImageItem) FilterValue() string { return i.Title() }
