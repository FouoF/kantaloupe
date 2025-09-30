package namespace

import (
	"os"
	"strings"

	"github.com/dynamia-ai/kantaloupe/pkg/utils/errs"
)

func GetCurrentNamespaceOrDefault() string {
	ns, err := GetCurrentNamespace()
	if err != nil {
		return "kantaloupe-system"
	}
	return ns
}

// GetCurrentNamespace fetch namespace the current pod running in.
//
//	Reference to client-go (config *inClusterClientConfig) Namespace() (string, bool, error).
func GetCurrentNamespace() (string, error) {
	if ns := os.Getenv("POD_NAMESPACE"); ns != "" {
		return ns, nil
	}

	if data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns, nil
		}
	}
	return "", errs.ErrCurrentNamespaceNotFound
}
