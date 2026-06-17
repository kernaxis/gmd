package cachalot

import (
	"context"
	"log"
	"slices"
	"strings"

	"github.com/docker/docker/api/types/image"
)

// Image is a lightweight, cache-friendly view of a Docker image: its own
// metadata plus its parent ID (reconstructed from history for intermediate
// layers, which ImageList does not report on its own).
type Image struct {
	ID          string
	RepoTags    []string
	RepoDigests []string
	Size        int64
	ParentID    string
}

// Tag returns the best human-readable identifier for the image: its first
// repo tag, falling back to its first repo digest, falling back to its ID.
func (img Image) Tag() string {
	if len(img.RepoTags) > 0 {
		return img.RepoTags[0]
	}
	if len(img.RepoDigests) > 0 {
		return img.RepoDigests[0]
	}
	return img.ID
}

// Images returns every image currently in the cache, sorted by tag. It
// never touches the Docker socket.
func (c *Client) Images() []Image {
	out := c.images.list()
	slices.SortFunc(out, func(a, b Image) int {
		return strings.Compare(a.Tag(), b.Tag())
	})
	return out
}

// Image returns the cached state of the image with the given ID. It never
// touches the Docker socket. If the image is not in the cache,
// ErrImageNotFound is returned.
func (c *Client) Image(id string) (Image, error) {
	if img, ok := c.images.get(id); ok {
		return img, nil
	}
	return Image{}, ErrImageNotFound
}

// ImagesUnused returns the cached images that no cached container currently
// references.
func (c *Client) ImagesUnused() []Image {
	used := make(map[string]bool)
	for _, cont := range c.containers.list() {
		if cont.Image != "" {
			used[cont.Image] = true
		}
	}

	out := make([]Image, 0)
	for _, img := range c.images.list() {
		if !used[img.ID] {
			out = append(out, img)
		}
	}

	slices.SortFunc(out, func(a, b Image) int {
		return strings.Compare(a.Tag(), b.Tag())
	})
	return out
}

// refreshImage re-lists images and updates the cache entry for id, evicting
// it if it is gone.
func (c *Client) refreshImage(ctx context.Context, id string) {
	images, err := c.ImageList(ctx, image.ListOptions{All: true})
	if err != nil {
		return
	}

	for _, img := range images {
		if img.ID == id {
			c.images.set(id, Image{
				ID:          img.ID,
				RepoTags:    img.RepoTags,
				RepoDigests: img.RepoDigests,
				Size:        img.Size,
				ParentID:    img.ParentID,
			})
			return
		}
	}

	c.images.delete(id)
}

// snapshotImages lists every image known to the daemon, then reconstructs
// untagged intermediate layers from each image's history so the cache also
// knows about parents that ImageList alone would not report.
func (c *Client) snapshotImages(ctx context.Context) ([]Image, error) {
	list, err := c.ImageList(ctx, image.ListOptions{All: true})
	if err != nil {
		return nil, err
	}

	out := make(map[string]Image)

	add := func(id string, tags, digests []string, size int64, parent string) {
		if _, ok := out[id]; ok {
			return
		}
		out[id] = Image{ID: id, RepoTags: tags, RepoDigests: digests, Size: size, ParentID: parent}
	}

	for _, img := range list {
		add(img.ID, img.RepoTags, img.RepoDigests, img.Size, img.ParentID)
	}

	for _, img := range list {
		history, err := c.ImageHistory(ctx, img.ID)
		if err != nil {
			log.Printf("cachalot: image history for %s: %v", img.ID, err)
			continue
		}

		for _, layer := range history {
			if layer.ID == "<missing>" || layer.ID == "" {
				continue
			}
			if _, ok := out[layer.ID]; !ok {
				add(layer.ID, []string{}, []string{}, layer.Size, "")
			}
		}
	}

	result := make([]Image, 0, len(out))
	for _, img := range out {
		result = append(result, img)
	}
	return result, nil
}
