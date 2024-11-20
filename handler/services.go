package handler

import (
	"fmt"

	"github.com/rohitxdev/go-api-starter/blobstore"
	"github.com/rohitxdev/go-api-starter/config"
	"github.com/rohitxdev/go-api-starter/email"
	"github.com/rohitxdev/go-api-starter/kvstore"
	"github.com/rohitxdev/go-api-starter/repo"
	"github.com/rs/zerolog"
)

type Services struct {
	BlobStore *blobstore.Store
	Config    *config.Config
	Email     *email.Client
	KVStore   *kvstore.Store
	Logger    *zerolog.Logger
	Repo      *repo.Repo
}

func (s *Services) Close() error {
	if err := s.KVStore.Close(); err != nil {
		return fmt.Errorf("Failed to close KV store: %w", err)
	}
	if err := s.Repo.Close(); err != nil {
		return fmt.Errorf("Failed to close repo: %w", err)
	}
	return nil
}
