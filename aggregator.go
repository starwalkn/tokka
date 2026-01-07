package tokka

import (
	"encoding/json"
	"errors"
	"maps"

	"go.uber.org/zap"
)

const (
	strategyMerge = "merge"
	strategyArray = "array"
)

type AggregatedResponse struct {
	Data    json.RawMessage
	Errors  []JSONError
	Partial bool
}

type aggregator interface {
	aggregate(responses []UpstreamResponse, mode string, allowPartialResults bool) AggregatedResponse
}

type defaultAggregator struct {
	log *zap.Logger
}

func (a *defaultAggregator) aggregate(responses []UpstreamResponse, mode string, allowPartialResults bool) AggregatedResponse {
	switch mode {
	case strategyMerge:
		return a.doMerge(responses, allowPartialResults)
	case strategyArray:
		return a.doArray(responses, allowPartialResults)
	default:
		a.log.Error("unknown aggregation strategy", zap.String("strategy", mode))
		return AggregatedResponse{}
	}
}

func (a *defaultAggregator) doMerge(responses []UpstreamResponse, allowPartialResults bool) AggregatedResponse {
	merged := make(map[string]any)

	var aggregationErrors []JSONError

	for _, resp := range responses {
		var obj map[string]any

		if resp.Body == nil {
			continue
		}

		// Handle upstream error.
		if resp.Err != nil {
			if !allowPartialResults {
				return internalAggregationError()
			} else {
				aggregationErrors = append(aggregationErrors, a.mapUpstreamError(resp.Err))

				a.log.Warn(
					"failed to unmarshal response",
					zap.Bool("allow_partial_results", allowPartialResults),
					zap.Error(resp.Err),
				)

				continue
			}
		}

		// Handle JSON unmarshaling error as internal.
		if err := json.Unmarshal(resp.Body, &obj); err != nil {
			if !allowPartialResults {
				return internalAggregationError()
			}

			aggregationErrors = append(aggregationErrors, JSONError{
				Code:    ErrorCodeInternal,
				Message: "server error",
			})

			a.log.Warn(
				"failed to unmarshal response",
				zap.Bool("allow_partial_results", allowPartialResults),
				zap.Error(err),
			)

			continue
		}

		maps.Copy(merged, obj)
	}

	data, err := json.Marshal(merged)
	if err != nil {
		return internalAggregationError()
	}

	aggregationResponse := AggregatedResponse{
		Data:    data,
		Errors:  dedupeErrors(aggregationErrors),
		Partial: len(aggregationErrors) > 0,
	}

	return aggregationResponse
}

func (a *defaultAggregator) doArray(responses []UpstreamResponse, allowPartialResults bool) AggregatedResponse {
	var arr []json.RawMessage

	var aggregationErrors []JSONError

	for _, resp := range responses {
		if resp.Body == nil {
			continue
		}

		// Handle upstream error.
		if resp.Err != nil {
			if !allowPartialResults {
				return internalAggregationError()
			}

			aggregationErrors = append(aggregationErrors, a.mapUpstreamError(resp.Err))

			a.log.Warn(
				"failed to unmarshal response",
				zap.Bool("allow_partial_results", allowPartialResults),
				zap.Error(resp.Err),
			)

			continue
		}

		arr = append(arr, resp.Body)
	}

	data, err := json.Marshal(arr)
	if err != nil {
		return internalAggregationError()
	}

	aggregationResponse := AggregatedResponse{
		Data:    data,
		Errors:  dedupeErrors(aggregationErrors),
		Partial: len(aggregationErrors) > 0,
	}

	return aggregationResponse
}

func (a *defaultAggregator) mapUpstreamError(err error) JSONError {
	var ue *UpstreamError

	if !errors.As(err, &ue) {
		return JSONError{
			Code:    ErrorCodeInternal,
			Message: "internal error",
		}
	}

	switch ue.Kind {
	case UpstreamTimeout, UpstreamConnection:
		return JSONError{
			Code:    ErrorCodeUpstreamUnavailable,
			Message: "service temporarily unavailable",
		}
	case UpstreamBadStatus:
		return JSONError{
			Code:    ErrorCodeUpstreamError,
			Message: "upstream error",
		}
	default:
		return JSONError{
			Code:    ErrorCodeInternal,
			Message: "internal error",
		}
	}
}

func internalAggregationError() AggregatedResponse {
	return AggregatedResponse{
		Data: nil,
		Errors: []JSONError{
			{
				Code:    ErrorCodeInternal,
				Message: "internal error",
			},
		},
		Partial: false,
	}
}

func dedupeErrors(errs []JSONError) []JSONError {
	seen := make(map[string]struct{})
	out := make([]JSONError, 0, len(errs))

	for _, e := range errs {
		if _, ok := seen[e.Code]; ok {
			continue
		}

		seen[e.Code] = struct{}{}
		out = append(out, e)
	}

	return out
}
