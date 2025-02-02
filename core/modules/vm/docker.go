package vm

import (
	"context"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/hamster-shared/hamster-provider/core/modules/utils"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"os/exec"
	"strconv"
	"time"
)

type DockerManager struct {
	template *Template
	//访问端口
	accessPort int
	// docker 客户端
	cli *client.Client
	// context
	ctx context.Context

	// ContainerName
	containerName string
	// image
	image string

	nodePort int
}

func NewDockerManager() (*DockerManager, error) {
	cli, err := client.NewClientWithOpts(client.WithVersion("1.38"))
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &DockerManager{
		cli: cli,
		ctx: context.Background(),
	}, nil
}

func (d *DockerManager) SetTemplate(t Template) {
	d.template = &t
	d.image = t.Image
	d.containerName = t.Name
	d.accessPort = 22
	d.nodePort = utils.RandomPort()
}

func (d *DockerManager) Status() (*Status, error) {
	containers, err := d.cli.ContainerList(d.ctx, types.ContainerListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("name", d.containerName)),
	})
	if err != nil {
		return nil, err
	}
	if len(containers) == 0 {
		return &Status{}, errors.New("container not exists")
	}

	// status 状态 0: 关闭,1: 启动 , 2：暂停, 3, other
	var status int

	switch containers[0].State {
	case "created":
		status = 0
		break
	case "restarting":
		status = 3
		break
	case "running":
		status = 1
		break
	case "removing":
		status = 3
		break
	case "paused":
		status = 2
		break
	case "exited":
	case "dead":
		status = 0
		break
	}

	return &Status{
		status: status,
		id:     containers[0].ID,
	}, nil
}

// query container ip address
func (d *DockerManager) GetIp() (string, error) {
	//status, err := d.Status()
	//
	//if err != nil {
	//	return "", err
	//}
	//
	//if status.id == "" {
	//	return "", errors.New("container id cannot be empty")
	//}
	//
	//containerJson, err := d.cli.ContainerInspect(d.ctx, status.id)
	//if err != nil {
	//	return "", err
	//}
	//return containerJson.NetworkSettings.IPAddress, nil

	// macos cannot directly access container ip
	return "127.0.0.1", nil
}

func (d *DockerManager) GetAccessPort() int {
	return d.nodePort
}

func (d *DockerManager) Create() error {

	// view all images
	imageLists, err := d.cli.ImageList(d.ctx, types.ImageListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("reference", d.image)),
	})
	if err != nil {
		log.Println(err)
		return err
	}
	if len(imageLists) == 0 {
		// pull image
		out, err := d.cli.ImagePull(d.ctx, d.image, types.ImagePullOptions{})
		if err != nil {
			log.Println(err)
			return err
		}
		_, err = io.Copy(os.Stdout, out)
		if err != nil {
			return err
		}
	}

	// determine whether there is a repeated start
	status, err := d.Status()
	if err != nil {
		log.Info(err.Error())
	}

	if status.id != "" {
		err = d.cli.ContainerRemove(d.ctx, status.id, types.ContainerRemoveOptions{Force: true})
		if err != nil {
			return err
		}
	}

	port, err := nat.NewPort("tcp", strconv.Itoa(d.accessPort))
	// create a container
	resp, err := d.cli.ContainerCreate(d.ctx, &container.Config{
		Image: d.image, //image name
		//Tty:        true,
		//OpenStdin:  true,
		//Cmd:        []string{cmd},
		//WorkingDir: workDir,
		ExposedPorts: nat.PortSet{
			port: struct{}{},
		},
	},
		&container.HostConfig{

			Resources: container.Resources{
				CPUCount: int64(d.template.Cpu),
				Memory:   int64(d.template.Memory << 30),
			},
			PortBindings: nat.PortMap{
				port: []nat.PortBinding{
					{
						HostPort: strconv.Itoa(d.nodePort),
					},
				},
			},
		}, nil, nil, d.containerName)

	if err != nil {
		log.Println(err)
		return err
	}

	log.WithField("containerId", resp.ID).Info("container created")

	return nil
}

// StartContainer running containers in the background
func (d *DockerManager) Start() error {
	status, err := d.Status()
	if err != nil {
		return err
	}

	id := status.id

	if status.status == 0 && id != "" {
		err = d.cli.ContainerStart(d.ctx, id, types.ContainerStartOptions{})
		return err
	} else {
		return errors.New("container status is invalid")
	}

}

func (d *DockerManager) CreateAndStart() error {
	err := d.Create()
	if err != nil {
		return err
	}
	return d.Start()
}

func (d *DockerManager) CreateAndStartAndInjectionPublicKey(publicKey string) error {
	// create a virtual machine
	err := d.CreateAndStart()
	if err != nil {
		return err
	}
	// wait for the virtual machine to start successfully
	for {
		status, err := d.Status()
		if err != nil {
			return err
		}
		if status.IsRunning() {
			break
		}
		time.Sleep(time.Second * 3)
	}
	return d.InjectionPublicKey(publicKey)
}

func (d *DockerManager) Stop() error {
	status, err := d.Status()
	if status.status != 1 {
		return errors.New("invalid container status")
	}
	if err != nil {
		return err
	}
	id := status.id

	timeout := time.Second * 3
	if id != "" {
		return d.cli.ContainerStop(d.ctx, status.id, &timeout)
	} else {
		return errors.New("container id is invalid")
	}
}

func (d *DockerManager) Reboot() error {
	status, err := d.Status()
	if err != nil {
		return err
	}
	id := status.id

	timeout := time.Second * 3
	if id != "" {
		return d.cli.ContainerRestart(d.ctx, status.id, &timeout)
	} else {
		return errors.New("container id is invalid")
	}
}

func (d *DockerManager) Shutdown() error {
	status, err := d.Status()
	if status.status != 1 {
		return errors.New("invalid container status")
	}
	if err != nil {
		return err
	}
	id := status.id
	timeout := time.Second * 3
	if id != "" {
		return d.cli.ContainerStop(d.ctx, status.id, &timeout)
	} else {
		return errors.New("container id is invalid")
	}
}

// Destroy delete container
func (d *DockerManager) Destroy() error {
	status, err := d.Status()
	if err != nil {
		return err
	}
	id := status.id

	if id != "" {
		return d.cli.ContainerRemove(d.ctx, status.id, types.ContainerRemoveOptions{Force: true})
	} else {
		return errors.New("container id is invalid")
	}
}

// InjectionPublicKey add the public key to the container
func (d *DockerManager) InjectionPublicKey(publicKey string) error {

	status, err := d.Status()
	if !status.IsRunning() {
		return errors.New("invalid container status")
	}
	if err != nil {
		return err
	}
	id := status.id
	if id != "" {
		cmd := fmt.Sprintf("echo %s  > /root/.ssh/authorized_keys", publicKey)
		command := exec.Command("docker", "exec", id, "bash", "-c", cmd)
		return command.Run()
	} else {
		return errors.New("container status is invalid")
	}
}
