package main

import (
	"context"
	"github.com/lucas-clemente/quic-go"
	"gobuilder/log"
	"gobuilder/quicpkg"
	"time"
)

func QUICConnectionIncoming(conn quic.Connection) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	stream, err := conn.AcceptStream(ctx)
	if err != nil {
		return err
	}

	var rawOp byte
	if err := quicpkg.Read[byte](stream, &rawOp); err != nil {
		return err
	}
	op := quicpkg.Operation(rawOp)

	switch op {
	case quicpkg.OperationPackageInfo:
		err = HandlePackageInfoCommand(stream)
	case quicpkg.OperationPackageGet:
		err = HandlePackageGetCommand(stream)
	case quicpkg.OperationPacketReplace:
		err = HandlePackageReplaceCommand(stream)
	}

	if err != nil {
		log.Error("handle `"+op.String()+"` error", err)
		resp, err := quicpkg.NewErrorPacket(quicpkg.ErrorCodeSystem, err.Error())
		if err != nil {
			return err
		}
		if err := resp.WriteWithOp(stream); err != nil {
			return err
		}
	}

	return nil
}
