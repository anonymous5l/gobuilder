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
	"sync"
)

var ServerConfig *GoBuilderServerConfig

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
	configPath := "server.yaml"

	if len(commands) > 0 {
		switch commands[0] {
		case "keygen":
			if err := GenerateCertAndKey(); err != nil {
				log.Error("generate failed", err)
			}
			return
		default:
			configPath = commands[0]
		}
	}

	config, err := readConfig(configPath)
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

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
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

	wg.Wait()
}
