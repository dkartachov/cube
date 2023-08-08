package task

import (
	"context"
	"io"
	"log"
	"os"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
)

type Task struct {
	ID            uuid.UUID
	Name          string
	State         State
	Image         string
	Memory        int
	Disk          int
	ExposedPorts  nat.PortSet
	PortBindings  map[string]string
	RestartPolicy string
	StartTime     time.Time
	FinishTime    time.Time
	ContainerId   string
}

type TaskEvent struct {
	ID        uuid.UUID
	State     State
	Timestamp time.Time
	Task      Task
}

type Config struct {
	Name          string
	AttachStdin   bool
	AttachStdout  bool
	AttachStderr  bool
	Cmd           []string
	Image         string
	Memory        int64
	Disk          int64
	Env           []string
	RestartPolicy string
}

// TODO Should ideally create some kind of ContainerRuntime interface to decouple
// the orchestrator from the container software. Right now it's heavily dependent on Docker.
type Docker struct {
	Client      *client.Client
	Config      Config
	ContainerId string
}

type DockerResult struct {
	Error       error
	Action      string
	ContainerId string
	Result      string
}

func (t *Task) NewConfig() Config {
	config := Config{
		Name:          t.Name,
		Image:         t.Image,
		RestartPolicy: t.RestartPolicy,
		// more stuff?
	}

	return config
}

func (t *Task) NewDocker(c Config) Docker {
	dc, _ := client.NewClientWithOpts(client.FromEnv)
	docker := Docker{
		Client: dc,
		Config: c,
	}

	return docker
}

func (d *Docker) Run() DockerResult {
	ctx := context.Background()
	reader, err := d.Client.ImagePull(ctx, d.Config.Image, types.ImagePullOptions{})

	if err != nil {
		log.Printf("[docker] error pulling image %s: %v", d.Config.Image, err)

		return DockerResult{Error: err}
	}

	io.Copy(os.Stdout, reader)

	rp := container.RestartPolicy{Name: d.Config.RestartPolicy}
	r := container.Resources{Memory: d.Config.Memory}
	cc := container.Config{
		Image: d.Config.Image,
		Env:   d.Config.Env,
	}
	hc := container.HostConfig{
		RestartPolicy:   rp,
		Resources:       r,
		PublishAllPorts: true,
	}

	resp, err := d.Client.ContainerCreate(ctx, &cc, &hc, nil, nil, d.Config.Name)

	if err != nil {
		log.Printf("[docker] error creating container using image %s: %v", d.Config.Image, err)

		return DockerResult{Error: err}
	}

	err = d.Client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})

	if err != nil {
		log.Printf("[docker] error starting container %s: %v", resp.ID, err)

		return DockerResult{Error: err}
	}

	d.ContainerId = resp.ID

	out, err := d.Client.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})

	if err != nil {
		log.Printf("[docker] error getting logs for container %s: %v", resp.ID, err)

		return DockerResult{Error: err}
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)

	return DockerResult{
		ContainerId: resp.ID,
		Action:      "start",
		Result:      "success",
	}
}

func (d *Docker) Stop() DockerResult {
	id := d.ContainerId

	log.Printf("[docker] attempting to stop container %s", id)

	ctx := context.Background()
	err := d.Client.ContainerStop(ctx, id, container.StopOptions{})

	if err != nil {
		log.Printf("[docker] error stopping container %s: %v", id, err)

		return DockerResult{Error: err}
	}

	err = d.Client.ContainerRemove(ctx, id, types.ContainerRemoveOptions{RemoveVolumes: true, RemoveLinks: false, Force: false})

	if err != nil {
		log.Printf("[docker] error removing container %s: %v", id, err)

		return DockerResult{Error: err}
	}

	return DockerResult{
		ContainerId: id,
		Action:      "stop",
		Result:      "success",
	}
}
