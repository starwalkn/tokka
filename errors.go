package tokka

import "errors"

var (
	ErrUpstreamPolicyRequireBody     = errors.New("empty body not allowed by upstream policy")
	ErrUpstreamPolicyAllowedStatuses = errors.New("upstream policy forbids status")
)
