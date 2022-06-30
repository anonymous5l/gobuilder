package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
)

type Version struct {
	Major int `json:"major"`
	Minor int `json:"minor"`
	Patch int `json:"patch"`
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

type GoBuilderPackage struct {
	Package          string   `yaml:"package"`
	VerbosePackage   string   `yaml:"verbose-package"`
	BuildFlag        []string `yaml:"build-flag,omitempty"` // suffix flag
	BuildMode        string   `yaml:"build-mode"`           // host or docker
	BuildOS          string   `yaml:"build-os,omitempty"`   // darwin or linux or windows
	BuildArch        string   `yaml:"build-arch,omitempty"` // arm64 or amd64 or ...
	Version          *Version `yaml:"version,omitempty"`
	Dest             string   `yaml:"dest,omitempty"`
	Deploy           string   `yaml:"deploy,omitempty"` // remote quic path
	CleanAfterDeploy bool     `yaml:"clean-after-deploy,omitempty"`
}

type GoBuilderConfig struct {
	Packages    map[string]*GoBuilderPackage `yaml:"packages,omitempty"`
	Version     string                       `yaml:"version,omitempty"` // golang version only build mode docker working
	Parallel    int                          `yaml:"parallel,omitempty"`
	AutoUpgrade bool                         `yaml:"auto-upgrade"`
	CA          string                       `yaml:"ca,omitempty"`
	Cert        string                       `yaml:"cert,omitempty"`
	Key         string                       `yaml:"key,omitempty"`
}

func (c GoBuilderConfig) GetTlsCert() (tls.Certificate, error) {
	certPEM, err := ioutil.ReadFile(c.Cert)
	if err != nil {
		return tls.Certificate{}, err
	}
	privateKeyPEM, err := ioutil.ReadFile(c.Key)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.X509KeyPair(certPEM, privateKeyPEM)
}
