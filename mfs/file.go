package mfs

import (
	"crypto/md5"
	"github.com/sirupsen/logrus"
	"io/fs"
	"time"
)

type MockFile struct {
	name     string
	size     int64
	mode     fs.FileMode
	modeTime time.Time
	isDir    bool
	data     *fileData
}

func (v MockFile) Name() string {
	return v.name
}

func (v MockFile) Size() int64 {
	return v.size
}

func (v MockFile) Mode() fs.FileMode {
	return v.mode
}

func (v MockFile) ModTime() time.Time {
	return v.modeTime
}

func (v MockFile) IsDir() bool {
	return v.isDir
}

func (v MockFile) Sys() interface{} {
	//TODO implement me
	return nil
}

func (v *MockFile) Stat() (fs.FileInfo, error) {
	return v, nil
}

func (v *MockFile) Read(p []byte) (int, error) {
	return v.data.Read(p)
}

func (v *MockFile) Close() error {
	return v.data.Close()
}

func NewDir(filePath string, mode fs.FileMode) fs.File {
	return &MockFile{
		name:     filePath,
		isDir:    true,
		modeTime: time.Now(),
	}
}

func NewMockRandomFile(filePath string, mode fs.FileMode, size int64) fs.File {
	md5hash := md5.Sum([]byte(filePath))
	var seed int64 = 0
	for _, v := range md5hash {
		seed += 1 << int(v)
	}
	logrus.Infof("NewMockRandomFile filePath:%s,seed:%d", filePath, seed)
	return &MockFile{
		name:     filePath,
		size:     size,
		mode:     mode,
		modeTime: time.Now(),
		isDir:    false,
		data:     newFileData(RandomSource, size, seed),
	}
}

func NewZeroFile(filePath string, mode fs.FileMode, size int64) fs.File {
	return &MockFile{
		name:     filePath,
		size:     size,
		mode:     mode,
		modeTime: time.Now(),
		isDir:    false,
		data:     newFileData(ZeroSource, size, 0),
	}
}

func NewVFile(filePath string, dataType SourceType, mode fs.FileMode, size, seed int64, buf ...byte) fs.File {
	return &MockFile{
		name:     filePath,
		size:     size,
		mode:     mode,
		modeTime: time.Now(),
		isDir:    false,
		data:     newFileData(dataType, size, seed, buf...),
	}
}
