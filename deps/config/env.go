package config

type Environment = string

const (
	EnvTest        Environment = "testing"
	EnvDevelopment Environment = "development"
	EnvStaging     Environment = "staging"
	EnvProduction  Environment = "production"
)
