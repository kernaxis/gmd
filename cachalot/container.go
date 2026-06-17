package cachalot

import (
	"context"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
)

// Containers returns every container currently in the cache. It never
// touches the Docker socket.
func (c *Client) Containers() []container.InspectResponse {
	return c.containers.list()
}

// Container returns the cached state of the container with the given ID.
// It never touches the Docker socket. If the container is not in the cache,
// ErrContainerNotFound is returned.
func (c *Client) Container(id string) (container.InspectResponse, error) {
	if cont, ok := c.containers.get(id); ok {
		return cont, nil
	}
	return container.InspectResponse{}, ErrContainerNotFound
}

// refreshContainer re-inspects the container behind a Docker event and
// updates the cache accordingly. On a destroy event it waits for the
// container to actually disappear from the API before evicting it, since
// Docker may emit the event slightly before ContainerInspect starts
// returning not-found.
func (c *Client) refreshContainer(ctx context.Context, msg events.Message) {
	if msg.Action == events.ActionDestroy {
		c.waitContainerGone(ctx, msg.Actor.ID)
		return
	}

	cont, err := c.ContainerInspect(ctx, msg.Actor.ID)
	if err != nil {
		c.containers.delete(msg.Actor.ID)
		return
	}
	c.containers.set(cont.ID, cont)
}

func (c *Client) waitContainerGone(ctx context.Context, id string) {
	const (
		attempts = 25
		delay    = 200 * time.Millisecond
	)

	for range attempts {
		if _, err := c.ContainerInspect(ctx, id); err != nil {
			break
		}
		time.Sleep(delay)
	}

	c.containers.delete(id)
}

// snapshotContainers lists every container known to the daemon, inspecting
// each one, for the initial cache population.
func (c *Client) snapshotContainers(ctx context.Context) ([]container.InspectResponse, error) {
	summaries, err := c.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, err
	}

	out := make([]container.InspectResponse, 0, len(summaries))
	for _, summary := range summaries {
		cont, err := c.ContainerInspect(ctx, summary.ID)
		if err != nil {
			continue
		}
		out = append(out, cont)
	}

	return out, nil
}
