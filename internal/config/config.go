package config

import (
	"fmt"
	"os"
)

type Config struct {
	AWSRegion         string
	EC2InstanceID     string
	BasicAuthUser     string
	BasicAuthPassword string
	Port              string
}

func Load() (Config, error) {
	cfg := Config{
		AWSRegion:         os.Getenv("EC2_REGION"),
		EC2InstanceID:     os.Getenv("EC2_INSTANCE_ID"),
		BasicAuthUser:     os.Getenv("BASIC_AUTH_USERNAME"),
		BasicAuthPassword: os.Getenv("BASIC_AUTH_PASSWORD"),
		Port:              os.Getenv("PORT"),
	}
	if cfg.AWSRegion == "" {
		cfg.AWSRegion = os.Getenv("AWS_REGION")
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	required := map[string]string{
		"EC2_REGION":          cfg.AWSRegion,
		"EC2_INSTANCE_ID":     cfg.EC2InstanceID,
		"BASIC_AUTH_USERNAME": cfg.BasicAuthUser,
		"BASIC_AUTH_PASSWORD": cfg.BasicAuthPassword,
	}
	for name, value := range required {
		if value == "" {
			return Config{}, fmt.Errorf("%s is required", name)
		}
	}
	return cfg, nil
}
