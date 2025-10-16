package main

import (
	"net/http"
	"net/url"
	"os"
)

type ProxyConfig struct {
	Transport http.RoundTripper
}

func LoadProxyConfig() *ProxyConfig {
	// Check for HTTP_PROXY and HTTPS_PROXY environment variables
	httpProxy := os.Getenv("HTTP_PROXY")
	httpsProxy := os.Getenv("HTTPS_PROXY")

	if httpProxy == "" && httpsProxy == "" {
		return nil
	}

	// Use default transport with proxy support
	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			if req.URL.Scheme == "https" && httpsProxy != "" {
				return url.Parse(httpsProxy)
			}
			if req.URL.Scheme == "http" && httpProxy != "" {
				return url.Parse(httpProxy)
			}
			return nil, nil
		},
	}

	return &ProxyConfig{
		Transport: transport,
	}
}
