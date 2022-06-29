package osop

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"sync"
)

var ErrSeekArg = fmt.Errorf("seek illegal arg")

const defaultPartSize = 16 * 1024 * 1024

var partPool = sync.Pool{New: func() interface{} {
	return &Part{
		buf:    make([]byte, defaultPartSize),
		offset: 0,
		end:    0,
	}
}}

type Part struct {
	buf    []byte
	offset int64
	end    int64
}

func (part Part) Cap() int64 {
	return int64(len(part.buf))
}

func (part *Part) Reset() {
	part.offset = 0
	part.end = 0
}

func (part *Part) Read(p []byte) (n int, err error) {
	if part.offset >= part.end {
		logrus.Tracef("part.offset:%d, part.end:%d", part.offset, part.end)
		return 0, io.EOF
	}
	n = copy(p, part.buf[part.offset:part.end])
	part.offset += int64(n)
	return n, nil
}

func (part *Part) Write(p []byte) (n int, err error) {
	if part.end >= int64(len(part.buf)) {
		logrus.Tracef("part.end:%d, len(part.buf):%d", part.end, len(part.buf))
		return 0, io.ErrShortWrite
	}
	n = copy(part.buf[part.end:], p)
	part.end += int64(n)
	if n < len(p) {
		logrus.Tracef("part.Write end:%d,copy n:%d,len(p):%d", part.end, n, len(p))
		return n, io.ErrShortWrite
	}
	return n, nil
}

func (part *Part) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		if offset < 0 {
			return 0, ErrSeekArg
		}
		part.offset = offset
	case io.SeekCurrent:
		if part.offset+offset < 0 {
			return 0, ErrSeekArg
		}
		part.offset += offset
	case io.SeekEnd:
		if part.end+offset < 0 {
			return 0, ErrSeekArg
		}
		part.offset = part.end + offset
	}
	return part.offset, nil
}
