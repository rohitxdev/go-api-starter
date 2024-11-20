package blobstore_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"testing"

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

	file, err := os.CreateTemp("", "test.txt")
	assert.Nil(t, err)

	defer func() {
		file.Close()
		os.Remove(file.Name())
	}()

	err = os.WriteFile(file.Name(), []byte("lorem ipsum dorem"), 0666)
	assert.Nil(t, err)

	testFileContent, err := io.ReadAll(file)
	assert.Nil(t, err)

	t.Run("Put file into bucket", func(t *testing.T) {
		args := blobstore.PutParams{
			BucketName:  cfg.S3BucketName,
			FileName:    file.Name(),
			ContentType: http.DetectContentType(testFileContent),
		}
		url, err := store.Put(ctx, &args)
		assert.Nil(t, err)

		res, err := http.DefaultClient.Post(url, "application/octet-stream", bytes.NewReader(testFileContent))
		assert.Nil(t, err)

		defer res.Body.Close()
	})

	t.Run("Get file from bucket", func(t *testing.T) {
		args := blobstore.GetParams{
			BucketName: cfg.S3BucketName,
			FileName:   file.Name(),
		}
		url, err := store.Get(ctx, &args)
		assert.Nil(t, err)

		res, err := http.DefaultClient.Get(url)
		assert.Nil(t, err)

		defer res.Body.Close()

		fileContent, err := io.ReadAll(res.Body)
		assert.Nil(t, err)
		assert.True(t, bytes.Equal(testFileContent, fileContent))

	})

	t.Run("Delete file from bucket", func(t *testing.T) {
		deleteArgs := blobstore.DeleteParams{
			BucketName: cfg.S3BucketName,
			FileName:   file.Name(),
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
			FileName:   file.Name(),
		}
		getURL, err := store.Get(ctx, &getArgs)
		assert.Nil(t, err)

		res, err = http.DefaultClient.Get(getURL)
		assert.Nil(t, err)

		defer res.Body.Close()
		assert.Equal(t, http.StatusNotFound, res.StatusCode)

	})

}
