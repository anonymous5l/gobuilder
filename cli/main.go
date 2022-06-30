package main

import (
	"fmt"
	"gobuilder/env"
)

func main() {
	fmt.Println("Hello World", env.Version, env.BuildStamp, env.BuildTool, env.GitHash)
}
