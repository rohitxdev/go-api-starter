package config

type appEnv = string

const (
	EnvTest        appEnv = "test"
	EnvDevelopment appEnv = "development"
	EnvStaging     appEnv = "staging"
	EnvProduction  appEnv = "production"
)
