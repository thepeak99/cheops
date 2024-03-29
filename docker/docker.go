package docker

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/mholt/archiver"
	"github.com/moby/moby/pkg/stdcopy"
	"golang.org/x/sync/errgroup"

	log "github.com/sirupsen/logrus"
)

func streamDockerOutput(reader io.ReadCloser) error {
	scanner := bufio.NewScanner(reader)
	defer reader.Close()
	for scanner.Scan() {
		line := scanner.Bytes()

		data := make(map[string]interface{})
		err := json.Unmarshal(line, &data)
		if err != nil {
			return err
		}

		if data["stream"] != nil {
			log.Info(data["stream"])
		} else if data["status"] != nil {
			status := data["status"].(string)
			if data["id"] != nil {
				id := data["id"].(string)
				log.WithFields(log.Fields{"id": id}).Info(status)
			} else {
				log.Info(status)
			}
		} else if data["error"] != nil {
			return errors.New(data["error"].(string))
		}
	}

	return scanner.Err()
}

type logWriter struct {
	stream string
}

func (l *logWriter) Write(data []byte) (int, error) {
	log.WithFields(log.Fields{
		"stream": l.stream,
	}).Info(string(data))
	return len(data), nil
}

func RunContainer(image string, commands, env []string) error {
	log.WithFields(log.Fields{
		"Image":    image,
		"Commands": commands,
	}).Debug("Running container")

	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}

	commandsShell := []string{"/bin/sh", "-c", strings.Join(commands, ";")}

	cont, err := cli.ContainerCreate(
		context.TODO(),
		&container.Config{
			Image:        image,
			Cmd:          commandsShell,
			Tty:          false,
			AttachStdout: true,
			AttachStderr: true,
		},
		&container.HostConfig{
			AutoRemove: true,
		},
		&network.NetworkingConfig{},
		"container",
	)
	if err != nil {
		return err
	}

	err = cli.ContainerStart(context.TODO(), cont.ID, types.ContainerStartOptions{})
	if err != nil {
		return err
	}

	res, err := cli.ContainerLogs(context.TODO(), cont.ID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})

	stdoutWriter := &logWriter{stream: "stdout"}
	stderrWriter := &logWriter{stream: "stderr"}

	stdcopy.StdCopy(stdoutWriter, stderrWriter, res)

	return nil
}

func PushImage(image, credentials string) error {
	log.WithFields(log.Fields{
		"image": image,
	}).Info("Pushing image")

	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}

	credentialsEnc := base64.StdEncoding.EncodeToString([]byte(credentials))
	reader, err := cli.ImagePush(context.TODO(), image, types.ImagePushOptions{
		RegistryAuth: credentialsEnc,
	})

	if err != nil {
		return err
	}

	return streamDockerOutput(reader)
}

func BuildImage(repoPath, dockerfilePath string, tags []string, args map[string]*string) error {
	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}
	reader, writer := io.Pipe()

	g := errgroup.Group{}

	g.Go(func() error {
		info, err := cli.ImageBuild(context.TODO(), reader, types.ImageBuildOptions{
			Tags:       tags,
			Dockerfile: dockerfilePath,
			BuildArgs:  args,
			PullParent: true,
		})
		if err != nil {
			return err
		}

		return streamDockerOutput(info.Body)
	})

	g.Go(func() error {
		t := archiver.NewTar()
		t.Create(writer)

		defer func() {
			t.Close()
			writer.Close()
		}()

		return filepath.Walk(
			repoPath,
			func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				relPath, err := filepath.Rel(repoPath, path)
				if err != nil {
					return err
				}

				file, err := os.Open(path)
				if err != nil {
					return err
				}
				defer file.Close()

				err = t.Write(archiver.File{
					FileInfo: archiver.FileInfo{
						FileInfo:   info,
						CustomName: relPath,
					},
					ReadCloser: file,
				})
				if err != nil {
					return err
				}

				return nil
			},
		)
	})

	return g.Wait()
}

// func main() {
// 	err := buildImage("https://github.com/thepeak99/nomad-docker", "Dockerfile", []string{"salam"}, nil)
// 	fmt.Println(err)
// }
