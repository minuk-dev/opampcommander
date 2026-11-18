package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

// InstallationLabel assigns a namespace to an administrator-selected installation.
const InstallationLabel = "opampcommander.io/installation-uid"

// Installation pins this process to one incarnation of the installation ConfigMap.
type Installation struct {
	UID    types.UID
	key    client.ObjectKey
	reader client.Reader
}

// LoadInstallation uses an uncached reader before the manager starts.
func LoadInstallation(ctx context.Context, reader client.Reader, key client.ObjectKey) (*Installation, error) {
	if key.Namespace == "" || key.Name == "" {
		return nil, fmt.Errorf("installation ConfigMap namespace and name are required")
	}
	anchor := &corev1.ConfigMap{}
	if err := reader.Get(ctx, key, anchor); err != nil {
		return nil, fmt.Errorf("read installation ConfigMap: %w", err)
	}
	if anchor.UID == "" || !anchor.DeletionTimestamp.IsZero() {
		return nil, fmt.Errorf("installation ConfigMap has no live identity")
	}
	return &Installation{UID: anchor.UID, key: key, reader: reader}, nil
}

// Check fails closed if the anchor disappears or is replaced while this process runs.
func (i *Installation) Check(ctx context.Context) error {
	if i == nil {
		return fmt.Errorf("installation identity is not configured")
	}
	anchor := &corev1.ConfigMap{}
	if err := i.reader.Get(ctx, i.key, anchor); err != nil {
		return fmt.Errorf("read installation ConfigMap: %w", err)
	}
	if anchor.UID != i.UID || !anchor.DeletionTimestamp.IsZero() {
		return fmt.Errorf("installation ConfigMap identity changed or is being deleted")
	}
	return nil
}

func (i *Installation) ownsNamespace(ctx context.Context, name string) (bool, error) {
	if err := i.Check(ctx); err != nil {
		return false, err
	}
	namespace := &corev1.Namespace{}
	if err := i.reader.Get(ctx, client.ObjectKey{Name: name}, namespace); err != nil {
		return false, client.IgnoreNotFound(err)
	}
	return namespace.DeletionTimestamp.IsZero() && namespace.Labels[InstallationLabel] == string(i.UID), nil
}

// Assignment changes must enqueue existing CRs even when their specs did not change.
func enqueueNamespace(reader client.Reader, newList func() client.ObjectList) handler.MapFunc {
	return func(ctx context.Context, namespace client.Object) []ctrl.Request {
		list := newList()
		if err := reader.List(ctx, list, client.InNamespace(namespace.GetName())); err != nil {
			ctrl.LoggerFrom(ctx).Error(err, "Failed to list namespace configuration")
			return nil
		}
		var requests []ctrl.Request
		if err := meta.EachListItem(list, func(object runtime.Object) error {
			metadata, err := meta.Accessor(object)
			if err != nil {
				return err
			}
			requests = append(requests, ctrl.Request{NamespacedName: client.ObjectKey{Namespace: metadata.GetNamespace(), Name: metadata.GetName()}})
			return nil
		}); err != nil {
			ctrl.LoggerFrom(ctx).Error(err, "Failed to read namespace configuration list")
		}
		return requests
	}
}
