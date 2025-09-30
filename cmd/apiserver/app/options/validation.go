package options

import (
	"github.com/dynamia-ai/kantaloupe/pkg/utils/errs"
)

// Validate validates server run options, to find
// options' misconfiguration.
func (o *Options) Validate() []error {
	var errors []error

	errors = append(errors, o.ServerRunOptions.Validate()...)

	return errors
}

func (s *ServerRunOptions) Validate() []error {
	var errList []error

	if s.SecurePort == 0 && s.InsecurePort == 0 {
		errList = append(errList, errs.ErrHTTPServerPortAllDisabled)
	}

	return errList
}
