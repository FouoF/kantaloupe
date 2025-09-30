package credential

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	credentialv1alpha1 "github.com/dynamia-ai/kantaloupe/api/credentials/v1alpha1"
	"github.com/dynamia-ai/kantaloupe/pkg/constants"
	"github.com/dynamia-ai/kantaloupe/pkg/engine"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/namespace"
)

// Service defines the interface for credential operations.
type Service interface {
	// ListCredentials lists credentials with filtering options.
	ListCredentials(ctx context.Context, credType credentialv1alpha1.CredentialType, namespace string) ([]*corev1.Secret, error)

	// CreateCredential creates a new credential.
	CreateCredential(ctx context.Context, name, ns, cluster string, credType credentialv1alpha1.CredentialType, data map[string]string) (*corev1.Secret, error)

	// UpdateCredential updates an existing credential.
	UpdateCredential(ctx context.Context, name, ns, cluster string, credType credentialv1alpha1.CredentialType, data map[string]string) (*corev1.Secret, error)

	// DeleteCredential deletes a credential by name.
	DeleteCredential(ctx context.Context, name, namespace string) error
}

// service implements the Service interface.
type service struct {
	clientManager engine.ClientManagerInterface
}

// NewService creates a new credential service.
func NewService(clientManager engine.ClientManagerInterface) Service {
	return &service{
		clientManager: clientManager,
	}
}

// getNamespace returns the namespace, defaulting if necessary.
func getNamespace(ns string) string {
	if ns == "" {
		return namespace.GetCurrentNamespaceOrDefault()
	}
	return ns
}

// ListCredentials lists credentials with optional filtering.
func (s *service) ListCredentials(ctx context.Context, credType credentialv1alpha1.CredentialType, namespace string) ([]*corev1.Secret, error) {
	client, err := s.clientManager.GeteClient(engine.LocalCluster)
	if err != nil {
		klog.ErrorS(err, "failed to get Kubernetes client")
		return nil, err
	}

	// Build label selector for managed resources
	labelSelector := constants.ManagedByLabelKey + "=" + constants.ManagedByLabelValue

	// Add type-specific label for Docker Registry
	if credType == credentialv1alpha1.CredentialType_DOCKER_REGISTRY {
		labelSelector += "," + constants.CredentialTypeLabelKey + "=" + constants.CredentialTypeDockerRegistry
	}

	// List secrets with label filtering
	var secretList *corev1.SecretList
	var listErr error

	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	if namespace == "" {
		secretList, listErr = client.CoreV1().Secrets(metav1.NamespaceAll).List(ctx, listOptions)
	} else {
		secretList, listErr = client.CoreV1().Secrets(namespace).List(ctx, listOptions)
	}

	if listErr != nil {
		klog.ErrorS(listErr, "failed to list secrets", "labelSelector", labelSelector)
		return nil, listErr
	}

	// Filter results by type
	// TODO: remove to bff
	filteredSecrets := make([]*corev1.Secret, 0, len(secretList.Items))
	for i := range secretList.Items {
		secret := &secretList.Items[i]

		if credType == credentialv1alpha1.CredentialType_DOCKER_REGISTRY {
			if secret.Type == corev1.SecretTypeDockerConfigJson {
				filteredSecrets = append(filteredSecrets, secret)
			}
		} else if credType == credentialv1alpha1.CredentialType_CREDENTIAL_TYPE_UNSPECIFIED {
			if secret.Type == corev1.SecretTypeDockerConfigJson {
				filteredSecrets = append(filteredSecrets, secret)
			}
		}
	}

	return filteredSecrets, nil
}

// CreateCredential creates a new credential.
func (s *service) CreateCredential(
	ctx context.Context,
	name, ns, cluster string,
	credType credentialv1alpha1.CredentialType,
	data map[string]string,
) (*corev1.Secret, error) {
	client, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		klog.ErrorS(err, "failed to get Kubernetes client", "cluster", cluster)
		return nil, err
	}

	// Prepare the secret based on credential type
	secret, err := prepareSecret(name, getNamespace(ns), credType, data)
	if err != nil {
		klog.ErrorS(err, "failed to prepare secret", "name", name, "type", credType)
		return nil, err
	}

	// Add management labels
	addCredentialLabels(secret, credType)

	// Create the secret
	createdSecret, err := client.CoreV1().Secrets(secret.Namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		klog.ErrorS(err, "failed to create secret", "name", name, "namespace", secret.Namespace)
		return nil, err
	}

	klog.V(4).InfoS("created credential", "name", name, "namespace", secret.Namespace, "type", credType)
	return createdSecret, nil
}

// UpdateCredential updates an existing credential.
func (s *service) UpdateCredential(
	ctx context.Context,
	name, ns, cluster string,
	credType credentialv1alpha1.CredentialType,
	data map[string]string,
) (*corev1.Secret, error) {
	client, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		klog.ErrorS(err, "failed to get Kubernetes client", "cluster", cluster)
		return nil, err
	}

	namespace := getNamespace(ns)

	// Get existing secret
	existingSecret, err := client.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		klog.ErrorS(err, "failed to get existing secret", "name", name, "namespace", namespace)
		return nil, err
	}

	// Prepare new secret data
	updatedSecret, err := prepareSecret(name, namespace, credType, data)
	if err != nil {
		klog.ErrorS(err, "failed to prepare updated secret", "name", name, "type", credType)
		return nil, err
	}

	// Preserve metadata
	updatedSecret.ResourceVersion = existingSecret.ResourceVersion
	if existingSecret.Labels != nil {
		for k, v := range existingSecret.Labels {
			if updatedSecret.Labels == nil {
				updatedSecret.Labels = make(map[string]string)
			}
			updatedSecret.Labels[k] = v
		}
	}
	if existingSecret.Annotations != nil {
		updatedSecret.Annotations = existingSecret.Annotations
	}

	// Update management labels
	addCredentialLabels(updatedSecret, credType)

	// Update the secret
	updatedSecret, err = client.CoreV1().Secrets(namespace).Update(ctx, updatedSecret, metav1.UpdateOptions{})
	if err != nil {
		klog.ErrorS(err, "failed to update secret", "name", name, "namespace", namespace)
		return nil, err
	}

	klog.V(4).InfoS("updated credential", "name", name, "namespace", namespace, "type", credType)
	return updatedSecret, nil
}

// DeleteCredential deletes a credential by name.
func (s *service) DeleteCredential(ctx context.Context, name, namespace string) error {
	client, err := s.clientManager.GeteClient(engine.LocalCluster)
	if err != nil {
		klog.ErrorS(err, "failed to get Kubernetes client")
		return err
	}

	namespace = getNamespace(namespace)

	// Verify it's a credential we manage
	secret, err := client.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			klog.V(4).InfoS("credential not found, skipping deletion", "name", name, "namespace", namespace)
			return nil
		}
		klog.ErrorS(err, "failed to get secret for deletion", "name", name, "namespace", namespace)
		return err
	}

	// Only delete Docker Registry secrets
	if secret.Type != corev1.SecretTypeDockerConfigJson {
		klog.V(4).InfoS("secret is not of Docker Registry type, skipping deletion", "name", name, "namespace", namespace)
		return nil
	}

	err = client.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		klog.ErrorS(err, "failed to delete secret", "name", name, "namespace", namespace)
		return err
	}

	klog.V(4).InfoS("deleted credential", "name", name, "namespace", namespace)
	return nil
}

// prepareSecret creates a Secret object based on credential type and data.
func prepareSecret(name, namespace string, credType credentialv1alpha1.CredentialType, data map[string]string) (*corev1.Secret, error) {
	switch credType {
	case credentialv1alpha1.CredentialType_CREDENTIAL_TYPE_UNSPECIFIED:
		return nil, fmt.Errorf("credential type cannot be unspecified")
	case credentialv1alpha1.CredentialType_DOCKER_REGISTRY:
		return prepareDockerRegistrySecret(name, namespace, data)
	case credentialv1alpha1.CredentialType_ACCESS_KEY:
		return prepareAccessKeySecret(name, namespace, data)
	default:
		return nil, fmt.Errorf("unsupported credential type: %v", credType)
	}
}

// prepareDockerRegistrySecret creates a Docker registry secret.
func prepareDockerRegistrySecret(name, namespace string, data map[string]string) (*corev1.Secret, error) {
	// Validate required fields
	server, ok := data["server"]
	if !ok || server == "" {
		return nil, fmt.Errorf("server is required for Docker registry credential")
	}

	username, ok := data["username"]
	if !ok || username == "" {
		return nil, fmt.Errorf("username is required for Docker registry credential")
	}

	password, ok := data["password"]
	if !ok || password == "" {
		return nil, fmt.Errorf("password is required for Docker registry credential")
	}

	// Create auth structure
	authConfig := map[string]interface{}{
		"auths": map[string]interface{}{
			server: map[string]string{
				"username": username,
				"password": password,
				"auth":     base64.StdEncoding.EncodeToString([]byte(username + ":" + password)),
			},
		},
	}

	// Convert to JSON
	authJSON, err := json.Marshal(authConfig)
	if err != nil {
		return nil, err
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: authJSON,
		},
	}, nil
}

// prepareAccessKeySecret creates an access key secret.
func prepareAccessKeySecret(name, namespace string, data map[string]string) (*corev1.Secret, error) {
	// Validate required fields
	accessKey, ok := data["accessKey"]
	if !ok || accessKey == "" {
		return nil, fmt.Errorf("accessKey is required for access key credential")
	}

	secretKey, ok := data["secretKey"]
	if !ok || secretKey == "" {
		return nil, fmt.Errorf("secretKey is required for access key credential")
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"accessKey": accessKey,
			"secretKey": secretKey,
		},
	}, nil
}

// addCredentialLabels adds management labels to a secret.
func addCredentialLabels(secret *corev1.Secret, credType credentialv1alpha1.CredentialType) {
	if secret.Labels == nil {
		secret.Labels = make(map[string]string)
	}

	// Mark as managed by kantaloupe
	secret.Labels[constants.ManagedByLabelKey] = constants.ManagedByLabelValue

	// Add type-specific label
	switch credType {
	case credentialv1alpha1.CredentialType_CREDENTIAL_TYPE_UNSPECIFIED:
		// No specific label for unspecified type
	case credentialv1alpha1.CredentialType_DOCKER_REGISTRY:
		secret.Labels[constants.CredentialTypeLabelKey] = constants.CredentialTypeDockerRegistry
	case credentialv1alpha1.CredentialType_ACCESS_KEY:
		// No specific label for access key type yet
	}
}
