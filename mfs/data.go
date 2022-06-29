package mfs

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"math/rand"
)

type SourceType string

const (
	RandomSource = SourceType("/dev/random")
	ZeroSource   = SourceType("/dev/zero")
	BufSource    = SourceType("buf")
)

const (
	defaultPieceSize = 1 * 1024 * 1024
)

type fileData struct {
	dataType   SourceType
	buf        []byte
	off        int64
	size       int64
	randomSeed int64
	pieceSize  int64
	pieceMap   []byte
}

func newFileData(dataType SourceType, size, seed int64, buf ...byte) *fileData {
	if dataType == BufSource {
		size = int64(len(buf))
	}
	return &fileData{
		dataType:   dataType,
		buf:        buf,
		size:       size,
		randomSeed: seed,
		pieceSize:  defaultPieceSize,
	}
}

func (d *fileData) Close() error {
	d.off = 0
	return nil
}

func (d *fileData) Read(p []byte) (n int, err error) {
	switch d.dataType {
	case BufSource:
		return d.readFromBuf(p)
	case RandomSource:
		return d.randomData(p)
	case ZeroSource:
		return d.zeroData(p)
	default:
		return 0, fmt.Errorf("unkown data type:%s", d.dataType)
	}
}

func (d *fileData) readFromBuf(p []byte) (n int, err error) {
	if d.off >= d.size {
		return 0, io.EOF
	}
	readSize := d.size - d.off
	if readSize > int64(len(p)) {
		readSize = int64(len(p))
	}
	n = copy(p, d.buf[d.off:d.off+readSize])
	d.off += readSize
	return n, nil
}

func (d *fileData) randomData(p []byte) (n int, err error) {
	if d.off >= d.size {
		return 0, io.EOF
	}
	readSize := d.size - d.off
	if readSize > int64(len(p)) {
		readSize = int64(len(p))
	}
	n = 0
	for i := int64(0); i < readSize; {
		pieceNum := d.off / d.pieceSize
		rand.Seed(d.randomSeed + pieceNum)
		pieceBuf := make([]byte, d.pieceSize)
		rand.Read(pieceBuf)
		cn := copy(p[i:readSize], pieceBuf[d.off%d.pieceSize:d.pieceSize])
		i += int64(cn)
		d.off += int64(cn)
		n += cn
	}
	return n, nil
}

func (d *fileData) zeroData(p []byte) (n int, err error) {
	if d.off >= d.size {
		return 0, io.EOF
	}

	readSize := d.size - d.off
	if readSize > int64(len(p)) {
		readSize = int64(len(p))
	}
	d.off += readSize
	return int(readSize), nil
}

func GetFileMd5Hash(reader io.Reader) (string, error) {
	hash := md5.New()
	n, err := io.Copy(hash, reader)
	if err != nil {
		log.Println("err:", err)
		return "", err
	}
	md5Hash := fmt.Sprintf("%x", hash.Sum(nil))
	log.Printf("fileLength:%d, md5hash:%s", n, md5Hash)
	return md5Hash, nil
}
