package osop

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/go-playground/assert/v2"
	"github.com/sirupsen/logrus"
	"github.com/yarenhere/daily_learn/mfs"
	"io/fs"
	"testing"
)

func TestUploadMultiPartFromReader(t *testing.T) {
	reader := mfs.NewMockRandomFile("/abc/d.txt", fs.ModePerm, 10*1024*1024)
	mfs.GetFileMd5Hash(reader)
	reader.Close()

	s3Cfg := aws.NewConfig()
	s3Cfg.WithDisableSSL(true)
	s3Cfg.WithEndpoint("http://127.0.0.1:9000")
	s3Cfg.WithCredentials(credentials.NewStaticCredentials("32jDAV1Cy8vVhGRI", "WWtiiYyx2QLDYVByn7N9GXUhV3VMqAJT", ""))
	s3Cfg.WithRegion("us-east-1")
	s3Cfg.WithS3ForcePathStyle(true)

	session, err := session.NewSession(s3Cfg)
	assert.Equal(t, nil, err)
	svc := s3.New(session)

	logrus.SetLevel(logrus.TraceLevel)
	err = UploadMultiPartFromReader(svc, reader, "test-bucket", "abc/d.txt")
	assert.Equal(t, nil, err)
}
