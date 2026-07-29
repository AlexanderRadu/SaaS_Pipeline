package api_gateway

import (
	"SaaS_Pipeline/pkg/resilience"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

func createProxy(serviceName string, targetURLStr string) (*httputil.ReverseProxy, error) {
	targetURL, err := url.Parse(targetURLStr)
	if err != nil {
		return nil, fmt.Errorf("invalid target url for %s (%s): %w", serviceName, targetURLStr, err)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	transport := http.DefaultTransport.(*http.Transport).Clone()

	transport.ResponseHeaderTimeout = 3 * time.Second

	proxy.Transport = &resilience.ResilientTransport{
		Next:       transport,
		CB:         resilience.NewCircuitBreaker(serviceName),
		MaxRetries: 2,
	}

	return proxy, nil
}
