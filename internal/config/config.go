package config

import (
	"os"
	"strings"
)

type Config struct {
	Port string
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8848"
	}
	return &Config{Port: port}
}

func SearchEngine() string {
	engine := strings.ToLower(os.Getenv("SEARCH_ENGINE"))
	if engine == "" {
		return "duckduckgo"
	}
	return engine
}

func TavilyAPIKey() string {
	return os.Getenv("TAVILY_API_KEY")
}
