package bff

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	credentialv1alpha1 "github.com/dynamia-ai/kantaloupe/api/credentials/v1alpha1"
)

// ConvertSecretsToCredentialResponses converts Kubernetes secrets to credential responses.
func convertSecretsToCredentialResponses(secrets []*corev1.Secret) []*credentialv1alpha1.CredentialResponse {
	items := make([]*credentialv1alpha1.CredentialResponse, 0, len(secrets))
	for _, secret := range secrets {
		resp, err := convertSecretToCredentialResponse(secret)
		if err != nil {
			klog.ErrorS(err, "Failed to convert credential")
			continue
		}
		items = append(items, resp)
	}
	return items
}

// ConvertSecretToCredentialResponse converts a Kubernetes secret to a credential response.
func convertSecretToCredentialResponse(secret *corev1.Secret) (*credentialv1alpha1.CredentialResponse, error) {
	if secret == nil {
		return nil, status.Error(codes.Internal, "secret is nil")
	}

	// Determine credential type based on secret type and contents
	var credType credentialv1alpha1.CredentialType
	if secret.Type == corev1.SecretTypeDockerConfigJson {
		credType = credentialv1alpha1.CredentialType_DOCKER_REGISTRY
	} else if _, hasAccessKey := secret.Data["accessKey"]; hasAccessKey {
		credType = credentialv1alpha1.CredentialType_ACCESS_KEY
	} else {
		credType = credentialv1alpha1.CredentialType_CREDENTIAL_TYPE_UNSPECIFIED
	}

	return &credentialv1alpha1.CredentialResponse{
		Name:        secret.Name,
		Type:        credType,
		Namespace:   secret.Namespace,
		CreatedTime: secret.CreationTimestamp.Unix(),
		Labels:      secret.Labels,
	}, nil
}
