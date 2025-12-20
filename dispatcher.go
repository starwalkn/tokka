package tokka

//go:generate mockgen -source=dispatcher.go -destination=mock/dispatcher.go -package=mock dispatcher

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"sync"

	"go.uber.org/zap"

	"github.com/starwalkn/tokka/internal/metric"
)

type dispatcher interface {
	dispatch(route *Route, original *http.Request) []UpstreamResponse
}

type defaultDispatcher struct {
	log     *zap.Logger
	metrics metric.Metrics
}

func (d *defaultDispatcher) dispatch(route *Route, original *http.Request) []UpstreamResponse {
	results := make([]UpstreamResponse, len(route.Upstreams))

	originalBody, readErr := io.ReadAll(original.Body)
	if readErr != nil {
		d.log.Error("cannot read body", zap.Error(readErr))
		return nil
	}
	if readErr = original.Body.Close(); readErr != nil {
		d.log.Warn("cannot close original request body", zap.Error(readErr))
	}

	var wg sync.WaitGroup

	for i, u := range route.Upstreams {
		wg.Add(1)

		go func(i int, u Upstream, originalBody []byte) {
			defer wg.Done()

			upstreamPolicy := u.Policy()

			resp := u.callWithRetry(original.Context(), original, originalBody, upstreamPolicy.RetryPolicy)
			if resp.Err != nil {
				d.metrics.IncFailedRequestsTotal(metric.FailReasonUpstreamError)
				d.log.Error("cannot call upstream",
					zap.String("name", u.Name()),
					zap.Error(resp.Err),
				)
			}

			if resp.Status != 0 {
				d.metrics.IncResponsesTotal(resp.Status)
			}

			var errs []error

			if !slices.Contains(upstreamPolicy.AllowedStatuses, resp.Status) {
				errs = append(errs, fmt.Errorf("status %d not allowed by upstream policy", resp.Status))
			}
			if !upstreamPolicy.AllowEmptyBody && len(resp.Body) == 0 {
				errs = append(errs, errors.New("empty body not allowed by upstream policy"))
			}

			if len(errs) > 0 {
				resp.Err = errors.Join(errs...)
			}

			if mapped, ok := upstreamPolicy.MapStatusCodes[resp.Status]; ok {
				resp.Status = mapped
			}

			results[i] = *resp
		}(i, u, originalBody)
	}

	wg.Wait()

	return results
}
