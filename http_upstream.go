package tokka

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"
)

type httpUpstream struct {
	name                string
	url                 string
	method              string
	timeout             int64
	headers             map[string]string
	forwardHeaders      []string
	forwardQueryStrings []string
	policy              UpstreamPolicy

	client *http.Client
}

func (u *httpUpstream) Name() string {
	return u.name
}

func (u *httpUpstream) Policy() UpstreamPolicy {
	return u.policy
}

func (u *httpUpstream) Call(ctx context.Context, original *http.Request, originalBody []byte) *UpstreamResponse {
	uresp := &UpstreamResponse{
		Headers: make(http.Header, 0),
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(u.timeout)*time.Millisecond)
	defer cancel()

	req, err := u.newRequest(ctx, original, originalBody)
	if err != nil {
		uresp.Err = err
		return uresp
	}

	hresp, err := u.client.Do(req)
	if err != nil {
		uresp.Err = err
		return uresp
	}
	defer hresp.Body.Close()

	uresp.Status = hresp.StatusCode
	uresp.Headers = hresp.Header.Clone()

	body, err := io.ReadAll(hresp.Body)
	if err != nil {
		uresp.Err = err
		return uresp
	}

	uresp.Body = body

	return uresp
}

func (u *httpUpstream) callWithRetry(ctx context.Context, original *http.Request, originalBody []byte, retryPolicy UpstreamRetryPolicy) *UpstreamResponse {
	resp := &UpstreamResponse{}

	for attempt := 0; attempt <= retryPolicy.MaxRetries; attempt++ {
		resp = u.Call(ctx, original, originalBody)
		if resp.Err == nil && !slices.Contains(retryPolicy.RetryOnStatuses, resp.Status) {
			break
		}

		if retryPolicy.BackoffMs > 0 {
			time.Sleep(time.Duration(retryPolicy.BackoffMs) * time.Millisecond)
		}
	}

	return resp
}

func (u *httpUpstream) newRequest(ctx context.Context, original *http.Request, originalBody []byte) (*http.Request, error) {
	method := u.method
	if method == "" {
		// Fallback method.
		method = original.Method
	}

	// Send request body only for body-acceptable methods requests.
	if method != http.MethodPost && method != http.MethodPut && method != http.MethodPatch {
		originalBody = nil
	}

	target, err := http.NewRequestWithContext(ctx, method, u.url, bytes.NewReader(originalBody))
	if err != nil {
		return nil, err
	}

	u.resolveQueryStrings(target, original)
	u.resolveHeaders(target, original)

	return target, nil
}

func (u *httpUpstream) resolveQueryStrings(target, original *http.Request) {
	q := target.URL.Query()

	for _, fqs := range u.forwardQueryStrings {
		if fqs == "*" {
			q = original.URL.Query()
			break
		}

		if original.URL.Query().Get(fqs) == "" {
			continue
		}

		q.Add(fqs, original.URL.Query().Get(fqs))
	}

	target.URL.RawQuery = q.Encode()
}

func (u *httpUpstream) resolveHeaders(target, original *http.Request) {
	// Set forwarding headers.
	for _, fw := range u.forwardHeaders {
		if fw == "*" {
			target.Header = original.Header.Clone()
			break
		}

		if strings.HasSuffix(fw, "*") {
			prefix := strings.TrimSuffix(fw, "*")

			for name, values := range original.Header {
				if strings.HasPrefix(name, prefix) {
					for _, v := range values {
						target.Header.Add(name, v)
					}
				}
			}

			continue
		}

		if original.Header.Get(fw) != "" {
			target.Header.Add(fw, original.Header.Get(fw))
		}
	}

	// Rewrite headers which exists in upstream headers configuration (rewriting only forwarded headers).
	for header, value := range u.headers {
		if !slices.Contains(u.forwardHeaders, header) {
			continue
		}

		target.Header.Set(header, value)
	}

	// Always forward the Content-Type header.
	target.Header.Set("Content-Type", original.Header.Get("Content-Type"))
}
