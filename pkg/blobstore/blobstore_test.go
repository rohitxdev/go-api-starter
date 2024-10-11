package blobstore_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/rohitxdev/go-api-starter/internal/config"
	"github.com/rohitxdev/go-api-starter/pkg/blobstore"
)

func TestStorageService(t *testing.T) {
	os.Setenv("SECRETS_FILE", "../../secrets.json")
	c, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	fs, err := blobstore.New(c.S3Endpoint, c.S3DefaultRegion, c.AwsAccessKeyId, c.AwsAccessKeySecret)
	if err != nil {
		t.Fatal(err)
	}

	testFile, err := os.CreateTemp("", "test.txt")
	if err != nil {
		t.Error(err)
	}
	defer os.Remove(testFile.Name())
	defer testFile.Close()
	if err = os.WriteFile(testFile.Name(), []byte("lorem ipsum dorem"), 0666); err != nil {
		t.Error(err)
	}
	testFileContent, err := io.ReadAll(testFile)
	if err != nil {
		t.Error(err)
	}

	t.Run("Upload file to bucket", func(t *testing.T) {
		req, err := fs.PresignPutObject(ctx, c.S3Endpoint, testFile.Name(), http.DetectContentType(testFileContent))
		if err != nil {
			t.Error(err)
		}
		res, err := http.DefaultClient.Post(req.URL, "application/octet-stream", bytes.NewReader(testFileContent))
		if err != nil {
			t.Error(err)
		}
		t.Log(res.StatusCode)
	})

	t.Run("Get file from bucket", func(t *testing.T) {
		req, err := fs.PresignGetObject(ctx, c.S3Endpoint, testFile.Name())
		if err != nil {
			t.Error(err)
		}
		res, err := http.DefaultClient.Get(req.URL)
		if err != nil {
			t.Error(err)
		}
		defer res.Body.Close()
		fileContent, err := io.ReadAll(res.Body)
		if err != nil {
			t.Error(err)
		}
		if !bytes.Equal(fileContent, testFileContent) {
			t.Error(blobstore.ErrFileEmpty)
		}
	})

	t.Run("Delete file from bucket", func(t *testing.T) {
		req, err := fs.PresignDeleteObject(ctx, c.S3Endpoint, testFile.Name())
		if err != nil {
			t.Error(err)
		}
		url, _ := url.Parse(req.URL)
		res, err := http.DefaultClient.Do(&http.Request{Method: http.MethodDelete, URL: url})
		if err != nil {
			t.Error(err)
		}
		t.Log(res.StatusCode)
		// _, err = fs.Get(ctx, c.S3Endpoint, testFile.Name())
		// if err == nil {
		// 	t.Error(err)
		// }
	})
}
