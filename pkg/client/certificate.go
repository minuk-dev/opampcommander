//nolint:dupl // Similar structure to other resource services is intentional
package client

import (
	"context"
	"fmt"
	"strconv"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

const (
	// ListCertificateURL is the path to list all certificates.
	ListCertificateURL = "/api/v1/namespaces/{namespace}/certificates"
	// GetCertificateURL is the path to get a certificate by name.
	GetCertificateURL = "/api/v1/namespaces/{namespace}/certificates/{id}"
	// CreateCertificateURL is the path to create a new certificate.
	CreateCertificateURL = "/api/v1/namespaces/{namespace}/certificates"
	// UpdateCertificateURL is the path to update an existing certificate.
	UpdateCertificateURL = "/api/v1/namespaces/{namespace}/certificates/{id}"
	// DeleteCertificateURL is the path to delete a certificate.
	DeleteCertificateURL = "/api/v1/namespaces/{namespace}/certificates/{id}"
)

// CertificateService provides methods to interact with certificates.
type CertificateService struct {
	service *service
}

// NewCertificateService creates a new CertificateService.
func NewCertificateService(service *service) *CertificateService {
	return &CertificateService{
		service: service,
	}
}

// GetCertificate retrieves a certificate by its namespace and name.
func (s *CertificateService) GetCertificate(
	ctx context.Context,
	namespace string,
	name string,
	opts ...GetOption,
) (*v1.Certificate, error) {
	var getSettings GetSettings
	for _, opt := range opts {
		opt.Apply(&getSettings)
	}

	var certificate v1.Certificate

	req := s.service.Resty.R().
		SetContext(ctx).
		SetResult(&certificate).
		SetPathParam("namespace", namespace).
		SetPathParam("id", name)

	if getSettings.includeDeleted != nil && *getSettings.includeDeleted {
		req.SetQueryParam("includeDeleted", "true")
	}

	res, err := req.Get(GetCertificateURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate(restyError): %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to get certificate(responseError): %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	return &certificate, nil
}

// CertificateListResponse represents a list of certificates with metadata.
type CertificateListResponse = v1.ListResponse[v1.Certificate]

// ListCertificates lists all certificates in a namespace.
func (s *CertificateService) ListCertificates(
	ctx context.Context,
	namespace string,
	opts ...ListOption,
) (*CertificateListResponse, error) {
	var listSettings ListSettings
	for _, opt := range opts {
		opt.Apply(&listSettings)
	}

	var listResponse CertificateListResponse

	req := s.service.Resty.R().
		SetContext(ctx).
		SetResult(&listResponse).
		SetPathParam("namespace", namespace)

	if listSettings.limit != nil {
		req.SetQueryParam("limit", strconv.Itoa(*listSettings.limit))
	}

	if listSettings.continueToken != nil {
		req.SetQueryParam("continue", *listSettings.continueToken)
	}

	if listSettings.includeDeleted != nil && *listSettings.includeDeleted {
		req.SetQueryParam("includeDeleted", "true")
	}

	res, err := req.Get(ListCertificateURL)
	if err != nil {
		return nil, fmt.Errorf("failed to list certificates(restyError): %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to list certificates(responseError): %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	return &listResponse, nil
}

// CreateCertificate creates a new certificate.
func (s *CertificateService) CreateCertificate(
	ctx context.Context,
	namespace string,
	createRequest *v1.Certificate,
) (*v1.Certificate, error) {
	var result v1.Certificate

	res, err := s.service.Resty.R().
		SetContext(ctx).
		SetPathParam("namespace", namespace).
		SetBody(createRequest).
		SetResult(&result).
		Post(CreateCertificateURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate(restyError): %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to create certificate(responseError): %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	return &result, nil
}

// UpdateCertificate updates an existing certificate.
func (s *CertificateService) UpdateCertificate(
	ctx context.Context,
	updateRequest *v1.Certificate,
) (*v1.Certificate, error) {
	var result v1.Certificate

	res, err := s.service.Resty.R().
		SetContext(ctx).
		SetPathParam("namespace", updateRequest.Metadata.Namespace).
		SetPathParam("id", updateRequest.Metadata.Name).
		SetBody(updateRequest).
		SetResult(&result).
		Put(UpdateCertificateURL)
	if err != nil {
		return nil, fmt.Errorf("failed to update certificate(restyError): %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to update certificate(responseError): %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	return &result, nil
}

// DeleteCertificate deletes a certificate by its namespace and name.
func (s *CertificateService) DeleteCertificate(
	ctx context.Context,
	namespace string,
	name string,
) error {
	res, err := s.service.Resty.R().
		SetContext(ctx).
		SetPathParam("namespace", namespace).
		SetPathParam("id", name).
		Delete(DeleteCertificateURL)
	if err != nil {
		return fmt.Errorf("failed to delete certificate(restyError): %w", err)
	}

	if res.IsError() {
		return fmt.Errorf("failed to delete certificate(responseError): %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	return nil
}
