package updater

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Fetcher downloads IP blocklist data with retry and disk cache fallback.
type Fetcher struct {
	client   *http.Client
	cacheDir string
	maxRetry int
	backoff  time.Duration
}

// NewFetcher creates a new Fetcher.
func NewFetcher(cacheDir string) *Fetcher {
	tr := &http.Transport{
		DisableCompression: false, // allow Go to auto-decompress gzip when Content-Encoding is set
	}
	return &Fetcher{
		client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: tr,
		},
		cacheDir: cacheDir,
		maxRetry: 3,
		backoff:  5 * time.Second,
	}
}

// Fetch downloads data from a source, with retry and fallback to cache.
func (f *Fetcher) Fetch(src Source) ([]byte, error) {
	var lastErr error
	for i := 0; i < f.maxRetry; i++ {
		if i > 0 {
			time.Sleep(f.backoff * time.Duration(i))
		}
		data, err := f.fetchOnce(src.URL, src.APIKey)
		if err == nil {
			f.saveToCache(src.Name, data)
			return data, nil
		}
		lastErr = err
	}
	if cached, err := f.loadFromCache(src.Name); err == nil {
		return cached, nil
	}
	return nil, fmt.Errorf("fetch failed after %d retries: %w", f.maxRetry, lastErr)
}

func (f *Fetcher) fetchOnce(url, apiKey string) ([]byte, error) {
	// Handle file:// URLs — read directly from disk
	if len(url) >= 7 && url[:7] == "file://" {
		return f.fetchFile(url[7:])
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "go-peerblock/0.1")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("empty response")
	}

	// Auto-decompress gzip if server returned raw gzip (without Content-Encoding header)
	if len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b {
		return f.decompressGzip(data)
	}

	return data, nil
}

// fetchFile reads data from a local file path (file:// URL handler).
func (f *Fetcher) fetchFile(path string) ([]byte, error) {
	// Trim leading slash on Windows: file:///C:/... → C:/...
	// On Windows, path after file:/// starts with /C:/...
	if len(path) > 2 && path[0] == '/' && path[2] == ':' {
		path = path[1:] // /C:/... → C:/...
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("file read error: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("empty file: %s", path)
	}
	return data, nil
}

func (f *Fetcher) decompressGzip(data []byte) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("gzip decompress: %w", err)
	}
	defer gr.Close()
	out, err := io.ReadAll(gr)
	if err != nil {
		return nil, fmt.Errorf("gzip read: %w", err)
	}
	return out, nil
}

func (f *Fetcher) saveToCache(name string, data []byte) {
	if f.cacheDir == "" {
		return
	}
	_ = os.MkdirAll(f.cacheDir, 0755)
	path := filepath.Join(f.cacheDir, name+".cache")
	_ = os.WriteFile(path, data, 0644)
}

func (f *Fetcher) loadFromCache(name string) ([]byte, error) {
	if f.cacheDir == "" {
		return nil, fmt.Errorf("cache dir not set")
	}
	path := filepath.Join(f.cacheDir, name+".cache")
	return os.ReadFile(path)
}
