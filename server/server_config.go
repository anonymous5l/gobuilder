package main

import (
	"crypto/tls"
	"io/ioutil"
	"os"
)

type GoBuilderServerPackage struct {
	Env          map[string]string `yaml:"env"`
	BeforeAction string            `yaml:"before-action"`
	Perm         os.FileMode       `yaml:"perm,omitempty"`
	Executable   string            `yaml:"executable"`
	AfterAction  string            `yaml:"after-action"`
}

type GoBuilderServerConfig struct {
	Packages map[string]*GoBuilderServerPackage `yaml:"packages"`
	Address  string                             `yaml:"address"`
	CA       string                             `yaml:"ca"`
	Cert     string                             `yaml:"cert"`
	Key      string                             `yaml:"key"`
	Handler  int                                `yaml:"handler"`
}

func (c GoBuilderServerConfig) GetTlsCert() (tls.Certificate, error) {
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
