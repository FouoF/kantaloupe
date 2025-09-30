package errs

import "errors"

// ErrDefaultNamespaceNotFound is returned when the namespace cannot be found in the default locations.
var ErrCurrentNamespaceNotFound = errors.New("current namespace not found")

// ErrHTTPServerPortAllDisabled is returned when the insecure and secure port can not be disabled at the same time.
var ErrHTTPServerPortAllDisabled = errors.New("insecure and secure port can not be disabled at the same time")

// ErrPrometheusClientUninitialized is returned when the prometheus client is uninitialized.
var ErrPrometheusClientUninitialized = errors.New("prometheus client uninitialized")
