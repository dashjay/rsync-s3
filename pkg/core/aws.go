package core

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/sirupsen/logrus"

	"github.com/dashjay/rsync-s3/pkg/config"
)

type S3Client struct {
	cli    *s3.S3
	bucket string
	prefix string
}

func NewS3Client(cfg *config.Config) *S3Client {
	var creds *credentials.Credentials
	if cfg.S3Accesskey != "" && cfg.S3Secretkey != "" {
		creds = credentials.NewStaticCredentials(cfg.S3Accesskey, cfg.S3Secretkey, "")
	} else {
		creds = credentials.NewCredentials(&credentials.SharedCredentialsProvider{})
	}
	awsCfg := &aws.Config{
		Endpoint:         aws.String(cfg.S3Endpoint),
		Credentials:      creds,
		S3ForcePathStyle: aws.Bool(true),
		DisableSSL:       aws.Bool(true),
		Region:           aws.String("default"),
	}
	awsSession, err := session.NewSession(awsCfg)
	if err != nil {
		panic(err)
	}
	return &S3Client{cli: s3.New(awsSession, awsCfg), bucket: cfg.S3Bucket, prefix: cfg.S3Prefix}
}

func (s *S3Client) ListObjects() (FileList, error) {
	logrus.WithField("bucket", s.bucket).WithField("prefix", s.prefix).Infoln("list object from aws")
	var out = make(FileList, 0)
	cnt := 0
	params := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(s.prefix),
	}
	isTruncated := true
	for isTruncated {
		resp, err := s.cli.ListObjectsV2(params)
		if err != nil {
			return nil, err
		}
		for _, c := range resp.Contents {
			out = append(out, FileInfo{
				Path:  []byte(*c.Key),
				Size:  aws.Int64Value(c.Size),
				Mtime: int32(c.LastModified.Unix()),
				Mode:  FileMode(33188),
			})
			cnt++
		}
		fmt.Printf("\r%d files listed", cnt)
		params.ContinuationToken = resp.NextContinuationToken
		isTruncated = *resp.IsTruncated
	}
	return out, nil
}
