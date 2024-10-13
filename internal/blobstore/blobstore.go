// Package blobstore provides a wrapper around S3 client.
package blobstore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var (
	ErrFileEmpty = errors.New("file is empty")
)

type Store struct {
	client    *s3.Client
	presigner *s3.PresignClient
}

func New(endpoint string, region string, accessKeyId string, accessKeySecret string) (*Store, error) {
	customResolver := aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: endpoint,
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithEndpointResolver(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyId, accessKeySecret, "")),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("could not load default config of S3 client: %w", err)
	}

	s3Client := s3.NewFromConfig(cfg)

	client := Store{
		client:    s3Client,
		presigner: s3.NewPresignClient(s3Client),
	}

	return &client, nil
}

/*----------------------------------- Upload File To Bucket ----------------------------------- */

func (s *Store) PresignPutObject(ctx context.Context, bucketName string, fileName string, contentType string) (*v4.PresignedHTTPRequest, error) {
	request, err := s.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      &bucketName,
		Key:         &fileName,
		ContentType: &contentType})
	return request, err
}

/*----------------------------------- Get File From Bucket ----------------------------------- */

func (s *Store) PresignGetObject(ctx context.Context, bucketName string, fileName string) (*v4.PresignedHTTPRequest, error) {
	return s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{Bucket: &bucketName, Key: &fileName}, func(po *s3.PresignOptions) { po.Expires = time.Minute * 2 })
}

/*----------------------------------- Delete File From Bucket ----------------------------------- */

func (s *Store) PresignDeleteObject(ctx context.Context, bucketName string, fileName string) (*v4.PresignedHTTPRequest, error) {
	return s.presigner.PresignDeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &bucketName, Key: &fileName})
}

/*----------------------------------- Get List Of Files ----------------------------------- */

type FileMetaData struct {
	LastModified time.Time `json:"last_modified"`
	FileName     string    `json:"file_name"`
	SizeInBytes  uint64    `json:"size_in_bytes"`
}

func (s *Store) GetList(ctx context.Context, bucketName string, subDir string) ([]FileMetaData, error) {
	var continuationToken *string
	var fileList []FileMetaData
	for {
		objects, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{Bucket: &bucketName, ContinuationToken: continuationToken, Prefix: aws.String(subDir)})
		if err != nil {
			return nil, err
		}
		for _, file := range objects.Contents {
			fileList = append(fileList, FileMetaData{FileName: *file.Key, LastModified: *aws.Time(*file.LastModified), SizeInBytes: uint64(*file.Size)})
		}
		if !*objects.IsTruncated {
			break
		}
		continuationToken = objects.NextContinuationToken
	}
	return fileList, nil
}
