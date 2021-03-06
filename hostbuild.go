package main

import (
	"fmt"
	"gobuilder/log"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func GitInfo(pkg string) (string, string) {
	// git info
	var (
		gitBranch    string
		gitShortHash string
	)
	gitCommand := NewGitCommand("rev-parse", "--abbrev-ref", "HEAD")
	if err := gitCommand.Start(); err == nil {
		if err := gitCommand.Wait(); err != nil {
			// ignore git command error
			log.Warn("package", pkg, "resolve git branch failed", err)
		} else {
			gitBranch = strings.TrimSpace(string(gitCommand.Stdout()))
		}
	}

	gitShortHashCommand := NewGitCommand("rev-parse", "--verify", "--short", "HEAD")
	if err := gitShortHashCommand.Start(); err == nil {
		if err := gitShortHashCommand.Wait(); err != nil {
			log.Warn("package", pkg, "resolve git hash failed", err)
		} else {
			gitShortHash = strings.TrimSpace(string(gitShortHashCommand.Stdout()))
		}
	}

	return gitBranch, gitShortHash
}

func GoBuildArgs(gitBranch, gitShortHash, goVersion, name string, pkg *GoBuilderPackage) []string {
	var args []string

	var ldflags []string
	if pkg.VerbosePackage != "" {
		strTime := time.Now().Format(time.RFC3339)
		ldflags = append(ldflags,
			"-w",
			"-X", "'"+pkg.VerbosePackage+".Version="+pkg.Version.String()+"'",
			"-X", "'"+pkg.VerbosePackage+".BuildStamp="+strTime+"'",
			"-X", "'"+pkg.VerbosePackage+".BuildTool=gobuilder/"+
				goVersion+
				"/"+pkg.BuildMode+
				"/"+runtime.GOOS+
				"/"+runtime.GOARCH+"'",
		)

		if gitShortHash != "" {
			ldflags = append(ldflags, "-X", "'"+pkg.VerbosePackage+".GitHash="+gitShortHash+
				"/"+gitBranch+"'")
		}
		args = append(args, "-ldflags="+strings.Join(ldflags, " "))
	}

	for i := 0; i < len(pkg.BuildFlag); i++ {
		args = append(args, pkg.BuildFlag[i])
	}

	if pkg.Dest != "" {
		args = append(args, "-o", filepath.Join(pkg.Dest, name))
	}

	return append(args, pkg.Package)
}

func HostBuild(name string, pkg *GoBuilderPackage) error {
	gitBranch, gitShortHash := GitInfo(pkg.Package)

	goVersion := strings.TrimPrefix(runtime.Version(), "go")
	if goVersion != BuildConfig.Version {
		log.Warn(fmt.Sprintf("host go version not match config version %s<->%s", goVersion, BuildConfig.Version))
		BuildConfig.Version = goVersion
	}

	// running host go command
	cmd := NewGoCommand()
	if pkg.BuildOS != "" {
		cmd.SetEnv("GOOS", pkg.BuildOS)
	}
	if pkg.BuildArch != "" {
		cmd.SetEnv("GOARCH", pkg.BuildArch)
	}

	args := GoBuildArgs(gitBranch, gitShortHash, goVersion, name, pkg)
	cmd.AppendArgs("build").
		AppendArgs(args...)

	log.Debug("build command", cmd.String(), "-", name)

	if err := cmd.Start(); err != nil {
		return err
	}

	return cmd.Wait()
}
