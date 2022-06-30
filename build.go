package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"github.com/lucas-clemente/quic-go"
	"gobuilder/log"
	"gobuilder/quicpkg"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

func GoBuild(name string, pkg *GoBuilderPackage) error {
	// check project exists
	if pkg.BuildMode == "host" {
		return HostBuild(name, pkg)
	} else if pkg.BuildMode == "docker" {
		return DockerBuild(name, pkg)
	}

	return errors.New("invalid `build-mode`")
}

func ProcessTask(wg *sync.WaitGroup, t Task) error {
	defer wg.Done()
	if err := GoBuild(t.Name, t.Package); err != nil {
		return err
	}

	log.Ok("`" + t.Name + "` build completed")
	if t.Package.Version != nil {
		t.Package.Version.Patch += 1
	}

	// try push deploy
	if t.Package.Deploy == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	tlsCert, err := BuildConfig.GetTlsCert()
	if err != nil {
		return err
	}

	pem, err := ioutil.ReadFile(BuildConfig.CA)
	if err != nil {
		return err
	}
	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(pem)

	tlsConfig := &tls.Config{
		RootCAs:      caPool,
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"gobuilder-quic"},
		ServerName:   "gobuilder-quic",
	}

	remote, err := quic.DialAddrContext(ctx, t.Package.Deploy, tlsConfig, nil)
	if err != nil {
		return err
	}

	// calc binary sha256

	binaryPath := filepath.Join(t.Package.Dest, t.Name)

	o, err := os.Open(binaryPath)
	if err != nil {
		return err
	}
	defer o.Close()

	fileBuffer := bytes.NewBuffer([]byte{})
	hashChunk := make([]byte, 1024)
	hashFunc := sha256.New()
	for {
		chunkLen, err := o.Read(hashChunk)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		chunk := hashChunk[:chunkLen]
		fileBuffer.Write(chunk)
		hashFunc.Write(chunk)
	}
	hashSum := hashFunc.Sum(nil)

	stream, err := remote.OpenStream()
	if err != nil {
		return err
	}

	fileBytes := fileBuffer.Bytes()

	request := quicpkg.PacketPackageReplace{
		PacketPackageName: quicpkg.PacketPackageName{
			Package: quicpkg.Data[uint16, string]{
				Size: uint16(len(t.Name)),
				Data: t.Name,
			},
		},
		PacketPackage: quicpkg.PacketPackage{
			Signature: quicpkg.Data[uint8, []byte]{
				Size: uint8(len(hashSum)),
				Data: hashSum,
			},
			Data: quicpkg.Data[uint64, []byte]{
				Size: uint64(len(fileBytes)),
				Data: fileBytes,
			},
		},
	}

	if err := request.WriteWithOp(stream); err != nil {
		return err
	}

	var op byte
	if err := quicpkg.Read(stream, &op); err != nil {
		return err
	}

	if op == quicpkg.OperationPacketError {
		var pktError quicpkg.PacketErrorResponse
		if err := pktError.Read(stream); err != nil {
			return err
		}
		log.Error("`"+t.Name+"` deploy failed",
			"["+strconv.FormatUint(uint64(pktError.ErrCode), 10)+"]",
			pktError.ErrMessage.Data)
		return nil
	}

	var response quicpkg.PacketPackageReplaceResponse
	if err := response.Read(stream); err != nil {
		return err
	}

	logOutput := []any{"`" + t.Name + "` deploy completed"}
	if response.BeforeStdout.Size > 0 {
		logOutput = append(logOutput, "\n  - before output", strings.TrimSpace(response.BeforeStdout.Data))
	}
	if response.AfterStdout.Size > 0 {
		logOutput = append(logOutput, "\n  -  after output", strings.TrimSpace(response.AfterStdout.Data))
	}

	log.Ok(logOutput...)

	if t.Package.CleanAfterDeploy {
		if err := os.RemoveAll(binaryPath); err != nil {
			return err
		}
	}

	return nil
}
