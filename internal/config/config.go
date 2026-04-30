package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Default parameters
const (
	DefaultPort            = "8848"
	DefaultSearchEngine    = "duckduckgo"
	DefaultFetchMethod     = "direct"
	DefaultChromeDebugAddr = "localhost:9222"
	DefaultCdpMode         = "connect"

	DefaultCacheMaxEntries   = 128
	DefaultCacheTTL          = 5 * time.Minute
	DefaultCacheMaxEntrySize = 512 * 1024
)

type Config struct {
	Port string
}

func Load(cliPort string) *Config {
	port := cliPort
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = DefaultPort
	}
	return &Config{Port: port}
}

func SearchEngine() string {
	engine := strings.ToLower(os.Getenv("SEARCH_ENGINE"))
	if engine == "" {
		return DefaultSearchEngine
	}
	return engine
}

func TavilyAPIKey() string {
	return os.Getenv("TAVILY_API_KEY")
}

func CdpMode() string {
	mode := strings.ToLower(os.Getenv("CDP_MODE"))
	if mode == "" {
		return DefaultCdpMode
	}
	return mode
}

func FetchMethod() string {
	method := strings.ToLower(os.Getenv("FETCH_METHOD"))
	if method == "" {
		return DefaultFetchMethod
	}
	return method
}

func ChromeDebugAddr() string {
	addr := os.Getenv("CHROME_DEBUG_ADDR")
	if addr == "" {
		return DefaultChromeDebugAddr
	}
	return addr
}

func CacheMaxEntries() int {
	s := os.Getenv("CACHE_MAX_ENTRIES")
	if s == "" {
		return DefaultCacheMaxEntries
	}
	var n int
	if _, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &n); err != nil || n < 0 {
		return DefaultCacheMaxEntries
	}
	return n
}

func CacheTTL() time.Duration {
	s := os.Getenv("CACHE_TTL")
	if s == "" {
		return DefaultCacheTTL
	}
	d, err := time.ParseDuration(strings.TrimSpace(s))
	if err != nil || d <= 0 {
		return DefaultCacheTTL
	}
	return d
}

func CacheMaxEntrySize() int {
	s := os.Getenv("CACHE_MAX_ENTRY_SIZE")
	if s == "" {
		return DefaultCacheMaxEntrySize
	}
	var n int
	if _, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &n); err != nil || n < 0 {
		return DefaultCacheMaxEntrySize
	}
	return n
}
