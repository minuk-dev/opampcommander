package controller

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	api "github.com/minuk-dev/opampcommander/operator/api/v1alpha1"
)

func installationFixture(t *testing.T) (client.Client, *Installation) {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := api.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	anchor := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "opamp-installation", Namespace: "system", UID: "installation-one"}}
	reader := fake.NewClientBuilder().WithScheme(scheme).WithObjects(anchor).
		WithStatusSubresource(&api.AgentGroup{}, &api.RemoteConfig{}).Build()
	installation, err := LoadInstallation(t.Context(), reader, client.ObjectKeyFromObject(anchor))
	if err != nil {
		t.Fatal(err)
	}
	return reader, installation
}

func TestInstallationIdentity(t *testing.T) {
	t.Parallel()
	for _, change := range []string{"data update", "missing", "replacement", "deleting"} {
		t.Run(change, func(t *testing.T) {
			t.Parallel()
			reader, installation := installationFixture(t)
			anchor := &corev1.ConfigMap{}
			if err := reader.Get(t.Context(), installation.key, anchor); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "data update":
				anchor.Data = map[string]string{"note": "same identity"}
				if err := reader.Update(t.Context(), anchor); err != nil {
					t.Fatal(err)
				}
			case "missing", "replacement":
				if err := reader.Delete(t.Context(), anchor); err != nil {
					t.Fatal(err)
				}
				if change == "replacement" {
					anchor.ResourceVersion = ""
					anchor.UID = "installation-two"
					if err := reader.Create(t.Context(), anchor); err != nil {
						t.Fatal(err)
					}
				}
			case "deleting":
				anchor.Finalizers = []string{"test/finalizer"}
				if err := reader.Update(t.Context(), anchor); err != nil {
					t.Fatal(err)
				}
				if err := reader.Delete(t.Context(), anchor); err != nil {
					t.Fatal(err)
				}
			}
			if err := installation.Check(t.Context()); (err == nil) != (change == "data update") {
				t.Fatalf("identity check: %v", err)
			}
			if change == "missing" || change == "deleting" {
				if _, err := LoadInstallation(t.Context(), reader, installation.key); err == nil {
					t.Fatal("startup accepted a missing/deleting anchor")
				}
			}
		})
	}
}

func TestInstallationNamespaceBoundary(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, assignment string
		deleting, want   bool
	}{
		{name: "owned", assignment: "installation-one", want: true},
		{name: "unassigned"},
		{name: "other installation", assignment: "installation-two"},
		{name: "terminating", assignment: "installation-one", deleting: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			reader, installation := installationFixture(t)
			namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "tenant", Labels: map[string]string{InstallationLabel: tc.assignment}}}
			if tc.deleting {
				now := metav1.Now()
				namespace.DeletionTimestamp = &now
				namespace.Finalizers = []string{"test/finalizer"}
			}
			// Seed terminating metadata directly; real namespaces are deleted by the namespace controller.
			reader = fake.NewClientBuilder().WithScheme(reader.Scheme()).WithObjects(namespace,
				&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: installation.key.Name, Namespace: installation.key.Namespace, UID: installation.UID}}).
				WithStatusSubresource(&api.AgentGroup{}, &api.RemoteConfig{}).Build()
			installation.reader = reader
			group := &api.AgentGroup{ObjectMeta: metav1.ObjectMeta{Name: "resource", Namespace: namespace.Name}}
			config := &api.RemoteConfig{ObjectMeta: group.ObjectMeta, Spec: api.RemoteConfigSpec{Value: "service: {}"}}
			for _, object := range []client.Object{group, config} {
				if err := reader.Create(t.Context(), object); err != nil {
					t.Fatal(err)
				}
			}
			request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(group)}
			for _, reconcile := range []func(context.Context, ctrl.Request) (ctrl.Result, error){
				(&AgentGroupReconciler{Client: reader, Installation: installation}).Reconcile,
				(&RemoteConfigReconciler{Client: reader, Installation: installation}).Reconcile,
			} {
				if _, err := reconcile(t.Context(), request); err != nil {
					t.Fatal(err)
				}
			}
			if err := reader.Get(t.Context(), request.NamespacedName, group); err != nil {
				t.Fatal(err)
			}
			if err := reader.Get(t.Context(), request.NamespacedName, config); err != nil {
				t.Fatal(err)
			}
			if (len(group.Status.Conditions) > 0) != tc.want || (len(config.Status.Conditions) > 0) != tc.want {
				t.Fatalf("unexpected status writes: group=%v config=%v", group.Status.Conditions, config.Status.Conditions)
			}
			if owned, err := installation.ownedNamespace(t.Context(), "absent"); err != nil || owned != nil {
				t.Fatalf("missing namespace accepted: owned=%v err=%v", owned, err)
			}
		})
	}
}
