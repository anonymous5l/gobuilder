package main

import (
	"encoding/hex"
	"errors"
	"github.com/lucas-clemente/quic-go"
	"gobuilder/quicpkg"
	"os"
	"os/exec"
	"strings"
)

func ExecAction(action string, name string, config *GoBuilderServerPackage, request quicpkg.PacketPackageReplace) ([]byte, error) {
	if action == "" {
		return nil, nil
	}

	args := strings.Split(action, " ")
	if len(args) == 0 {
		return nil, nil
	}

	command := exec.Command(args[0], args[1:]...)
	command.Env = append(os.Environ(),
		"PACKAGE_NAME="+name,
		"PACKAGE_HASH="+hex.EncodeToString(request.Signature.Data),
		"PACKAGE_PATH="+config.Executable,
	)

	for k, v := range config.Env {
		command.Env = append(command.Env, k+"="+v)
	}

	return command.CombinedOutput()
}

func HandlePackageReplaceCommand(stream quic.Stream) error {
	request := quicpkg.PacketPackageReplace{}
	if err := request.Read(stream); err != nil {
		return err
	}

	pkg, ok := ServerConfig.Packages[request.Package.Data]
	if !ok {
		resp, err := quicpkg.NewErrorPacket(quicpkg.ErrorCodeNotFoundPackage,
			"package `"+request.Package.Data+"` invalid")
		if err != nil {
			return err
		}
		if err := resp.WriteWithOp(stream); err != nil {
			return err
		}
		return nil
	}

	// running command
	beforeStdout, err := ExecAction(pkg.BeforeAction, request.Package.Data, pkg, request)
	if err != nil {
		return err
	}

	filePerm := os.FileMode(0755)
	if pkg.Perm > 0 {
		filePerm = pkg.Perm
	}

	o, err := os.OpenFile(pkg.Executable, os.O_RDWR|os.O_CREATE|os.O_TRUNC, filePerm)
	if err != nil {
		return err
	}

	n, err := o.Write(request.Data.Data)
	if err != nil {
		return err
	}
	if uint64(n) != request.Data.Size {
		return errors.New("data corrupt")
	}

	if err := o.Close(); err != nil {
		return err
	}

	afterStdout, err := ExecAction(pkg.AfterAction, request.Package.Data, pkg, request)
	if err != nil {
		return err
	}

	response := quicpkg.PacketPackageReplaceResponse{
		BeforeStdout: quicpkg.Data[uint32, string]{
			Size: uint32(len(beforeStdout)),
			Data: string(beforeStdout),
		},
		AfterStdout: quicpkg.Data[uint32, string]{
			Size: uint32(len(afterStdout)),
			Data: string(afterStdout),
		},
	}

	return response.WriteWithOp(stream)
}
