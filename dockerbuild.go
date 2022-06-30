package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/k0kubun/go-ansi"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/schollz/progressbar/v3"
	"gobuilder/log"
	"io"
	"math"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func getImage(dockerApi *client.Client, goOs, goArch, goVersion string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	images, err := dockerApi.ImageList(ctx, types.ImageListOptions{
		All: true,
	})
	if err != nil {
		return "", err
	}

	for i := 0; i < len(images); i++ {
		img := images[i]
		inspect, _, err := dockerApi.ImageInspectWithRaw(ctx, img.ID)
		if err != nil {
			return "", err
		}
		if len(inspect.RepoTags) > 0 && strings.HasPrefix(inspect.RepoTags[0], "golang:"+goVersion) {
			if inspect.Os == goOs && inspect.Architecture == goArch {
				return img.ID, nil
			}
		}
	}

	return "", nil
}

type ProgressDetail struct {
	Current int64 `json:"current"`
	Total   int64 `json:"total"`
}

type PullImageEvent struct {
	Status         string         `json:"status"`
	ProgressDetail ProgressDetail `json:"progressDetail"`
	Id             string         `json:"id"`
}

type ImageFSLayerStatus struct {
	Current int64
	Total   int64
	Flag    int
}

type GoModule struct {
	Path      string `json:"Path"`
	Main      bool   `json:"Main"`
	Dir       string `json:"Dir"`
	GoMod     string `json:"GoMod"`
	GoVersion string `json:"GoVersion"`
}

func ReadDockerLogs(r io.ReadCloser) (int32, []byte, error) {
	var (
		t    int32
		size int32
	)

	if err := binary.Read(r, binary.BigEndian, &t); err != nil {
		return 0, nil, err
	}

	if err := binary.Read(r, binary.BigEndian, &size); err != nil {
		return 0, nil, err
	}

	buf := make([]byte, size)
	_, err := r.Read(buf)
	if err != nil {
		return 0, nil, err
	}

	return t, buf, nil
}

func DockerBuild(name string, pkg *GoBuilderPackage) error {
	gitBranch, gitShortHash := GitInfo(pkg.Package)

	// use moby api interface
	dockerApi, err := client.NewClientWithOpts()
	if err != nil {
		return err
	}

	if err := client.FromEnv(dockerApi); err != nil {
		return err
	}

	goOs := pkg.BuildOS
	if goOs == "" {
		goOs = HostGoEnv["GOHOSTOS"]
	}

	goArch := pkg.BuildArch
	if goArch == "" {
		goArch = HostGoEnv["GOHOSTARCH"]
	}

	goVersion := BuildConfig.Version
	if goVersion == "" {
		goVersion = "latest"
	}

	imageId, err := getImage(dockerApi, goOs, goArch, goVersion)
	if err != nil {
		return err
	}

	if imageId == "" {
		pullResponse, err := dockerApi.ImagePull(context.Background(), "golang:"+goVersion, types.ImagePullOptions{
			Platform: pkg.BuildOS + "/" + pkg.BuildArch,
		})
		if err != nil {
			return err
		}
		defer pullResponse.Close()

		log.Log("Pulling docker image", pkg.BuildOS+"/"+pkg.BuildArch, "...")

		decoder := json.NewDecoder(pullResponse)

		commonProgressBar := progressbar.NewOptions(100,
			progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
			progressbar.OptionEnableColorCodes(true),
			progressbar.OptionSetWidth(30),
			progressbar.OptionSetPredictTime(false),
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        "-",
				SaucerHead:    ">",
				SaucerPadding: " ",
				BarStart:      "[",
				BarEnd:        "]",
			}))

		progress := make(map[string]*ImageFSLayerStatus)

		for {
			var event PullImageEvent
			if err := decoder.Decode(&event); err != nil {
				if err == io.EOF {
					break
				}
				return err
			}

			if event.Status == "Pulling fs layer" {
				progress[event.Id] = &ImageFSLayerStatus{Flag: 0}
			}

			if event.Status == "Waiting" {
				progress[event.Id].Flag = 1
			}

			if event.Status == "Downloading" {
				p := progress[event.Id]
				p.Flag = 2
				p.Current = event.ProgressDetail.Current
				p.Total = event.ProgressDetail.Total

				var (
					allTotal      float64
					allCurrent    float64
					validCount    int
					completeCount int
				)

				for _, v := range progress {
					allTotal += float64(v.Total)
					allCurrent += float64(v.Current)
					validCount++
					if v.Flag > 2 {
						completeCount++
					}
				}

				commonProgressBar.Describe(
					fmt.Sprintf("[cyan][%d/%d][reset] [light_green]Pulling fs layer ...[reset]", completeCount, validCount))

				percent := (allCurrent / allTotal) * 100.0

				if err := commonProgressBar.Set(int(math.Round(percent))); err != nil {
					return err
				}
			}

			if event.Status == "Extracting" {
				p := progress[event.Id]
				p.Flag = 3
				p.Current = event.ProgressDetail.Current
				p.Total = event.ProgressDetail.Total

				var (
					allTotal      float64
					allCurrent    float64
					validCount    int
					completeCount int
				)

				for _, v := range progress {
					allTotal += float64(v.Total)
					allCurrent += float64(v.Current)
					if v.Total == v.Current {
						completeCount++
					}
					validCount++
				}

				commonProgressBar.Describe(
					fmt.Sprintf("[cyan][%d/%d][reset] [light_green]Extracting ...[reset]", completeCount, validCount))

				percent := (allCurrent / allTotal) * 100.0

				if err := commonProgressBar.Set(int(math.Round(percent))); err != nil {
					return err
				}
			}

			if event.Status == "Pull complete" {
				p := progress[event.Id]
				p.Flag = 4
			}

			if strings.HasPrefix(event.Status, "Digest") {
				imageId = strings.Split(event.Status, ":")[2]
			}
		}

		if err := commonProgressBar.Close(); err != nil {
			return err
		}

		imageId, err = getImage(dockerApi, goOs, goArch, goVersion)
		if err != nil {
			return err
		}
	}

	if imageId == "" {
		return fmt.Errorf("please manual download image `docker pull --platform %s/%s golang:go%s`",
			goOs, goArch, goVersion)
	}

	listCommand := NewGoCommand("list", "-m", "-json")
	if err := listCommand.Start(); err != nil {
		return err
	}
	if err := listCommand.Wait(); err != nil {
		return err
	}

	var mod GoModule
	if err := listCommand.JSONStdout(&mod); err != nil {
		return err
	}

	labels := make(map[string]string)
	labels["gobuilder"] = runtime.Version()

	projectDir := "/go/" + filepath.Base(mod.Dir)

	containerConfig := &container.Config{
		Hostname:   "gobuilder",
		User:       "root",
		Image:      imageId,
		WorkingDir: projectDir,
		Entrypoint: strslice.StrSlice{"go"},
		Cmd: append(strslice.StrSlice{"build"},
			GoBuildArgs(gitBranch, gitShortHash, BuildConfig.Version, name, pkg)...),
		Labels: labels,
		Env: []string{
			"GIT_BRANCH=" + gitBranch,
			"GIT_HASH=" + gitShortHash,
			"GOOS=" + goOs,
			"GOARCH=" + goArch,
		},
	}
	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   "bind",
				Source: mod.Dir,
				Target: projectDir,
			},
		},
	}
	networkingConfig := &network.NetworkingConfig{}
	platform := &specs.Platform{
		Architecture: goArch,
		OS:           goOs,
	}

	resp, err := dockerApi.ContainerCreate(context.Background(),
		containerConfig,
		hostConfig,
		networkingConfig,
		platform,
		fmt.Sprintf("gobuilder-%s-%s-%s", goOs, goArch, name))
	if err != nil {
		return err
	}

	if err := dockerApi.ContainerStart(context.Background(), resp.ID,
		types.ContainerStartOptions{}); err != nil {
		return err
	}

	var exitCode int

	for {
		c, err := dockerApi.ContainerInspect(context.Background(), resp.ID)
		if err != nil {
			return err
		}

		if !c.State.Running {
			exitCode = c.State.ExitCode
			break
		}

		time.Sleep(time.Second)
	}

	defer func() {
		_ = dockerApi.ContainerRemove(context.Background(), resp.ID, types.ContainerRemoveOptions{
			RemoveVolumes: true,
		})
	}()

	if exitCode != 0 {
		logs, err := dockerApi.ContainerLogs(context.Background(), resp.ID, types.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
		})
		if err != nil {
			return err
		}
		defer logs.Close()

		str := bytes.NewBufferString("")

		for {
			_, logs, err := ReadDockerLogs(logs)
			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}
			str.Write(logs)
		}

		return errors.New(strings.TrimSpace(str.String()))
	}

	return nil
}
