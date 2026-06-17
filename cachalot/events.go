package cachalot

import (
	"context"

	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
)

// EventType identifies what kind of cache update an Event carries.
type EventType string

const (
	// ContainersLoaded is emitted once, right after the initial container snapshot.
	ContainersLoaded EventType = "containers-loaded"
	// ImagesLoaded is emitted once, right after the initial image snapshot.
	ImagesLoaded EventType = "images-loaded"
	// ContainerUpdated is emitted whenever the container cache was refreshed from a Docker event.
	ContainerUpdated EventType = EventType(events.ContainerEventType)
	// ImageUpdated is emitted whenever the image cache was refreshed from a Docker event.
	ImageUpdated EventType = EventType(events.ImageEventType)
)

// Event notifies a consumer that the cache changed, so it can re-read
// Containers/Container/Images/Image instead of polling Docker on a timer.
type Event struct {
	Type    EventType
	ActorID string
}

// startEvents subscribes to the Docker daemon's event stream, filtered to
// the container and image actions the cache knows how to apply.
func (c *Client) startEvents(ctx context.Context) (<-chan events.Message, <-chan error) {
	f := filters.NewArgs()
	f.Add("type", string(events.ContainerEventType))
	f.Add("type", string(events.ImageEventType))

	f.Add("event", string(events.ActionCreate))
	f.Add("event", string(events.ActionStart))
	f.Add("event", string(events.ActionRestart))
	f.Add("event", string(events.ActionStop))
	f.Add("event", string(events.ActionRemove))
	f.Add("event", string(events.ActionDie))
	f.Add("event", string(events.ActionKill))
	f.Add("event", string(events.ActionPause))
	f.Add("event", string(events.ActionUnPause))
	f.Add("event", string(events.ActionRename))
	f.Add("event", string(events.ActionDestroy))

	f.Add("event", string(events.ActionPush))
	f.Add("event", string(events.ActionPull))
	f.Add("event", string(events.ActionPrune))
	f.Add("event", string(events.ActionDelete))

	return c.Events(ctx, events.ListOptions{Filters: f})
}

// listenEvents applies Docker events to the cache as they arrive and
// forwards a notification to consumers. It returns when the events context
// is cancelled or the Docker event stream ends.
func (c *Client) listenEvents() {
	defer close(c.done)

	for {
		select {
		case msg, ok := <-c.dockerEvents:
			if !ok {
				return
			}
			c.handleEvent(msg)
		case _, ok := <-c.dockerErrors:
			if !ok {
				return
			}
			return
		case <-c.eventsCtx.Done():
			return
		}
	}
}

func (c *Client) handleEvent(msg events.Message) {
	switch msg.Type {
	case events.ContainerEventType:
		c.refreshContainer(context.Background(), msg)
		c.emit(Event{Type: ContainerUpdated, ActorID: msg.Actor.ID})
	case events.ImageEventType:
		c.refreshImage(context.Background(), msg.Actor.ID)
		c.emit(Event{Type: ImageUpdated, ActorID: msg.Actor.ID})
	}
}

func (c *Client) emit(ev Event) {
	select {
	case c.out <- ev:
	case <-c.eventsCtx.Done():
	}
}
