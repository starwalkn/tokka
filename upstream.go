package tokka

//go:generate mockgen -source=upstream.go -destination=mock/upstream.go -package=mock Upstream

import (
	"context"
	"net/http"
)

type Upstream interface {
	Name() string
	Policy() UpstreamPolicy
	Call(ctx context.Context, original *http.Request, originalBody []byte) *UpstreamResponse
	callWithRetry(ctx context.Context, original *http.Request, originalBody []byte, retryPolicy UpstreamRetryPolicy) *UpstreamResponse
}

type UpstreamPolicy struct {
	AllowedStatuses []int
	AllowEmptyBody  bool
	MapStatusCodes  map[int]int
	RetryPolicy     UpstreamRetryPolicy
}

type UpstreamRetryPolicy struct {
	MaxRetries      int
	RetryOnStatuses []int
	BackoffMs       int64
}

type UpstreamResponse struct {
	Status  int
	Headers http.Header
	Body    []byte
	Err     error
}
