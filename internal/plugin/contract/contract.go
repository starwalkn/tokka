package contract

import "time"

type RateLimit interface {
	Allow(key string) bool
}

type Metrics interface {
	ObserveRequest(method, path string, duration time.Duration, status int)
	// ObserveBackend(routePath, backendURL, method string, duration time.Duration, status int)
}
