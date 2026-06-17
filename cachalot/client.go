package cachalot

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
)

// Client is a Docker client that behaves like the official one (it embeds
// client.APIClient, so every official method is still available and goes
// straight to the Docker socket) while also keeping an in-memory cache of
// containers and images, refreshed from the Docker events stream.
//
// Use Containers/Container/Images/Image to read from the cache without
// touching the socket. Use the embedded client.APIClient methods (or the
// docker CLI conventions they mirror) for everything else, including
// mutations: the cache catches up automatically once Docker emits the
// corresponding event.
type Client struct {
	client.APIClient

	containers *store[container.InspectResponse]
	images     *store[Image]

	eventsCtx    context.Context
	eventsCancel context.CancelFunc
	dockerEvents <-chan events.Message
	dockerErrors <-chan error

	out  chan Event
	done chan struct{}
}

// NewClient returns a Client connected to the Docker daemon (using the same
// client.FromEnv conventions as the official client, extendable via opts),
// with its container and image cache already populated and the events
// subscription already running.
func NewClient(opts ...client.Opt) (*Client, error) {
	base := []client.Opt{client.FromEnv, client.WithAPIVersionNegotiation()}
	cli, err := client.NewClientWithOpts(append(base, opts...)...)
	if err != nil {
		return nil, fmt.Errorf("cachalot: create docker client: %w", err)
	}

	c := &Client{
		APIClient:  cli,
		containers: newStore[container.InspectResponse](),
		images:     newStore[Image](),
		out:        make(chan Event, 64),
		done:       make(chan struct{}),
	}

	if err := c.bootstrap(); err != nil {
		return nil, err
	}

	return c, nil
}

// bootstrap subscribes to Docker events, starts the goroutine that applies
// them to the cache, then takes a synchronous snapshot of containers and
// images so NewClient only returns once the cache is warm.
func (c *Client) bootstrap() error {
	c.eventsCtx, c.eventsCancel = context.WithCancel(context.Background())
	c.dockerEvents, c.dockerErrors = c.startEvents(c.eventsCtx)

	go c.listenEvents()

	ctx := context.Background()

	conts, err := c.snapshotContainers(ctx)
	if err != nil {
		c.eventsCancel()
		return fmt.Errorf("cachalot: initial container snapshot: %w", err)
	}
	for _, cont := range conts {
		c.containers.set(cont.ID, cont)
	}
	c.out <- Event{Type: ContainersLoaded}

	imgs, err := c.snapshotImages(ctx)
	if err != nil {
		c.eventsCancel()
		return fmt.Errorf("cachalot: initial image snapshot: %w", err)
	}
	for _, img := range imgs {
		c.images.set(img.ID, img)
	}
	c.out <- Event{Type: ImagesLoaded}

	return nil
}

// Updates returns the channel of cache-update notifications. Consumers can
// read it to know when to re-read Containers()/Images() instead of polling
// on a timer. It is named Updates, not Events, so it does not shadow the
// promoted client.APIClient.Events method.
func (c *Client) Updates() <-chan Event {
	return c.out
}

// Close stops the Docker events subscription and the cache update loop,
// then closes the underlying official client. It shadows the promoted
// client.APIClient.Close with the same signature, so callers using Client
// as a client.APIClient still get a clean shutdown.
func (c *Client) Close() error {
	c.eventsCancel()
	<-c.done
	close(c.out)
	return c.APIClient.Close()
}
