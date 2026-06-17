package images

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/docker/api/types/image"
)

type ImagesLoadedMsg struct {
	Images []ImageItem
	Err    error
}

type DeleteImageMsg struct {
	ID  string
	Err error
}

func (m Model) FetchImagesCmd() tea.Cmd {
	return func() tea.Msg {
		images := m.cli.Images()
		imagesItems := make([]ImageItem, len(images))
		for i, img := range images {
			imagesItems[i] = ImageItem(img)
		}
		return ImagesLoadedMsg{Images: imagesItems, Err: nil}
	}
}

func (m Model) DeleteImagesCmd(id string) tea.Cmd {
	return func() tea.Msg {
		_, err := m.cli.ImageRemove(context.Background(), id, image.RemoveOptions{Force: false, PruneChildren: true})
		if err != nil {
			return DeleteImageMsg{ID: id, Err: err}
		}
		return DeleteImageMsg{ID: id, Err: nil}
	}
}
