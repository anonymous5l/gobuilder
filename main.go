package main

import (
	"fmt"
	"gobuilder/log"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
	"sync"
)

func motd() {
	fmt.Println("\u001B[32mGolang build tool\u001B[0m")
	log.Log("GoHostEnv")
	log.Log("  Version:", strings.TrimPrefix(HostGoEnv["GOVERSION"], "go"))
	log.Log("  OS/ARCH:", HostGoEnv["GOHOSTOS"]+"/"+HostGoEnv["GOHOSTARCH"])
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
			log.Error("start process failed", err)
			return
		}
		if err := cmd.Wait(); err != nil {
			log.Error("exec process failed", err)
			return
		}

		if err := cmd.JSONStdout(&HostGoEnv); err != nil {
			log.Error("unmarshal json failed", err)
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

	// read config file suffix
	goBuilderEnv := os.Getenv("GOBUILDER_ENV")

	goBuilderConfigPath := ".gobuilder"
	if goBuilderEnv != "" {
		goBuilderConfigPath += "." + goBuilderEnv
	}

	o, err := os.Open(goBuilderConfigPath)
	if err != nil {
		log.Error("`" + goBuilderConfigPath + "` invalid")
		return
	}

	err = yaml.NewDecoder(o).Decode(&BuildConfig)
	if err != nil {
		log.Error("yaml decode file failed", err)
		return
	}

	if err := o.Close(); err != nil {
		log.Error("close file failed", err)
		return
	}

	commands := os.Args[1:]

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
				if err := ProcessTask(&wg, t); err != nil {
					log.Error("build package `"+t.Name+"` failed", err)
				}
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
			log.Error("marshal config failed", err)
			return
		}
		o, err := os.Create(goBuilderConfigPath)
		if err != nil {
			log.Error("create config failed", err)
			return
		}
		defer o.Close()
		if _, err := o.Write(configBytes); err != nil {
			log.Error("write config failed", err)
			return
		}
	}
}
