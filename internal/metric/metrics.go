package metric

//go:generate mockgen -source=metrics.go -destination=mock/metrics.go -package=mock Metrics

import "time"

type FailReason string

const (
	FailReasonGatewayError   FailReason = "gateway_error"
	FailReasonUpstreamError  FailReason = "upstream_error"
	FailReasonNoMatchedRoute FailReason = "no_matched_route"
)

type Metrics interface {
	IncRequestsTotal()
	UpdateRequestsDuration(time.Time)
	IncResponsesTotal(int)
	IncRequestsInFlight()
	DecRequestsInFlight()
	IncFailedRequestsTotal(FailReason)
}
