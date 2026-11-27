package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	api "github.com/minuk-dev/opampcommander/operator/api/v1alpha1"
)

const (
	dependencyIndex  = "opampcommander.io/dependency"
	secretKind       = "Secret"
	remoteConfigKind = "RemoteConfig"
	packageKind      = "AgentPackage"
	schemaKind       = "RemoteConfigSchema"
	endpointKind     = "Endpoint"
)

type dependency struct {
	kind string
	name string
}

func configurationDependencies(object client.Object) []dependency {
	var refs []dependency
	switch object := object.(type) {
	case *api.AgentGroup:
		for _, ref := range object.Spec.RemoteConfigs {
			refs = append(refs, dependency{remoteConfigKind, ref.Name})
		}
		for _, ref := range object.Spec.PackageRefs {
			refs = append(refs, dependency{packageKind, ref.Name})
		}
	case *api.RemoteConfig:
		for _, ref := range object.Spec.SchemaRefs {
			refs = append(refs, dependency{schemaKind, ref.Name})
		}
		for _, ref := range object.Spec.EndpointRefs {
			refs = append(refs, dependency{endpointKind, ref.Name})
		}
	}
	return refs
}

func secretReferences(object client.Object) []api.SecretKeyReference {
	var refs []api.SecretKeyReference
	add := func(ref *api.SecretKeyReference) {
		if ref != nil {
			refs = append(refs, *ref)
		}
	}
	switch object := object.(type) {
	case *api.AgentGroup:
		add(object.Spec.ConnectionSettingsSecretRef)
	case *api.RemoteConfig:
		add(object.Spec.ValueFrom)
	case *api.AgentPackage:
		add(object.Spec.SignatureSecretRef)
		for _, header := range object.Spec.Headers {
			add(&header.ValueFrom)
		}
	case *api.Endpoint:
		for _, tenant := range object.Spec.Tenants {
			for _, header := range tenant.Headers {
				add(&header.ValueFrom)
			}
		}
	}
	return refs
}

func dependencyKeys(object client.Object) []string {
	refs := configurationDependencies(object)
	secrets := secretReferences(object)
	keys := make([]string, 0, len(refs)+len(secrets))
	for _, ref := range refs {
		keys = append(keys, ref.kind+"/"+ref.name)
	}
	for _, ref := range secrets {
		keys = append(keys, secretKind+"/"+ref.Name)
	}
	return keys
}

func (ref dependency) object() client.Object {
	switch ref.kind {
	case remoteConfigKind:
		return &api.RemoteConfig{}
	case packageKind:
		return &api.AgentPackage{}
	case schemaKind:
		return &api.RemoteConfigSchema{}
	case endpointKind:
		return &api.Endpoint{}
	default:
		// Secret watches must not cache private values.
		return &metav1.PartialObjectMetadata{TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: secretKind}}
	}
}

type configurationInputs struct {
	seen       map[dependency]bool
	secretKeys []api.SecretKeyReference
	revisions  []string
}

func (i *configurationInputs) observe(kind string, object client.Object, revision string) {
	i.revisions = append(i.revisions, fmt.Sprintf("%q/%q/%q/%q", kind, object.GetName(), object.GetUID(), revision))
}

func (i *configurationInputs) fingerprint() string {
	slices.Sort(i.revisions)
	sum := sha256.Sum256([]byte(strings.Join(i.revisions, "\n")))
	return hex.EncodeToString(sum[:])
}

// referencesFound reads the current dependency graph without copying configuration to Mongo.
// It collects Secret key references for a separate uncached check.
func (r *AgentGroupReconciler) referencesFound(ctx context.Context, object client.Object, inputs *configurationInputs) (bool, error) {
	inputs.secretKeys = append(inputs.secretKeys, secretReferences(object)...)
	for _, ref := range configurationDependencies(object) {
		if inputs.seen[ref] {
			continue
		}
		inputs.seen[ref] = true
		child := ref.object()
		if err := r.Get(ctx, client.ObjectKey{Namespace: object.GetNamespace(), Name: ref.name}, child); err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			return false, fmt.Errorf("read %s dependency: %w", ref.kind, err)
		}
		if !child.GetDeletionTimestamp().IsZero() {
			return false, nil
		}
		inputs.observe(ref.kind, child, strconv.FormatInt(child.GetGeneration(), 10))
		found, err := r.referencesFound(ctx, child, inputs)
		if err != nil || !found {
			return found, err
		}
	}
	return true, nil
}

// secretKeysAvailable checks both existence and key availability with one read per Secret.
func secretKeysAvailable(ctx context.Context, reader client.Reader, namespace string, inputs *configurationInputs) (found, available bool, err error) {
	keysByName := make(map[string][]string)
	var names []string
	for _, ref := range inputs.secretKeys {
		if _, exists := keysByName[ref.Name]; !exists {
			names = append(names, ref.Name)
		}
		keysByName[ref.Name] = append(keysByName[ref.Name], ref.Key)
	}
	available = true
	for _, name := range names {
		secret := &corev1.Secret{}
		if err := reader.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, secret); err != nil {
			if apierrors.IsNotFound(err) {
				return false, false, nil
			}
			return false, false, errors.New("could not check Secret keys")
		}
		if !secret.DeletionTimestamp.IsZero() {
			return false, false, nil
		}
		for _, key := range keysByName[name] {
			if len(secret.Data[key]) == 0 {
				available = false
				break
			}
		}
		inputs.observe(secretKind, secret, secret.ResourceVersion)
	}
	return true, available, nil
}

func (r *AgentGroupReconciler) enqueueDependents(kind string) handler.MapFunc {
	return func(ctx context.Context, object client.Object) []ctrl.Request {
		pending := []dependency{{kind, object.GetName()}}
		seen := make(map[dependency]bool)
		groups := make(map[client.ObjectKey]bool)
		for len(pending) > 0 {
			ref := pending[0]
			pending = pending[1:]
			if seen[ref] {
				continue
			}
			seen[ref] = true
			for _, list := range []client.ObjectList{&api.AgentGroupList{}, &api.RemoteConfigList{}, &api.AgentPackageList{}, &api.EndpointList{}} {
				if err := r.List(ctx, list, client.InNamespace(object.GetNamespace()), client.MatchingFields{dependencyIndex: ref.kind + "/" + ref.name}); err != nil {
					// The periodic group repair also covers notifications whose index lookup failed.
					ctrl.LoggerFrom(ctx).Error(err, "Failed to list dependent resources", "kind", ref.kind)
					continue
				}
				if err := meta.EachListItem(list, func(item runtime.Object) error {
					switch item := item.(type) {
					case *api.AgentGroup:
						groups[client.ObjectKeyFromObject(item)] = true
					case *api.RemoteConfig:
						pending = append(pending, dependency{remoteConfigKind, item.Name})
					case *api.AgentPackage:
						pending = append(pending, dependency{packageKind, item.Name})
					case *api.Endpoint:
						pending = append(pending, dependency{endpointKind, item.Name})
					}
					return nil
				}); err != nil {
					ctrl.LoggerFrom(ctx).Error(err, "Failed to read dependent resource list")
				}
			}
		}
		requests := make([]ctrl.Request, 0, len(groups))
		for key := range groups {
			requests = append(requests, ctrl.Request{NamespacedName: key})
		}
		return requests
	}
}

// Ignore status-only writes but observe deletion while finalizers still hold the object.
var configurationChanged = predicate.Funcs{
	UpdateFunc: func(e event.UpdateEvent) bool {
		return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration() ||
			e.ObjectOld.GetUID() != e.ObjectNew.GetUID() ||
			!e.ObjectOld.GetDeletionTimestamp().Equal(e.ObjectNew.GetDeletionTimestamp())
	},
}

func (r *AgentGroupReconciler) watchDependencies(mgr ctrl.Manager, controller *builder.Builder) error {
	for _, object := range []client.Object{&api.AgentGroup{}, &api.RemoteConfig{}, &api.AgentPackage{}, &api.Endpoint{}} {
		if err := mgr.GetFieldIndexer().IndexField(context.Background(), object, dependencyIndex, dependencyKeys); err != nil {
			return fmt.Errorf("index dependencies: %w", err)
		}
	}
	for _, object := range []client.Object{&api.RemoteConfigSchema{}, (dependency{kind: secretKind}).object()} {
		if _, err := mgr.GetCache().GetInformer(context.Background(), object); err != nil {
			return fmt.Errorf("initialize dependency cache: %w", err)
		}
	}
	controller.
		Watches(&api.RemoteConfig{}, handler.EnqueueRequestsFromMapFunc(r.enqueueDependents(remoteConfigKind)), builder.WithPredicates(configurationChanged)).
		Watches(&api.AgentPackage{}, handler.EnqueueRequestsFromMapFunc(r.enqueueDependents(packageKind)), builder.WithPredicates(configurationChanged)).
		Watches(&api.RemoteConfigSchema{}, handler.EnqueueRequestsFromMapFunc(r.enqueueDependents(schemaKind)), builder.WithPredicates(configurationChanged)).
		Watches(&api.Endpoint{}, handler.EnqueueRequestsFromMapFunc(r.enqueueDependents(endpointKind)), builder.WithPredicates(configurationChanged)).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(r.enqueueDependents(secretKind)), builder.OnlyMetadata, builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}))
	return nil
}
