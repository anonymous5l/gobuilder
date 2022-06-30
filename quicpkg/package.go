package quicpkg

import "io"

type PacketPackageName struct {
	Package Data[uint16, string]
}

func (p *PacketPackageName) Read(stream io.Reader) error {
	if err := ReadData(stream, &p.Package); err != nil {
		return err
	}
	return nil
}
func (p *PacketPackageName) Write(stream io.Writer) error {
	if err := WriteData(stream, p.Package); err != nil {
		return err
	}
	return nil
}
func (p *PacketPackageName) WriteWithOp(stream io.Writer) error {
	if err := Write[byte](stream, OperationPackageInfo); err != nil {
		return err
	}
	return p.Write(stream)
}

type PacketPackageReplace struct {
	PacketPackageName
	PacketPackage
}

func (p *PacketPackageReplace) Read(stream io.Reader) error {
	if err := p.PacketPackageName.Read(stream); err != nil {
		return err
	}
	if err := p.PacketPackage.Read(stream); err != nil {
		return err
	}
	return nil
}
func (p *PacketPackageReplace) Write(stream io.Writer) error {
	if err := p.PacketPackageName.Write(stream); err != nil {
		return err
	}
	if err := p.PacketPackage.Write(stream); err != nil {
		return err
	}
	return nil
}
func (p *PacketPackageReplace) WriteWithOp(stream io.Writer) error {
	if err := Write[byte](stream, OperationPacketReplace); err != nil {
		return err
	}
	return p.Write(stream)
}

type PacketPackageInfo struct {
	BinarySize uint64
	Signature  Data[uint8, []byte]
}

func (p *PacketPackageInfo) Read(stream io.Reader) error {
	if err := Read(stream, &p.BinarySize); err != nil {
		return err
	}
	if err := ReadData(stream, &p.Signature); err != nil {
		return err
	}

	return nil
}
func (p *PacketPackageInfo) Write(stream io.Writer) error {
	if err := Write(stream, p.BinarySize); err != nil {
		return err
	}
	if err := WriteData(stream, p.Signature); err != nil {
		return err
	}
	return nil
}
func (p *PacketPackageInfo) WriteWithOp(stream io.Writer) error {
	if err := Write[byte](stream, OperationPackageInfo); err != nil {
		return err
	}
	return p.Write(stream)
}

type PacketPackage struct {
	Signature Data[uint8, []byte]
	Data      Data[uint64, []byte]
}

func (p *PacketPackage) Read(stream io.Reader) error {
	if err := ReadData(stream, &p.Signature); err != nil {
		return err
	}
	if err := ReadData(stream, &p.Data); err != nil {
		return err
	}
	return nil
}
func (p *PacketPackage) Write(stream io.Writer) error {
	if err := WriteData(stream, p.Signature); err != nil {
		return err
	}
	if err := WriteData(stream, p.Data); err != nil {
		return err
	}
	return nil
}
func (p *PacketPackage) WriteWithOp(stream io.Writer) error {
	if err := Write[byte](stream, OperationPackageGet); err != nil {
		return err
	}
	return p.Write(stream)
}

type PacketPackageReplaceResponse struct {
	BeforeStdout Data[uint32, string]
	AfterStdout  Data[uint32, string]
}

func (p *PacketPackageReplaceResponse) Read(stream io.Reader) error {
	if err := ReadData(stream, &p.BeforeStdout); err != nil {
		return err
	}
	if err := ReadData(stream, &p.AfterStdout); err != nil {
		return err
	}
	return nil
}
func (p *PacketPackageReplaceResponse) Write(stream io.Writer) error {
	if err := WriteData(stream, p.BeforeStdout); err != nil {
		return err
	}
	if err := WriteData(stream, p.AfterStdout); err != nil {
		return err
	}
	return nil
}
func (p *PacketPackageReplaceResponse) WriteWithOp(stream io.Writer) error {
	if err := Write[byte](stream, OperationPacketReplace); err != nil {
		return err
	}
	return p.Write(stream)
}
