// Convenience helpers built on top of the embedded official client.
//
// These are the only methods cachalot adds beyond the cache accessors: each
// one orchestrates several official calls or adds decoding/comparison logic
// that has real value on its own. Thin one-call passthroughs (starting,
// stopping, restarting or removing a container, removing an image) are
// intentionally NOT duplicated here — call the embedded client.APIClient
// methods directly (e.g. cli.ContainerStart(ctx, id, container.StartOptions{})).
package cachalot

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"runtime"
	"slices"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/hashicorp/go-version"
)

// RecreateContainer stops, removes and re-creates the container with the
// given ID using its current configuration, then starts the new container.
// It returns the ID of the newly created container.
func (c *Client) RecreateContainer(id string) (string, error) {
	ctx := context.Background()

	containerConfig, err := c.ContainerInspect(ctx, id)
	if err != nil {
		return "", fmt.Errorf("unable to recreate container %s: %w", id, err)
	}

	if err := c.ContainerStop(ctx, id, container.StopOptions{}); err != nil {
		return "", fmt.Errorf("unable to recreate container %s: %w", id, err)
	}

	if err := c.ContainerRemove(ctx, id, container.RemoveOptions{}); err != nil {
		return "", fmt.Errorf("unable to recreate container %s: %w", id, err)
	}

	r, err := c.CreateContainerFromConfig(containerConfig)
	if err != nil {
		return "", fmt.Errorf("unable to recreate container %s: %w", id, err)
	}

	if err := c.ContainerStart(ctx, r.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("unable to recreate container %s: %w", id, err)
	}

	return r.ID, nil
}

// CreateContainerFromConfig creates a container based on the given
// inspected container configuration, sanitizing it for compatibility with
// the daemon's API version first.
func (c *Client) CreateContainerFromConfig(config container.InspectResponse) (container.CreateResponse, error) {
	ctx := context.Background()

	info, err := c.ServerVersion(ctx)
	if err != nil {
		return container.CreateResponse{}, fmt.Errorf("get docker version: %w", err)
	}

	sanitizeContainerJSONVersion(&config, info.APIVersion)

	netConfig := &network.NetworkingConfig{
		EndpointsConfig: config.NetworkSettings.Networks,
	}

	return c.ContainerCreate(ctx, config.Config, config.HostConfig, netConfig, nil, config.Name)
}

func sanitizeContainerJSONVersion(containerJSON *container.InspectResponse, apiVersionString string) {
	apiVersion, err := version.NewVersion(apiVersionString)
	if err != nil {
		log.Printf("cachalot: parse docker API version %q: %v", apiVersionString, err)
		return
	}

	if apiVersion.LessThan(version.Must(version.NewVersion("1.44"))) {
		for netName, netConf := range containerJSON.NetworkSettings.Networks {
			netConf.MacAddress = ""
			containerJSON.NetworkSettings.Networks[netName] = netConf
		}
	}

	if containerJSON.HostConfig.NetworkMode == "host" || strings.HasPrefix(string(containerJSON.HostConfig.NetworkMode), "container:") {
		containerJSON.Config.Hostname = ""
		containerJSON.HostConfig.PortBindings = nil
		containerJSON.Config.ExposedPorts = nil
		containerJSON.HostConfig.PublishAllPorts = false
	}
}

// ContainerStats returns a single-shot snapshot of the stats of the
// container with the given ID, decoded from the official streaming
// response.
func (c *Client) ContainerStats(id string) (container.StatsResponse, error) {
	var v container.StatsResponse

	stats, err := c.ContainerStatsOneShot(context.Background(), id)
	if err != nil {
		return v, err
	}
	defer func() { _ = stats.Body.Close() }()

	err = json.NewDecoder(stats.Body).Decode(&v)
	return v, err
}

// PullImageWithProgress pulls an image from its registry, invoking progress
// for every JSON progress message the daemon streams back.
func (c *Client) PullImageWithProgress(ctx context.Context, imageRef string, progress func(*jsonmessage.JSONMessage)) (err error) {
	reader, err := c.ImagePull(ctx, imageRef, image.PullOptions{})
	if err != nil {
		return err
	}
	defer func() {
		err = reader.Close()
	}()

	decoder := json.NewDecoder(reader)
	for decoder.More() {
		var msg jsonmessage.JSONMessage
		if err := decoder.Decode(&msg); err != nil {
			return err
		}
		progress(&msg)
	}

	return nil
}

// CheckUpdate reports whether the image backing the given container has a
// newer digest available in its remote registry.
func (c *Client) CheckUpdate(containerID string) (bool, error) {
	ctx := context.Background()

	cont, err := c.ContainerInspect(ctx, containerID)
	if err != nil {
		return false, err
	}

	localDigests, err := c.getLocalDigests(ctx, cont.Config.Image)
	if err != nil {
		return false, err
	}

	remoteDigest, err := getRemoteDigest(cont.Config.Image)
	if err != nil {
		return false, err
	}

	f := func(s string) bool {
		return strings.HasPrefix(s, remoteDigest) || strings.HasSuffix(s, remoteDigest)
	}

	return !slices.ContainsFunc(localDigests, f), nil
}

func (c *Client) getLocalDigests(ctx context.Context, imageRef string) ([]string, error) {
	imgInspect, err := c.ImageInspect(ctx, imageRef)
	if err != nil {
		return nil, err
	}
	if len(imgInspect.RepoDigests) == 0 {
		return nil, fmt.Errorf("no RepoDigests for %s", imageRef)
	}
	return imgInspect.RepoDigests, nil
}

func getRemoteDigest(imageRef string) (string, error) {
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return "", err
	}

	desc, err := remote.Head(ref, remote.WithPlatform(v1.Platform{Architecture: runtime.GOARCH, OS: runtime.GOOS}))
	if err != nil {
		return "", err
	}

	return desc.Digest.String(), nil
}
