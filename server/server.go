package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/lucas-clemente/quic-go"
	"github.com/panjf2000/ants/v2"
	"gobuilder/log"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
)

var (
	ServerConfigPath string
	ServerConfig     *GoBuilderServerConfig
)

func motd() {
	fmt.Println("\u001B[32mGolang build tool server side\u001B[0m")
}

func readConfig(configPath string) (*GoBuilderServerConfig, error) {
	o, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer o.Close()

	var config GoBuilderServerConfig
	if err := yaml.NewDecoder(o).Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func main() {
	motd()

	commands := os.Args[1:]
	ServerConfigPath = "server.yaml"

	if len(commands) > 0 {
		switch commands[0] {
		case "keygen":
			if err := GenerateCertAndKey(); err != nil {
				log.Error("generate failed", err)
			}
			return
		default:
			ServerConfigPath = commands[0]
		}
	}

	config, err := readConfig(ServerConfigPath)
	if err != nil {
		log.Error("read config file failed", err)
		return
	}

	ServerConfig = config

	if ServerConfig.Address == "" {
		ServerConfig.Address = ":2030"
	}

	tlsCert, err := ServerConfig.GetTlsCert()
	if err != nil {
		log.Error("read tls cert failed", err)
		return
	}

	pem, err := ioutil.ReadFile(ServerConfig.CA)
	if err != nil {
		log.Error("read tls ca cert failed", err)
		return
	}
	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(pem)

	tlsConfig := &tls.Config{
		RootCAs:      caPool,
		ClientCAs:    caPool,
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"gobuilder-quic"},
		ServerName:   "gobuilder-quic",
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}

	quicConfig := &quic.Config{
		EnableDatagrams: true,
	}

	listener, err := quic.ListenAddr(ServerConfig.Address, tlsConfig, quicConfig)
	if err != nil {
		log.Error("listen quic at `" + ServerConfig.Address + "` failed")
		return
	}

	if ServerConfig.Handler == 0 {
		ServerConfig.Handler = 128
	}

	gPool, err := ants.NewPool(ServerConfig.Handler,
		ants.WithPanicHandler(func(i interface{}) {
			log.Error("task pool panic", i)
		}),
		ants.WithNonblocking(true),
		ants.WithPreAlloc(true),
		ants.WithMaxBlockingTasks(ServerConfig.Handler))
	if err != nil {
		log.Error("create goroutine pool failed", err)
		return
	}

	signalChan := make(chan os.Signal, 1)
	closeChan := make(chan struct{}, 1)
	signal.Notify(signalChan, syscall.SIGUSR2, syscall.SIGINT)

	go func() {
		defer close(closeChan)
		for {
			conn, err := listener.Accept(context.Background())
			if err != nil {
				log.Error("accept connection failed", err)
				return
			}
			if err := gPool.Submit(func() {
				if err := QUICConnectionIncoming(conn); err != nil {
					log.Error("handle incoming connection failed", err)
				}
			}); err != nil {
				log.Error("pool may overflow", err)
			}
		}
	}()

	for {
		select {
		case sig, ok := <-signalChan:
			if ok {
				switch sig {
				case syscall.SIGUSR2:
					config, err = readConfig(ServerConfigPath)
					if err != nil {
						log.Error("read config file failed", err)
					} else {
						ServerConfig.Packages = config.Packages
						log.Ok("reload config file")
					}
				case syscall.SIGINT:
					if err = listener.Close(); err != nil {
						log.Error("listener close failed", err)
						return
					}
				}
			}
		case _, ok := <-closeChan:
			if !ok {
				return
			}
		}
	}
}
