package constants

import "k8s.io/kube-openapi/pkg/validation/errors"

var ErrInvalidStorageType = errors.New(401, "invalid storage type")
