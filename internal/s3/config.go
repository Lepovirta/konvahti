package s3

import (
	"fmt"
	"strings"
)

type Config struct {
	Endpoint        string `yaml:"endpoint"`
	AccessKeyId     string `yaml:"accessKeyId"`
	SecretAccessKey string `yaml:"secretAccessKey"`
	BucketName      string `yaml:"bucketName"`
	BucketPrefix    string `yaml:"bucketPrefix"`
	Directory       string `yaml:"directory"`
	DisableTLS      bool   `yaml:"disableTls,omitempty"`
}

func (c *Config) Validate() error {
	if c.Endpoint == "" {
		return fmt.Errorf("no S3 endpoint specified")
	}
	if c.AccessKeyId == "" {
		return fmt.Errorf("no S3 access key ID specified")
	}
	if c.SecretAccessKey == "" {
		return fmt.Errorf("no S3 secret access key specified")
	}
	if c.BucketName == "" {
		return fmt.Errorf("no S3 bucket name specified")
	}
	if c.Directory == "" {
		return fmt.Errorf("no local directory specified")
	}
	return nil
}

func (c *Config) sanitizeBucketPrefix() {
	c.BucketPrefix = sanitizeBucketPrefix(c.BucketPrefix)
}

func sanitizeBucketPrefix(bucketPrefix string) string {
	trimmedPrefix := strings.Trim(bucketPrefix, "/")
	if trimmedPrefix == "" {
		return ""
	}
	return fmt.Sprintf("%s/", trimmedPrefix)
}
