package blobstore_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/rohitxdev/go-api-starter/blobstore"
	"github.com/rohitxdev/go-api-starter/config"
	"github.com/stretchr/testify/assert"
)

func TestBlobStore(t *testing.T) {
	cfg, err := config.Load()
	assert.Nil(t, err)

	ctx := context.Background()
	store, err := blobstore.New(cfg.S3Endpoint, cfg.S3DefaultRegion, cfg.AWSAccessKeyID, cfg.AWSAccessKeySecret)
	assert.Nil(t, err)

	fileName := "test.txt"
	fileContent := []byte("lorem ipsum dorem")
	fileContentType := http.DetectContentType(fileContent)
	URLExpiresIn := time.Second * 10

	t.Run("Put file into bucket", func(t *testing.T) {
		args := blobstore.PutParams{
			BucketName:  cfg.S3BucketName,
			FileName:    fileName,
			ContentType: fileContentType,
			ExpiresIn:   URLExpiresIn,
		}
		putURL, err := store.Put(ctx, &args)
		assert.Nil(t, err)
		parsedURL, err := url.Parse(putURL)
		assert.Nil(t, err)

		req := http.Request{
			URL:    parsedURL,
			Method: http.MethodPut,
			Body:   io.NopCloser(bytes.NewReader(fileContent)),
			Header: http.Header{
				"Content-Type": []string{fileContentType},
			},
		}
		res, err := http.DefaultClient.Do(&req)
		assert.Nil(t, err)
		defer res.Body.Close()
	})

	t.Run("Get file from bucket", func(t *testing.T) {
		args := blobstore.GetParams{
			BucketName: cfg.S3BucketName,
			FileName:   fileName,
			ExpiresIn:  URLExpiresIn,
		}
		getURL, err := store.Get(ctx, &args)
		assert.Nil(t, err)

		res, err := http.DefaultClient.Get(getURL)
		assert.Nil(t, err)
		defer res.Body.Close()

		resBody, err := io.ReadAll(res.Body)
		assert.Nil(t, err)
		assert.True(t, bytes.Equal(fileContent, resBody))

	})

	t.Run("Delete file from bucket", func(t *testing.T) {
		deleteArgs := blobstore.DeleteParams{
			BucketName: cfg.S3BucketName,
			FileName:   fileName,
			ExpiresIn:  URLExpiresIn,
		}
		deleteURL, err := store.Delete(ctx, &deleteArgs)
		assert.Nil(t, err)
		parsedURL, err := url.Parse(deleteURL)
		assert.Nil(t, err)

		res, err := http.DefaultClient.Do(&http.Request{Method: http.MethodDelete, URL: parsedURL})
		assert.Nil(t, err)
		defer res.Body.Close()

		getArgs := blobstore.GetParams{
			BucketName: cfg.S3BucketName,
			FileName:   fileName,
			ExpiresIn:  URLExpiresIn,
		}
		getURL, err := store.Get(ctx, &getArgs)
		assert.Nil(t, err)

		res, err = http.DefaultClient.Get(getURL)
		assert.Nil(t, err)
		defer res.Body.Close()
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
	})

}
