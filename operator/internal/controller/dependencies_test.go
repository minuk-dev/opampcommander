package controller

import (
	"context"
	"errors"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/event"

	api "github.com/minuk-dev/opampcommander/operator/api/v1alpha1"
)

func TestAgentGroupSecretReferencesUseLiveState(t *testing.T) {
	t.Parallel()
	const privateValue = "private-value"
	for _, tc := range []struct {
		name string
		deny bool
		want metav1.ConditionStatus
	}{
		{name: "cached Secret has been deleted", want: metav1.ConditionFalse},
		{name: "live Secret cannot be read", deny: true, want: metav1.ConditionUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			reader, installation := installationFixture(t)
			namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "tenant", Labels: map[string]string{InstallationLabel: string(installation.UID)}}}
			group := &api.AgentGroup{ObjectMeta: metav1.ObjectMeta{Name: "group", Namespace: namespace.Name}, Spec: api.AgentGroupSpec{
				ConnectionSettingsSecretRef: &api.SecretKeyReference{Name: "settings", Key: "bundle"},
			}}
			foreign := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "settings", Namespace: "other"},
				Data:       map[string][]byte{"bundle": []byte(privateValue)},
			}
			for _, object := range []client.Object{namespace, group, foreign} {
				if err := reader.Create(t.Context(), object); err != nil {
					t.Fatal(err)
				}
			}
			clientWithStaleMetadata := interceptor.NewClient(reader.(client.WithWatch), interceptor.Funcs{Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, object client.Object, opts ...client.GetOption) error {
				if object.GetObjectKind().GroupVersionKind() == corev1.SchemeGroupVersion.WithKind(secretKind) {
					object.SetName(key.Name)
					object.SetNamespace(key.Namespace)
					return nil
				}
				if _, ok := object.(*corev1.Secret); ok && tc.deny {
					return errors.New(privateValue)
				}
				return c.Get(ctx, key, object, opts...)
			}})
			installation.reader = clientWithStaleMetadata
			r := &AgentGroupReconciler{Client: clientWithStaleMetadata, Installation: installation}
			key := client.ObjectKeyFromObject(group)
			_, err := r.Reconcile(t.Context(), ctrl.Request{NamespacedName: key})
			if (err != nil) != tc.deny {
				t.Fatalf("Reconcile error = %v, want error = %v", err, tc.deny)
			}
			if err != nil && strings.Contains(err.Error(), privateValue) {
				t.Fatal("Reconcile returned the raw Secret read error")
			}
			if err := reader.Get(t.Context(), key, group); err != nil {
				t.Fatal(err)
			}
			condition := meta.FindStatusCondition(group.Status.Conditions, "ReferencesFound")
			if condition == nil || condition.Status != tc.want {
				t.Fatalf("ReferencesFound = %v, want %s", condition, tc.want)
			}
			keys := meta.FindStatusCondition(group.Status.Conditions, "SecretKeysAvailable")
			if tc.deny && (keys == nil || keys.Status != metav1.ConditionUnknown || strings.Contains(keys.Message, privateValue)) {
				t.Fatalf("SecretKeysAvailable = %v, want a redacted Unknown condition", keys)
			}
			if !tc.deny && keys != nil {
				t.Fatalf("SecretKeysAvailable = %v for a missing Secret", keys)
			}
		})
	}
}

func TestEnqueueDependents(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := api.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	objects := make([]client.Object, 0, 6)
	objects = append(objects,
		&api.AgentGroup{ObjectMeta: metav1.ObjectMeta{Name: "group", Namespace: "one"}, Spec: api.AgentGroupSpec{
			RemoteConfigs: []api.RemoteConfigReference{{Name: "config"}, {Name: "config", Filename: "second"}},
			PackageRefs:   []api.LocalReference{{Name: "package"}}, ConnectionSettingsSecretRef: &api.SecretKeyReference{Name: "connections"},
		}},
		&api.RemoteConfig{ObjectMeta: metav1.ObjectMeta{Name: "config", Namespace: "one"}, Spec: api.RemoteConfigSpec{
			ValueFrom: &api.SecretKeyReference{Name: "config-secret"}, SchemaRefs: []api.LocalReference{{Name: "schema"}}, EndpointRefs: []api.LocalReference{{Name: "endpoint"}},
		}},
		&api.AgentPackage{ObjectMeta: metav1.ObjectMeta{Name: "package", Namespace: "one"}, Spec: api.AgentPackageSpec{
			Headers: []api.HeaderReference{{ValueFrom: api.SecretKeyReference{Name: "package-secret"}}}, SignatureSecretRef: &api.SecretKeyReference{Name: "signature"},
		}},
		&api.Endpoint{ObjectMeta: metav1.ObjectMeta{Name: "endpoint", Namespace: "one"}, Spec: api.EndpointSpec{
			Tenants: []api.EndpointTenant{{Headers: []api.HeaderReference{{ValueFrom: api.SecretKeyReference{Name: "endpoint-secret"}}}}},
		}},
	)
	other := objects[0].DeepCopyObject().(*api.AgentGroup)
	other.Namespace = "two"
	objects = append(objects, other, &api.AgentGroup{ObjectMeta: metav1.ObjectMeta{Name: "unrelated", Namespace: "one"}})
	reader := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...)
	for _, object := range []client.Object{&api.AgentGroup{}, &api.RemoteConfig{}, &api.AgentPackage{}, &api.Endpoint{}} {
		reader.WithIndex(object, dependencyIndex, dependencyKeys)
	}
	r := &AgentGroupReconciler{Client: reader.Build(), Scheme: scheme}
	for _, ref := range []dependency{
		{"RemoteConfig", "config"}, {"RemoteConfigSchema", "schema"}, {"AgentPackage", "package"}, {"Endpoint", "endpoint"},
		{"Secret", "connections"}, {"Secret", "config-secret"}, {"Secret", "package-secret"}, {"Secret", "signature"}, {"Secret", "endpoint-secret"},
	} {
		t.Run(ref.kind+"/"+ref.name, func(t *testing.T) {
			object := ref.object()
			object.SetName(ref.name)
			object.SetNamespace("one")
			requests := r.enqueueDependents(ref.kind)(context.Background(), object)
			if len(requests) != 1 || requests[0].NamespacedName != (types.NamespacedName{Namespace: "one", Name: "group"}) {
				t.Fatalf("expected only one affected group, got %v", requests)
			}
		})
	}
}

func TestConfigurationChanged(t *testing.T) {
	old := &api.AgentGroup{ObjectMeta: metav1.ObjectMeta{Generation: 1, ResourceVersion: "1", UID: "original"}}
	for _, tc := range []struct {
		name   string
		change func(*api.AgentGroup)
		want   bool
	}{
		{"no-op", func(*api.AgentGroup) {}, false},
		{"status", func(g *api.AgentGroup) {
			g.ResourceVersion = "2"
			g.Status.Conditions = []metav1.Condition{{Type: "Ready"}}
		}, false},
		{"spec", func(g *api.AgentGroup) { g.Generation++ }, true},
		{"deleting", func(g *api.AgentGroup) { now := metav1.Now(); g.DeletionTimestamp = &now }, true},
		{"recreated", func(g *api.AgentGroup) { g.UID = "replacement" }, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			updated := old.DeepCopy()
			tc.change(updated)
			if got := configurationChanged.Update(event.UpdateEvent{ObjectOld: old, ObjectNew: updated}); got != tc.want {
				t.Fatalf("enqueue = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSecretDependencyUsesMetadata(t *testing.T) {
	object := (dependency{kind: "Secret"}).object()
	if _, ok := object.(*metav1.PartialObjectMetadata); !ok || object.GetObjectKind().GroupVersionKind() != corev1.SchemeGroupVersion.WithKind("Secret") {
		t.Fatal("Secret dependency must use metadata, not cache private values")
	}
}
