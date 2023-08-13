package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	S3Accesskey string `json:"s3_accesskey,omitempty"`
	S3Secretkey string `json:"s3_secretkey,omitempty"`
	S3Endpoint  string `json:"s3_endpoint,omitempty"`
	S3Bucket    string `json:"s3_bucket,omitempty"`
	S3Prefix    string `json:"s3_prefix,omitempty"`

	RsyncEndpoint string `json:"rsync_endpoint,omitempty"`

	LogLevel  string `json:"log_level,omitempty"`
	PProfPort string `json:"pprof_port"`
}

func FromFile(path string) *Config {
	content, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	var cfg Config
	err = json.Unmarshal(content, &cfg)
	if err != nil {
		panic(err)
	}
	return &cfg
}
