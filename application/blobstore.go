// Package blobstore provides a wrapper around S3 client.
package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var (
	ErrFileEmpty = errors.New("file is empty")
)

type BlobStore struct {
	client        *s3.Client
	presignClient *s3.PresignClient
}

func NewBlobStore(endpoint string, region string, accessKeyId string, accessKeySecret string) (*BlobStore, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithBaseEndpoint(endpoint),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyId, accessKeySecret, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to default config of s3 client: %w", err)
	}

	s3Client := s3.NewFromConfig(cfg)
	client := BlobStore{
		client:        s3Client,
		presignClient: s3.NewPresignClient(s3Client),
	}

	return &client, nil
}

type BlobPutParams struct {
	BucketName  string
	FileName    string
	ContentType string
	ExpiresIn   time.Duration
}

// Put returns presigned URL to upload a file to S3 bucket.
func (s *BlobStore) Put(ctx context.Context, p *BlobPutParams) (string, error) {
	args := &s3.PutObjectInput{
		Bucket:      &p.BucketName,
		Key:         &p.FileName,
		ContentType: &p.ContentType,
	}
	req, err := s.presignClient.PresignPutObject(ctx, args, s3.WithPresignExpires(p.ExpiresIn))
	if err != nil {
		return "", err
	}
	return req.URL, err
}

type BlobGetParams struct {
	BucketName string
	FileName   string
	ExpiresIn  time.Duration
}

// Get returns presigned URL to download a file from S3 bucket.
func (s *BlobStore) Get(ctx context.Context, p *BlobGetParams) (string, error) {
	args := &s3.GetObjectInput{
		Bucket: &p.BucketName,
		Key:    &p.FileName,
	}
	req, err := s.presignClient.PresignGetObject(ctx, args, s3.WithPresignExpires(p.ExpiresIn))
	if err != nil {
		return "", err
	}
	return req.URL, err
}

type BlobDeleteParams struct {
	BucketName string
	FileName   string
	ExpiresIn  time.Duration
}

// Delete returns presigned URL to delete a file from S3 bucket.
func (s *BlobStore) Delete(ctx context.Context, p *BlobDeleteParams) (string, error) {
	args := &s3.DeleteObjectInput{
		Bucket: &p.BucketName,
		Key:    &p.FileName,
	}
	req, err := s.presignClient.PresignDeleteObject(ctx, args, s3.WithPresignExpires(p.ExpiresIn))
	if err != nil {
		return "", err
	}
	return req.URL, err
}

type FileMetaData struct {
	LastModified time.Time `json:"last_modified"`
	FileName     string    `json:"file_name"`
	SizeInBytes  int64     `json:"size_in_bytes"`
}

type BlobListParams struct {
	BucketName string
	SubDir     string
}

// List returns a list of files in a given directory in S3 bucket.
func (s *BlobStore) List(ctx context.Context, p *BlobListParams) ([]FileMetaData, error) {
	var token *string
	var files []FileMetaData
	for {
		objects, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{Bucket: &p.BucketName, ContinuationToken: token, Prefix: aws.String(p.SubDir)})
		if err != nil {
			return nil, err
		}
		for _, file := range objects.Contents {
			files = append(files, FileMetaData{FileName: *file.Key, LastModified: *aws.Time(*file.LastModified), SizeInBytes: *file.Size})
		}
		if !*objects.IsTruncated {
			break
		}
		token = objects.NextContinuationToken
	}
	return files, nil
}
