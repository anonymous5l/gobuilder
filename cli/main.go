package main

import (
	"fmt"
	"gobuilder/cli/env"
)

func main() {
	fmt.Println("Hello World", env.Version, env.BuildStamp, env.BuildTool, env.GitHash)
}
