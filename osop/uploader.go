package osop

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/sirupsen/logrus"
	"io"
)

func UploadMultiPartFromReader(svc *s3.S3, reader io.Reader, bucket, key string) error {
	uploadId, err := startUploadPart(svc, bucket, key)
	if err != nil {
		logrus.Errorf("startUploadPart failed:%v", err)
		return err
	}
	if _, err := putParts(svc, reader, bucket, key, uploadId); err != nil {
		logrus.Errorf("putParts failed:%v", err)
		if err := abortPutPart(svc, bucket, key, uploadId); err != nil {
			logrus.Errorf("abortPutPart failed:%v", err)
		}
		return err
	}

	completeParts, err := getCompleteParts(svc, bucket, key, uploadId)
	if err != nil {
		logrus.Errorf("getCompleteParts failed:%v", err)
	}

	if err := finishMultiUploadPart(svc, bucket, key, uploadId, completeParts); err != nil {
		logrus.Tracef("finishMultiUploadPart failed:%v", err)
		return err
	}
	return nil
}

func startUploadPart(svc *s3.S3, bucket, key string) (uploadId string, err error) {
	input := new(s3.CreateMultipartUploadInput)
	input.SetBucket(bucket)
	input.SetKey(key)
	input.SetACL("public-read")
	output, err := svc.CreateMultipartUpload(input)
	if err != nil {
		return "", err
	}
	return aws.StringValue(output.UploadId), nil
}

func putParts(svc *s3.S3, reader io.Reader, bucket, key, uploadId string) (int64, error) {
	var partNum int64 = 0
	part := partPool.New().(*Part)
	defer partPool.Put(part)
	defer part.Reset()
	fileLength := int64(0)
	for {
		n, err := io.CopyN(part, reader, part.Cap())
		if err != nil && err != io.EOF {
			logrus.Tracef("io.CopyN failed:%v", err)
			return 0, err
		}
		if err := putPart(svc, part, bucket, key, uploadId, partNum); err != nil {
			logrus.Tracef("putPart failed:%v", err)
			return 0, err
		}
		logrus.Tracef("putPart success:%v, %d", partNum, n)
		part.Reset()
		partNum++
		fileLength += n
		if n < part.Cap() {
			break
		}
	}
	return fileLength, nil
}

func putPart(svc *s3.S3, part io.ReadSeeker, bucket, key, uploadId string, partNum int64) error {
	input := &s3.UploadPartInput{}
	input.SetBody(part)
	input.SetBucket(bucket)
	input.SetKey(key)
	input.SetPartNumber(partNum)
	input.SetUploadId(uploadId)
	output, err := svc.UploadPart(input)
	if err != nil {
		return err
	}
	logrus.Trace(output.String())
	return nil
}

func getCompleteParts(svc *s3.S3, bucket, key, uploadId string) (*s3.CompletedMultipartUpload, error) {
	input := new(s3.ListPartsInput)
	input.SetBucket(bucket)
	input.SetKey(key)
	input.SetUploadId(uploadId)
	output, err := svc.ListParts(input)
	if err != nil {
		return nil, err
	}

	completeParts := make([]*s3.CompletedPart, 0, len(output.Parts))
	for _, part := range output.Parts {
		completeParts = append(completeParts, &s3.CompletedPart{
			ETag:       part.ETag,
			PartNumber: part.PartNumber,
		})
	}

	completePartsUpload := &s3.CompletedMultipartUpload{Parts: completeParts}
	logrus.Trace(completePartsUpload.String())
	return completePartsUpload, nil
}

func finishMultiUploadPart(svc *s3.S3, bucket, key, uploadId string, completeParts *s3.CompletedMultipartUpload) error {
	input := new(s3.CompleteMultipartUploadInput)
	input.SetBucket(bucket)
	input.SetKey(key)
	input.SetUploadId(uploadId)
	input.SetMultipartUpload(completeParts)

	_, err := svc.CompleteMultipartUpload(input)
	if err != nil {
		return err
	}
	return nil
}

func abortPutPart(svc *s3.S3, bucket, key, uploadId string) error {
	input := new(s3.AbortMultipartUploadInput)
	input.SetBucket(bucket)
	input.SetKey(key)
	input.SetUploadId(uploadId)
	_, err := svc.AbortMultipartUpload(input)
	if err != nil {
		return err
	}
	return nil
}
