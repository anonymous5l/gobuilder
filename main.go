package main

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
	"sync"
)

func motd() {
	fmt.Println("\u001B[32mGolang build tool\u001B[0m")
}

func GoBuild(name string, pkg *GoBuilderPackage) error {
	// check project exists
	if pkg.BuildMode == "host" {
		return HostBuild(name, pkg)
	} else if pkg.BuildMode == "docker" {
		return DockerBuild(name, pkg)
	}

	return errors.New("invalid `build-mode`")
}

var (
	readEnvOnce sync.Once
	HostGoEnv   map[string]string
	BuildConfig GoBuilderConfig
)

func init() {
	readEnvOnce.Do(func() {
		// read go env
		cmd := NewGoCommand("env", "-json")
		if err := cmd.Start(); err != nil {
			Error("start process failed", err)
			return
		}
		if err := cmd.Wait(); err != nil {
			Error("exec process failed", err)
			return
		}

		if err := cmd.JSONStdout(&HostGoEnv); err != nil {
			Error("unmarshal json failed", err)
			return
		}
	})
}

type Task struct {
	Name    string
	Package *GoBuilderPackage
}

func main() {
	motd()

	o, err := os.Open(".gobuilder")
	if err != nil {
		Error("`.gobuilder` invalid")
		return
	}

	err = yaml.NewDecoder(o).Decode(&BuildConfig)
	if err != nil {
		Error("yaml decode file failed", err)
		return
	}

	if err := o.Close(); err != nil {
		Error("close file failed", err)
		return
	}

	commands := os.Args[1:]

	Log("GoHostEnv")
	Log("  Version:", strings.TrimPrefix(HostGoEnv["GOVERSION"], "go"))
	Log("  OS/ARCH:", HostGoEnv["GOHOSTOS"]+"/"+HostGoEnv["GOHOSTARCH"])

	parallel := 1
	if BuildConfig.Parallel > 1 {
		parallel = BuildConfig.Parallel
	}

	wg := sync.WaitGroup{}
	parallelWaitGroup := sync.WaitGroup{}
	parallelWaitGroup.Add(parallel)

	taskQueue := make(chan Task, parallel)

	for i := 0; i < parallel; i++ {
		go func() {
			for {
				t, ok := <-taskQueue
				if !ok {
					parallelWaitGroup.Done()
					return
				}
				if err := GoBuild(t.Name, t.Package); err != nil {
					Error("build package `"+t.Name+"` failed", err)
				} else {
					Ok("`" + t.Name + "` build completed")
				}
				if t.Package.Version != nil {
					t.Package.Version.Patch += 1
				}
				wg.Done()
			}
		}()
	}

	if len(commands) > 0 {
		for _, n := range commands {
			for pName, pkg := range BuildConfig.Packages {
				if pName == n {
					taskQueue <- Task{Name: pName, Package: pkg}
					wg.Add(1)
					break
				}
			}
		}
	} else {
		// start build all command
		for k, v := range BuildConfig.Packages {
			taskQueue <- Task{Name: k, Package: v}
			wg.Add(1)
		}
	}

	wg.Wait()
	close(taskQueue)
	parallelWaitGroup.Wait()

	// clean up

	if BuildConfig.AutoUpgrade {
		configBytes, err := yaml.Marshal(BuildConfig)
		if err != nil {
			Error("marshal config failed", err)
			return
		}
		o, err := os.Create(".gobuilder")
		if err != nil {
			Error("create config failed", err)
			return
		}
		defer o.Close()
		if _, err := o.Write(configBytes); err != nil {
			Error("write config failed", err)
			return
		}
	}
}
