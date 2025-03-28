package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"golang.org/x/sync/errgroup"
)

// Advanced Configuration Structures
type ProxyConfig struct {
	URL      string
	Protocol string
	Auth     *ProxyAuth
}

type ProxyAuth struct {
	Username string
	Password string
}

type SourceConfig struct {
	Name            string
	URL             string
	Method          string
	Headers         map[string]string
	RateLimit       time.Duration
	Timeout         time.Duration
	RetryStrategy   RetryStrategy
	ParsingStrategy func([]byte) ([]string, error)
}

type RetryStrategy struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

// Enhanced Proxy Manager
type ProxyManager struct {
	proxies      []ProxyConfig
	currentIndex int
	mu           sync.Mutex
}

func NewProxyManager(proxies []ProxyConfig) *ProxyManager {
	return &ProxyManager{
		proxies: proxies,
	}
}

func (pm *ProxyManager) GetNextProxy() (*url.URL, *ProxyAuth, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if len(pm.proxies) == 0 {
		return nil, nil, errors.New("no proxies available")
	}

	proxy := pm.proxies[pm.currentIndex]
	proxyURL, err := url.Parse(proxy.URL)
	if err != nil {
		return nil, nil, err
	}

	pm.currentIndex = (pm.currentIndex + 1) % len(pm.proxies)
	return proxyURL, proxy.Auth, nil
}

// Intelligent HTTP Client with Advanced Features
type ResilientHTTPClient struct {
	client       *http.Client
	proxyManager *ProxyManager
	logger       *log.Logger
}

func NewResilientHTTPClient(proxyManager *ProxyManager) *ResilientHTTPClient {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
	}

	return &ResilientHTTPClient{
		client:       &http.Client{Transport: transport},
		proxyManager: proxyManager,
		logger:       log.New(os.Stdout, "HTTPClient: ", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

func (c *ResilientHTTPClient) Do(req *http.Request, strategy RetryStrategy) (*http.Response, error) {
	for attempt := 0; attempt < strategy.MaxAttempts; attempt++ {
		// Exponential Backoff with Jitter
		delay := calculateBackoff(strategy, attempt)
		time.Sleep(delay)

		// Proxy Rotation
		if c.proxyManager != nil {
			proxyURL, proxyAuth, err := c.proxyManager.GetNextProxy()
			if err == nil {
				c.client.Transport.(*http.Transport).Proxy = http.ProxyURL(proxyURL)
				
				// Handle Proxy Authentication if needed
				if proxyAuth != nil {
					req.SetBasicAuth(proxyAuth.Username, proxyAuth.Password)
				}
			}
		}

		// Execute Request
		resp, err := c.client.Do(req)
		if err == nil {
			return resp, nil
		}

		// Log and track errors
		c.logger.Printf("Request attempt %d failed: %v", attempt+1, err)

		// Circuit breaker-like mechanism
		if isUnrecoverableError(err) {
			return nil, err
		}
	}

	return nil, fmt.Errorf("request failed after %d attempts", strategy.MaxAttempts)
}

// Intelligent Backoff Calculation
func calculateBackoff(strategy RetryStrategy, attempt int) time.Duration {
	// Exponential backoff with full jitter
	exponentialDelay := strategy.BaseDelay * time.Duration(math.Pow(2, float64(attempt)))
	jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
	
	delay := exponentialDelay + jitter
	if delay > strategy.MaxDelay {
		delay = strategy.MaxDelay
	}

	return delay
}

// Error Classification
func isUnrecoverableError(err error) bool {
	// Implement sophisticated error classification
	unrecoverableErrors := []error{
		// Add specific error types that should not be retried
	}

	for _, uErr := range unrecoverableErrors {
		if errors.Is(err, uErr) {
			return true
		}
	}
	return false
}

// Comprehensive Source Management
var SourceConfigurations = []SourceConfig{
	{
		Name:   "CrtSh",
		URL:    "https://crt.sh/?q=%%25.%s&output=json",
		Method: "GET",
		Headers: map[string]string{
			"User-Agent": "Mozilla/5.0 (Advanced Recon Tool)",
		},
		RateLimit: 2 * time.Second,
		Timeout:   10 * time.Second,
		RetryStrategy: RetryStrategy{
			MaxAttempts: 3,
			BaseDelay:   1 * time.Second,
			MaxDelay:    30 * time.Second,
		},
		ParsingStrategy: parseCrtShResponse,
	},
	// Add more sources with similar configuration
}

// Subdomain Enumerator with Advanced Features
type SubdomainEnumerator struct {
	sources       []SourceConfig
	proxyManager  *ProxyManager
	results       map[string]bool
	mu            sync.Mutex
	errorHandling chan error
	logger        *log.Logger
}

func NewSubdomainEnumerator(proxies []ProxyConfig) *SubdomainEnumerator {
	return &SubdomainEnumerator{
		sources:       SourceConfigurations,
		proxyManager:  NewProxyManager(proxies),
		results:       make(map[string]bool),
		errorHandling: make(chan error, 10),
		logger:        log.New(os.Stdout, "Enumerator: ", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

func (e *SubdomainEnumerator) Enumerate(domain string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	group, ctx := errgroup.WithContext(ctx)
	httpClient := NewResilientHTTPClient(e.proxyManager)

	for _, sourceConfig := range e.sources {
		sourceConfig := sourceConfig // Capture range variable
		
		group.Go(func() error {
			subdomains, err := e.fetchSubdomainsFromSource(ctx, httpClient, domain, sourceConfig)
			if err != nil {
				e.errorHandling <- err
				return nil
			}

			e.mu.Lock()
			for _, subdomain := range subdomains {
				e.results[subdomain] = true
			}
			e.mu.Unlock()

			return nil
		})
	}

	// Asynchronous Error Handling
	go func() {
		for err := range e.errorHandling {
			e.logger.Printf("Enumeration Error: %v", err)
		}
	}()

	if err := group.Wait(); err != nil {
		return nil, err
	}

	// Convert results to slice
	var uniqueSubdomains []string
	for subdomain := range e.results {
		uniqueSubdomains = append(uniqueSubdomains, subdomain)
	}

	return uniqueSubdomains, nil
}

func (e *SubdomainEnumerator) fetchSubdomainsFromSource(
	ctx context.Context, 
	client *ResilientHTTPClient, 
	domain string, 
	config SourceConfig,
) ([]string, error) {
	url := fmt.Sprintf(config.URL, domain)
	req, err := http.NewRequestWithContext(ctx, config.Method, url, nil)
	if err != nil {
		return nil, err
	}

	// Set custom headers
	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req, config.RetryStrategy)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Use source-specific parsing strategy
	return config.ParsingStrategy(body)
}

// Parsing Strategies
func parseCrtShResponse(body []byte) ([]string, error) {
	var results []map[string]string
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, err
	}

	subdomains := make(map[string]bool)
	for _, result := range results {
		if commonName, ok := result["common_name"]; ok {
			subdomains[commonName] = true
		}
	}

	var uniqueSubdomains []string
	for sd := range subdomains {
		uniqueSubdomains = append(uniqueSubdomains, sd)
	}

	return uniqueSubdomains, nil
}

func main() {
	// Example proxy configurations
	proxies := []ProxyConfig{
		{
			URL:      "http://proxy1.example.com:8080",
			Protocol: "http",
			Auth: &ProxyAuth{
				Username: "user",
				Password: "pass",
			},
		},
		{
			URL:      "http://proxy2.example.com:8080",
			Protocol: "http",
		},
	}

	domain := "example.com"
	enumerator := NewSubdomainEnumerator(proxies)
	
	subdomains, err := enumerator.Enumerate(domain)
	if err != nil {
		log.Fatalf("Enumeration failed: %v", err)
	}

	fmt.Println("Discovered Subdomains:")
	for _, subdomain := range subdomains {
		color.Green(subdomain)
	}
}
