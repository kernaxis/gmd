// Package cachalot wraps the official Docker client (github.com/docker/docker/client).
//
// A cachalot.Client embeds client.APIClient, so it can be used anywhere the
// official client is expected: every method of the official API is still
// available and goes straight to the Docker socket.
//
// On top of that, NewClient immediately snapshots containers and images and
// subscribes to the Docker events stream to keep that snapshot up to date in
// memory. Callers that need to poll Docker state frequently should read it
// through Containers/Container/Images/Image instead of hitting the API
// directly, so the Docker socket only sees the initial snapshot plus the
// events stream, not one request per poll.
package cachalot
