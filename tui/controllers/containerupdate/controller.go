package containerupdate

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/kernaxis/gmd/cachalot"
	style "github.com/kernaxis/gmd/tui/styles"
)

type ControllerUpdateMsg struct {
	RecreateOnly bool
}

type createResponse struct {
	Resp container.CreateResponse
	Err  error
}

type Controller struct {
	m          sync.RWMutex
	cli        *cachalot.Client
	updateChan chan ControllerUpdateMsg

	order  []string
	layers map[string]string
	lines  []string
}

func New(cli *cachalot.Client) *Controller {
	c := Controller{
		cli:        cli,
		updateChan: make(chan ControllerUpdateMsg, 10),
	}
	return &c
}

func (c *Controller) Events() <-chan ControllerUpdateMsg {
	return c.updateChan
}

func (c *Controller) GetLines() []string {
	c.m.RLock()
	defer c.m.RUnlock()
	return c.lines
}

func (c *Controller) StartUpdate(cont container.InspectResponse) {
	c.order = []string{}
	c.layers = make(map[string]string)
	go c.updateContainer(cont)
}

func (c *Controller) updateContainer(cont container.InspectResponse) {

	containerName := strings.TrimPrefix(cont.Name, "/")

	done := make(chan error)
	defer close(done)

	err := c.cli.PullImageWithProgress(context.Background(), cont.Config.Image, func(msg *jsonmessage.JSONMessage) {

		layerId := msg.ID
		if layerId == "" {
			layerId = fmt.Sprintf("general-%d", len(c.layers)) // évite collision
		}

		line := msg.Status

		if msg.Progress != nil {
			currentProgress := msg.Progress.Current
			totalProgress := msg.Progress.Total
			if currentProgress > 0 && totalProgress > 0 {
				line += " " + msg.Progress.String()
			}
		}

		c.m.Lock()
		if _, exists := c.layers[layerId]; !exists {
			c.order = append(c.order, layerId) // première fois qu’on voit ce layer
		}
		c.layers[layerId] = line

		c.lines = c.lines[:0]
		for _, id := range c.order {
			c.lines = append(c.lines, c.layers[id])
		}
		c.m.Unlock()

		log.Printf("line: %s", line)

		c.updateChan <- ControllerUpdateMsg{}
	})

	if err != nil {
		log.Printf("Error pull for image %s : %v", cont.Config.Image, err)
		c.m.Lock()
		c.lines = append(c.lines, fmt.Sprintf("Error pull image: %v", err))
		c.m.Unlock()
		c.updateChan <- ControllerUpdateMsg{}
		return
	}

	containerConfig, err := c.cli.ContainerInspect(context.Background(), cont.ID)
	if err != nil {
		log.Printf("Error get config for container %s : %v", cont.ID, err)
		c.m.Lock()
		c.lines = append(c.lines, fmt.Sprintf("Error get config: %v", err))
		c.m.Unlock()
		c.updateChan <- ControllerUpdateMsg{}
		return
	}

	add := true
	err = spinUntilDone(func() error {
		return c.cli.ContainerStop(context.Background(), cont.ID, container.StopOptions{})
	}, func(frame string) {
		c.m.Lock()
		if add {
			c.lines = append(c.lines, fmt.Sprintf("%s Stoping container: %s", frame, containerName))
			add = false
		} else {
			c.lines[len(c.lines)-1] = fmt.Sprintf("%s Stoping container: %s", frame, containerName)
		}
		c.m.Unlock()
		c.updateChan <- ControllerUpdateMsg{}
	})

	if err != nil {
		c.m.Lock()
		c.lines = append(c.lines, fmt.Sprintf("Error stop: %v", err))
		c.m.Unlock()
		c.updateChan <- ControllerUpdateMsg{}
		return
	}

	c.m.Lock()
	c.lines[len(c.lines)-1] = fmt.Sprintf("%s Stoping container: %s", style.Success().Render("✓"), containerName)
	c.m.Unlock()
	c.updateChan <- ControllerUpdateMsg{}

	add = true
	err = spinUntilDone(func() error {
		return c.cli.ContainerRemove(context.Background(), cont.ID, container.RemoveOptions{})
	}, func(frame string) {
		c.m.Lock()
		if add {
			c.lines = append(c.lines, fmt.Sprintf("%s Removing container: %s", frame, containerName))
			add = false
		} else {
			c.lines[len(c.lines)-1] = fmt.Sprintf("%s Removing container: %s", frame, containerName)
		}
		c.m.Unlock()
		c.updateChan <- ControllerUpdateMsg{}
	})

	if err != nil {
		c.m.Lock()
		c.lines = append(c.lines, fmt.Sprintf("Error remove: %v", err))
		c.m.Unlock()
		c.updateChan <- ControllerUpdateMsg{}
		return
	}

	c.m.Lock()
	c.lines[len(c.lines)-1] = fmt.Sprintf("%s Removing container: %s", style.Success().Render("✓"), containerName)
	c.m.Unlock()
	c.updateChan <- ControllerUpdateMsg{}

	add = true
	cr := spinUntilDone(func() createResponse {
		r, e := c.cli.CreateContainerFromConfig(containerConfig)
		return createResponse{Resp: r, Err: e}
	}, func(frame string) {
		c.m.Lock()
		if add {
			c.lines = append(c.lines, fmt.Sprintf("%s Creating container: %s", frame, containerName))
			add = false
		} else {
			c.lines[len(c.lines)-1] = fmt.Sprintf("%s Creating container: %s", frame, containerName)
		}
		c.m.Unlock()
		c.updateChan <- ControllerUpdateMsg{}
	})

	err = cr.Err

	if err != nil {
		c.m.Lock()
		c.lines = append(c.lines, fmt.Sprintf("Error create: %v", err))
		c.m.Unlock()
		c.updateChan <- ControllerUpdateMsg{}
		return
	}

	c.m.Lock()
	c.lines[len(c.lines)-1] = fmt.Sprintf("%s Creating container: %s", style.Success().Render("✓"), containerName)
	c.m.Unlock()
	c.updateChan <- ControllerUpdateMsg{}

	add = true
	err = spinUntilDone(func() error {
		return c.cli.ContainerStart(context.Background(), cr.Resp.ID, container.StartOptions{})
	}, func(frame string) {
		c.m.Lock()
		if add {
			c.lines = append(c.lines, fmt.Sprintf("%s Starting container: %s", frame, containerName))
			add = false
		} else {
			c.lines[len(c.lines)-1] = fmt.Sprintf("%s Starting container: %s", frame, containerName)
		}
		c.m.Unlock()
		c.updateChan <- ControllerUpdateMsg{}
	})

	if err != nil {
		c.m.Lock()
		c.lines = append(c.lines, fmt.Sprintf("Error start: %v", err))
		c.m.Unlock()
		c.updateChan <- ControllerUpdateMsg{}
		return
	}

	c.m.Lock()
	c.lines[len(c.lines)-1] = fmt.Sprintf("%s Starting container: %s", style.Success().Render("✓"), containerName)
	c.m.Unlock()
	c.updateChan <- ControllerUpdateMsg{}

	c.m.Lock()
	c.lines = append(c.lines, "update complete, press enter to close...")
	c.m.Unlock()

	close(c.updateChan)
}

func spinUntilDone[T any](
	action func() T,
	updateLine func(frame string),
) T {

	done := make(chan T)

	// Lance l’action en arrière-plan
	go func() {
		done <- action()
	}()

	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	index := 0

	for {
		select {
		case r := <-done:
			// Terminé
			return r

		case <-time.After(100 * time.Millisecond):
			// Frame suivante
			frame := spinner[index]
			index = (index + 1) % len(spinner)

			updateLine(style.Spinner().Render(frame))
		}
	}
}
