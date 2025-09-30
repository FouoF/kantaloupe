package bff

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"k8s.io/klog/v2"

	credentialv1alpha1 "github.com/dynamia-ai/kantaloupe/api/credentials/v1alpha1"
	kantaloupeapi "github.com/dynamia-ai/kantaloupe/api/v1"
	credentialservice "github.com/dynamia-ai/kantaloupe/pkg/service/credential"
	"github.com/dynamia-ai/kantaloupe/pkg/utils"
)

var _ kantaloupeapi.CredentialServer = &CredentialHandler{}

// CredentialHandler handles credential related API requests.
type CredentialHandler struct {
	credentialService credentialservice.Service
	kantaloupeapi.UnimplementedCredentialServer
}

// NewCredentialHandler creates a new credential handler.
func NewCredentialHandler(credentialService credentialservice.Service) *CredentialHandler {
	return &CredentialHandler{
		credentialService: credentialService,
	}
}

// ListCredentials lists credentials with optional filtering.
func (h *CredentialHandler) ListCredentials(ctx context.Context, req *credentialv1alpha1.ListCredentialsRequest) (*credentialv1alpha1.ListCredentialsResponse, error) {
	// Get credentials from service
	credentials, err := h.credentialService.ListCredentials(ctx, req.GetType(), req.GetNamespace())
	if err != nil {
		klog.ErrorS(err, "failed to list credentials", "Error:", err)
		return nil, status.Errorf(codes.Internal, "failed to list credentials: %v", err)
	}

	paged := utils.PagedItems(credentials, req.GetPage(), req.GetPageSize())

	// Convert to response objects
	items := convertSecretsToCredentialResponses(paged)

	return &credentialv1alpha1.ListCredentialsResponse{
		Items:      items,
		Pagination: utils.NewPage(req.Page, req.PageSize, len(credentials)),
	}, nil
}

// CreateCredential creates a new credential.
func (h *CredentialHandler) CreateCredential(ctx context.Context, req *credentialv1alpha1.CreateCredentialRequest) (*credentialv1alpha1.CredentialResponse, error) {
	// Validate request
	if err := validateCreateRequest(req); err != nil {
		return nil, err
	}

	// Create the credential
	secret, err := h.credentialService.CreateCredential(
		ctx,
		req.GetName(),
		req.GetNamespace(),
		req.GetCluster(),
		req.GetType(),
		req.GetData(),
	)
	if err != nil {
		klog.ErrorS(err, "failed to create credential", "name", req.GetName(), "Error:", err)
		return nil, status.Errorf(codes.Internal, "failed to create credential: %v", err)
	}

	// Convert to response
	return convertSecretToCredentialResponse(secret)
}

// UpdateCredential updates an existing credential.
func (h *CredentialHandler) UpdateCredential(ctx context.Context, req *credentialv1alpha1.UpdateCredentialRequest) (*credentialv1alpha1.CredentialResponse, error) {
	// Validate request
	if err := validateUpdateRequest(req); err != nil {
		return nil, err
	}

	// Update the credential
	secret, err := h.credentialService.UpdateCredential(
		ctx,
		req.GetName(),
		req.GetNamespace(),
		req.GetCluster(),
		req.GetType(),
		req.GetData(),
	)
	if err != nil {
		klog.ErrorS(err, "failed to update credential", "name", req.GetName(), "Error:", err)
		return nil, status.Errorf(codes.Internal, "failed to update credential: %v", err)
	}

	// Convert to response
	return convertSecretToCredentialResponse(secret)
}

// DeleteCredential deletes a credential.
func (h *CredentialHandler) DeleteCredential(ctx context.Context, req *credentialv1alpha1.DeleteCredentialRequest) (*emptypb.Empty, error) {
	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "credential name cannot be empty")
	}

	// Delete credential
	err := h.credentialService.DeleteCredential(ctx, req.GetName(), req.GetNamespace())
	if err != nil {
		klog.ErrorS(err, "failed to delete credential", "name", req.GetName(), "Error:", err)
		return nil, status.Errorf(codes.Internal, "failed to delete credential: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// validateCreateRequest validates CreateCredentialRequest.
func validateCreateRequest(req *credentialv1alpha1.CreateCredentialRequest) error {
	if req.GetName() == "" {
		return status.Error(codes.InvalidArgument, "credential name cannot be empty")
	}

	if req.GetType() == credentialv1alpha1.CredentialType_CREDENTIAL_TYPE_UNSPECIFIED {
		return status.Error(codes.InvalidArgument, "credential type cannot be unspecified")
	}

	if len(req.GetData()) == 0 {
		return status.Error(codes.InvalidArgument, "credential data cannot be empty")
	}

	// Validate type-specific required fields
	return validateTypeSpecificFields(req.GetType(), req.GetData())
}

// validateUpdateRequest validates UpdateCredentialRequest.
func validateUpdateRequest(req *credentialv1alpha1.UpdateCredentialRequest) error {
	if req.GetName() == "" {
		return status.Error(codes.InvalidArgument, "credential name cannot be empty")
	}

	if req.GetType() == credentialv1alpha1.CredentialType_CREDENTIAL_TYPE_UNSPECIFIED {
		return status.Error(codes.InvalidArgument, "credential type cannot be unspecified")
	}

	if len(req.GetData()) == 0 {
		return status.Error(codes.InvalidArgument, "credential data cannot be empty")
	}

	// Validate type-specific required fields
	return validateTypeSpecificFields(req.GetType(), req.GetData())
}

// validateTypeSpecificFields validates fields based on credential type.
func validateTypeSpecificFields(credType credentialv1alpha1.CredentialType, data map[string]string) error {
	switch credType {
	case credentialv1alpha1.CredentialType_CREDENTIAL_TYPE_UNSPECIFIED:
		// This case should not be reached due to prior validation,
		// but we handle it to satisfy the exhaustive check
		return status.Error(codes.InvalidArgument, "credential type cannot be unspecified")
	case credentialv1alpha1.CredentialType_DOCKER_REGISTRY:
		if _, ok := data["server"]; !ok {
			return status.Error(codes.InvalidArgument, "server is required for Docker registry credential")
		}
		if _, ok := data["username"]; !ok {
			return status.Error(codes.InvalidArgument, "username is required for Docker registry credential")
		}
		if _, ok := data["password"]; !ok {
			return status.Error(codes.InvalidArgument, "password is required for Docker registry credential")
		}
	case credentialv1alpha1.CredentialType_ACCESS_KEY:
		if _, ok := data["accessKey"]; !ok {
			return status.Error(codes.InvalidArgument, "accessKey is required for access key credential")
		}
		if _, ok := data["secretKey"]; !ok {
			return status.Error(codes.InvalidArgument, "secretKey is required for access key credential")
		}
	}
	return nil
}
