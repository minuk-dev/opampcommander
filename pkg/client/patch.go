package client

import (
	"context"
	"fmt"
)

func patchResource[T any](ctx context.Context, svc *service, path, namespace, name string, patch []byte) (*T, error) {
	var result T

	response, err := svc.Resty.R().SetContext(ctx).
		SetHeader("Content-Type", "application/merge-patch+json").
		SetPathParam("namespace", namespace).SetPathParam("id", name).
		SetBody(patch).SetResult(&result).Patch(path)
	if err != nil {
		return nil, fmt.Errorf("patch resource: %w", err)
	}

	if response.IsError() {
		return nil, &ResponseError{StatusCode: response.StatusCode(), ErrorMessage: response.String()}
	}

	return &result, nil
}
