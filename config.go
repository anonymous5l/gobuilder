package main

import "fmt"

type Version struct {
	Major int `json:"major"`
	Minor int `json:"minor"`
	Patch int `json:"patch"`
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

type GoBuilderPackage struct {
	Package        string   `yaml:"package"`
	VerbosePackage string   `yaml:"verbose-package"`
	BuildFlag      []string `yaml:"build-flag"` // suffix flag
	BuildMode      string   `yaml:"build-mode"` // host or docker
	BuildOS        string   `yaml:"build-os"`   // darwin or linux or windows
	BuildArch      string   `yaml:"build-arch"` // arm64 or amd64 or ...
	Version        *Version `yaml:"version"`
	Dest           string   `yaml:"dest"`
}

type GoBuilderConfig struct {
	Packages    map[string]*GoBuilderPackage `yaml:"packages"`
	Version     string                       `yaml:"version"` // golang version only build mode docker working
	Parallel    int                          `yaml:"parallel"`
	AutoUpgrade bool                         `yaml:"auto-upgrade"`
}
