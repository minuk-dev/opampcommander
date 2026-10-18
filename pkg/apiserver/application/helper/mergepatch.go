package helper

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

// PatchResource applies a merge patch using the existing conditional update rules.
// Only callers without a revision opt into rebasing after a storage conflict.
func PatchResource[T any](ctx context.Context, patch []byte,
	get func(context.Context) (*T, error), update func(context.Context, *T) (*T, error),
) (*T, error) {
	var changes map[string]any

	err := decodePatchJSON(patch, &changes)
	if err != nil || changes == nil {
		return nil, fmt.Errorf("%w: patch must be a JSON object", model.ErrInvalidArgument)
	}

	err = validatePatch(changes, reflect.TypeFor[T](), "")
	if err != nil {
		return nil, err
	}

	conditional, err := patchPrecondition(changes)
	if err != nil {
		return nil, err
	}

	const attempts = 5
	for range attempts {
		err := ctx.Err()
		if err != nil {
			return nil, fmt.Errorf("patch canceled: %w", err)
		}

		current, err := get(ctx)
		if err != nil {
			return nil, err
		}

		merged, err := mergeResource(current, changes)
		if err != nil {
			return nil, err
		}

		result, err := update(ctx, merged)
		if conditional || !errors.Is(err, model.ErrConflict) {
			return result, err
		}
	}

	return nil, fmt.Errorf("%w: patch retry limit exceeded", model.ErrConflict)
}

func decodePatchJSON(data []byte, target any) error {
	if !json.Valid(data) {
		return model.ErrInvalidArgument
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()

	err := decoder.Decode(target)
	if err != nil {
		return fmt.Errorf("decode patch: %w", err)
	}

	return nil
}

func patchPrecondition(changes map[string]any) (bool, error) {
	metadata, _ := changes["metadata"].(map[string]any)

	revision, supplied := metadata["resourceVersion"]
	if !supplied {
		return false, nil
	}

	text, ok := revision.(string)

	version, err := strconv.ParseInt(text, 10, 64)
	if !ok || err != nil || version <= 0 || strings.Trim(text, "0123456789") != "" {
		return false, fmt.Errorf("%w: metadata.resourceVersion must be a positive revision string", model.ErrInvalidArgument)
	}

	return true, nil
}

func mergeResource[T any](current *T, changes map[string]any) (*T, error) {
	data, err := json.Marshal(current)
	if err != nil {
		return nil, fmt.Errorf("encode current resource: %w", err)
	}

	var target map[string]any

	err = decodePatchJSON(data, &target)
	if err != nil {
		return nil, err
	}

	for _, key := range []string{"kind", "apiVersion"} {
		if value, supplied := changes[key]; supplied && value != target[key] {
			return nil, fmt.Errorf("%w: %s cannot change", model.ErrInvalidArgument, key)
		}
	}

	metadata, _ := changes["metadata"].(map[string]any)

	currentMetadata, _ := target["metadata"].(map[string]any)
	for _, key := range []string{"name", "namespace"} {
		if value, supplied := metadata[key]; supplied && value != currentMetadata[key] {
			return nil, fmt.Errorf("%w: metadata.%s differs from request path", model.ErrInvalidArgument, key)
		}
	}

	data, err = json.Marshal(mergePatch(target, changes))
	if err != nil {
		return nil, fmt.Errorf("encode merged resource: %w", err)
	}

	var result T

	err = decodePatchJSON(data, &result)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", model.ErrInvalidArgument, err)
	}

	return &result, nil
}

// mergePatch follows RFC 7396; JSON numbers retain their original precision.
func mergePatch(target any, patch any) any {
	changes, object := patch.(map[string]any)
	if !object {
		return patch
	}

	result, ok := target.(map[string]any)
	if !ok {
		result = make(map[string]any)
	}

	for key, value := range changes {
		if value == nil {
			delete(result, key)
		} else {
			result[key] = mergePatch(result[key], value)
		}
	}

	return result
}

// validatePatch rejects unknown fields and removal of non-nullable schema fields.
func validatePatch(value any, schema reflect.Type, path string) error {
	if value == nil {
		if slices.Contains([]reflect.Kind{reflect.Map, reflect.Slice, reflect.Pointer}, schema.Kind()) {
			return nil
		}

		return fmt.Errorf("%w: %s cannot be null", model.ErrInvalidArgument, path)
	}

	if schema.Kind() == reflect.Pointer {
		schema = schema.Elem()
	}

	if array, ok := value.([]any); ok && schema.Kind() == reflect.Slice {
		for _, child := range array {
			err := validatePatch(child, schema.Elem(), path)
			if err != nil {
				return err
			}
		}

		return nil
	}

	object, isObject := value.(map[string]any)
	if !isObject {
		return nil
	} // The typed decode checks scalar/array types after merging.

	if schema.Kind() == reflect.Struct {
		return validatePatchFields(object, schema, path)
	}

	if schema.Kind() != reflect.Map {
		return fmt.Errorf("%w: %s must not be an object", model.ErrInvalidArgument, path)
	}

	for key, child := range object {
		if child == nil {
			continue
		} // Null deletes a map entry, even for non-nullable values.

		err := validatePatch(child, schema.Elem(), path+key+".")
		if err != nil {
			return err
		}
	}

	return nil
}

func validatePatchFields(object map[string]any, schema reflect.Type, path string) error {
	fields := make(map[string]reflect.Type)

	for field := range schema.Fields() {
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		fields[name] = field.Type
	}

	for key, child := range object {
		field, exists := fields[key]
		if !exists || (path == "" && key == "status") ||
			(path == "metadata." && (key == "createdAt" || key == "deletedAt")) {
			return fmt.Errorf("%w: field %s%s is not writable", model.ErrInvalidArgument, path, key)
		}

		err := validatePatch(child, field, path+key+".")
		if err != nil {
			return err
		}
	}

	return nil
}
