package cheops

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/mholt/archiver"
)

func pushImage() {

}

func buildImage(repoPath, dockerfilePath string, tags []string, args map[string]*string) error {
	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}
	reader, writer := io.Pipe()

	errChan := make(chan error)
	doneChan := make(chan int)

	go func() {
		info, err := cli.ImageBuild(context.Background(), reader, types.ImageBuildOptions{
			Tags:       tags,
			Dockerfile: dockerfilePath,
			//		BuildArgs:  args,
		})
		if err != nil {
			errChan <- err
			return
		}

		bufReader := bufio.NewReader(info.Body)
		for {
			line, _, err := bufReader.ReadLine()
			if err != nil {
				break
			}
			data := make(map[string]string)
			err = json.Unmarshal(line, &data)
			if err != nil {
				break
			}
			fmt.Print(data["stream"])
		}

		doneChan <- 1
	}()

	go func() {
		t := archiver.NewTar()
		t.Create(writer)

		defer func() {
			t.Close()
			writer.Close()
		}()

		err = filepath.Walk(
			repoPath,
			func(path string, info os.FileInfo, err error) error {
				if info == nil {
					return errors.New("Can't walk directory")
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

				if err != nil {
					return err
				}

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

		if err != nil {
			errChan <- err
			return
		}

		doneChan <- 1
	}()

	doneCount := 0
	for {
		select {
		case err := <-errChan:
			return err
		case <-doneChan:
			doneCount++
			if doneCount == 2 {
				return nil
			}
		}
	}
}

func main() {
	err := buildImage("https://github.com/thepeak99/nomad-docker", "Dockerfile", []string{"salam"}, nil)
	fmt.Println(err)
}
