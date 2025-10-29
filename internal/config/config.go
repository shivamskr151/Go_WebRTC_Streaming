package config

import (
	"os"
	"strconv"
)

type Config struct {
	HTTP HTTPConfig `json:"http"`
	RTMP RTMPConfig `json:"rtmp"`
}

type HTTPConfig struct {
	Port int `json:"port"`
}

type RTMPConfig struct {
	Port int    `json:"port"`
	URL  string `json:"url"`
}

func Load() (*Config, error) {
	cfg := &Config{
		HTTP: HTTPConfig{
			Port: getEnvAsInt("HTTP_PORT", 8080),
		},
		RTMP: RTMPConfig{
			Port: getEnvAsInt("RTMP_PORT", 1936),
			URL:  getEnv("RTMP_URL", "rtmp://safetycaptain.arresto.in/camera_0051/0051?username=wrakash&password=akash@1997"),
		},
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
