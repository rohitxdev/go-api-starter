package assets

import "embed"

//go:embed public/* templates/*
var FS embed.FS
