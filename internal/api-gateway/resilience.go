package api_gateway

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/sony/gobreaker"
)

type ResilientTransport struct {
	Next       http.RoundTripper
	CB         *gobreaker.CircuitBreaker
	MaxRetries int
}

func NewCircuitBreaker(name string) *gobreaker.CircuitBreaker {
	return gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        name,
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     5 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.Requests >= 5 && float64(counts.TotalFailures)/float64(counts.Requests) >= 0.5
		},
	})
}

func (t *ResilientTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	respInterface, err := t.CB.Execute(func() (interface{}, error) {
		return t.doWithRetry(req)
	})

	if err != nil {
		return t.fallbackResponse(req, err), nil
	}

	return respInterface.(*http.Response), nil
}

func (t *ResilientTransport) doWithRetry(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	for attempt := 0; attempt <= t.MaxRetries; attempt++ {
		resp, err = t.Next.RoundTrip(req)

		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}

		if attempt == t.MaxRetries {
			break
		}

		time.Sleep(100 * time.Millisecond * time.Duration(attempt+1))
	}

	if err != nil {
		return nil, err
	}
	return nil, errors.New("service returned 50x errors repeatedly")
}

func (t *ResilientTransport) fallbackResponse(req *http.Request, err error) *http.Response {
	msg := "Service is temporarily unavailable. Please try again later."
	if errors.Is(err, gobreaker.ErrOpenState) {
		msg = "Service is under heavy load. Circuit breaker is OPEN. Please wait."
	}

	body := `{"error": "` + msg + `"}`

	return &http.Response{
		StatusCode:    http.StatusServiceUnavailable,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Body:          io.NopCloser(bytes.NewBufferString(body)),
		ContentLength: int64(len(body)),
		Request:       req,
		Header:        make(http.Header),
	}
}
