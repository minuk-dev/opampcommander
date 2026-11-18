//go:build integration

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	api "github.com/minuk-dev/opampcommander/operator/api/v1alpha1"
)

var _ = Describe("Configuration API contract", func() {
	It("accepts every documented sample with server-side dry-run", func() {
		paths, err := filepath.Glob(filepath.Join("..", "..", "config", "samples", "opampcommander_v1alpha1_*.yaml"))
		Expect(err).NotTo(HaveOccurred())
		Expect(paths).To(HaveLen(5))
		for _, path := range paths {
			data, err := os.ReadFile(path)
			Expect(err).NotTo(HaveOccurred())
			object := &unstructured.Unstructured{}
			Expect(yaml.Unmarshal(data, &object.Object)).To(Succeed())
			object.SetNamespace(metav1.NamespaceDefault)
			Expect(applyObject(object, "sample-validation", client.DryRunAll)).To(Succeed(), path)
		}
	})

	It("validates all five resource shapes and immutable artifact references", func() {
		artifact := api.ArtifactReference{URL: "https://artifacts.example.test/collector.tar.gz", Digest: "sha256:" + strings.Repeat("a", 64), SizeBytes: 1024}
		objects := []client.Object{
			&api.AgentGroup{ObjectMeta: metav1.ObjectMeta{Name: "contract-group", Namespace: metav1.NamespaceDefault}, Spec: api.AgentGroupSpec{RemoteConfigs: []api.RemoteConfigReference{{Name: "public", Filename: ""}}, ConnectionSettingsSecretRef: &api.SecretKeyReference{Name: "connections", Key: "settings.json"}}},
			&api.RemoteConfig{ObjectMeta: metav1.ObjectMeta{Name: "contract-config", Namespace: metav1.NamespaceDefault}, Spec: api.RemoteConfigSpec{ValueFrom: &api.SecretKeyReference{Name: "config", Key: "collector.yaml"}, SchemaRefs: []api.LocalReference{{Name: "contrib"}}}},
			&api.AgentPackage{ObjectMeta: metav1.ObjectMeta{Name: "contract-package", Namespace: metav1.NamespaceDefault}, Spec: api.AgentPackageSpec{PackageType: "otelcol", Version: "0.157.0", Artifact: artifact, Headers: []api.HeaderReference{{Name: "Authorization", ValueFrom: api.SecretKeyReference{Name: "download", Key: "token"}}}}},
			&api.RemoteConfigSchema{ObjectMeta: metav1.ObjectMeta{Name: "contract-schema", Namespace: metav1.NamespaceDefault}, Spec: api.RemoteConfigSchemaSpec{Binary: "otelcol-contrib", Version: "0.157.0", Catalog: artifact}},
			&api.Endpoint{ObjectMeta: metav1.ObjectMeta{Name: "contract-endpoint", Namespace: metav1.NamespaceDefault}, Spec: api.EndpointSpec{URL: "https://backend.example.test/v1/traces", Protocol: "otlphttp", Signals: []api.TelemetrySignal{"traces"}}},
		}
		for _, object := range objects {
			Expect(k8sClient.Create(ctx, object)).To(Succeed())
			DeferCleanup(func() { Expect(k8sClient.Delete(ctx, object)).To(Succeed()) })
		}
		pkg := objects[2].(*api.AgentPackage)
		pkg.Spec.Artifact.Digest = "sha256:" + strings.Repeat("b", 64)
		Expect(apierrors.IsInvalid(k8sClient.Update(ctx, pkg))).To(BeTrue())
	})

	DescribeTable("rejects invalid configuration at the API boundary", func(kind string, spec map[string]any) {
		object := rawResource(kind, "invalid-contract", spec)
		Expect(apierrors.IsInvalid(k8sClient.Create(ctx, object))).To(BeTrue())
	},
		Entry("missing selector", "AgentGroup", map[string]any{}),
		Entry("empty reference name", "AgentGroup", map[string]any{"selector": map[string]any{}, "remoteConfigs": []any{map[string]any{"name": "", "filename": ""}}}),
		Entry("missing filename", "AgentGroup", map[string]any{"selector": map[string]any{}, "remoteConfigs": []any{map[string]any{"name": "config"}}}),
		Entry("ambiguous config source", "RemoteConfig", map[string]any{"value": "service: {}", "valueFrom": map[string]any{"name": "config", "key": "yaml"}}),
		Entry("missing config source", "RemoteConfig", map[string]any{}),
		Entry("invalid secret key", "RemoteConfig", map[string]any{"valueFrom": map[string]any{"name": "config", "key": "../private/key"}}),
		Entry("UTF-8 byte budget", "RemoteConfig", map[string]any{"value": strings.Repeat("한", 90000)}),
		Entry("invalid digest", "AgentPackage", map[string]any{"packageType": "otelcol", "version": "1", "artifact": map[string]any{"url": "https://example.test/artifact", "digest": "sha256:bad", "sizeBytes": 10}}),
		Entry("catalog too large", "RemoteConfigSchema", map[string]any{"binary": "contrib", "version": "1", "catalog": map[string]any{"url": "https://example.test/catalog", "digest": "sha256:" + strings.Repeat("a", 64), "sizeBytes": 16777217}}),
		Entry("credential URL", "Endpoint", map[string]any{"url": "https://user:password@example.test/", "protocol": "otlphttp", "signals": []any{"traces"}}),
		Entry("unknown signal", "Endpoint", map[string]any{"url": "https://example.test/", "protocol": "otlphttp", "signals": []any{"unknown"}}),
	)

	It("preserves opaque config, defaults content type, and isolates status writes", func() {
		text := "vendor:\n  unknown_future_field: true\n"
		resource := &api.RemoteConfig{ObjectMeta: metav1.ObjectMeta{Name: "contract-status", Namespace: metav1.NamespaceDefault}, Spec: api.RemoteConfigSpec{Value: text}, Status: api.RemoteConfigStatus{Conditions: []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, Reason: "Injected", Message: "ignored", LastTransitionTime: metav1.Now()}}}}
		Expect(k8sClient.Create(ctx, resource)).To(Succeed())
		DeferCleanup(func() { Expect(k8sClient.Delete(ctx, resource)).To(Succeed()) })
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(resource), resource)).To(Succeed())
		Expect(resource.Spec.Value).To(Equal(text))
		Expect(resource.Spec.ContentType).To(Equal("application/yaml"))
		Expect(resource.Status.Conditions).To(BeEmpty())
		generation := resource.Generation
		resource.Status.Conditions = []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, Reason: "Pending", Message: "not delivered", ObservedGeneration: generation, LastTransitionTime: metav1.Now()}}
		resource.Spec.Value = "attempted spec change through status"
		Expect(k8sClient.Status().Update(ctx, resource)).To(Succeed())
		Expect(resource.Generation).To(Equal(generation))
		Expect(resource.Spec.Value).To(Equal(text))
		stale := resource.DeepCopy()
		resource.Spec.Value = "vendor: {changed: true}"
		resource.Status.Conditions = nil
		Expect(k8sClient.Update(ctx, resource)).To(Succeed())
		Expect(resource.Generation).To(Equal(generation + 1))
		Expect(resource.Status.Conditions).To(HaveLen(1))
		stale.Spec.Value = "stale writer"
		Expect(apierrors.IsConflict(k8sClient.Update(ctx, stale))).To(BeTrue())
	})

	It("enforces server-side apply ownership and supports list/watch", func() {
		draft := rawResource("RemoteConfig", "contract-dry-run", map[string]any{"value": "vendor: {field: true}"})
		Expect(applyObject(draft, "preview", client.DryRunAll)).To(Succeed())
		Expect(apierrors.IsNotFound(k8sClient.Get(ctx, client.ObjectKeyFromObject(draft), &api.RemoteConfig{}))).To(BeTrue())
		watcher, err := client.NewWithWatch(cfg, client.Options{Scheme: k8sClient.Scheme()})
		Expect(err).NotTo(HaveOccurred())
		list := &api.RemoteConfigList{}
		Expect(watcher.List(ctx, list, client.InNamespace(metav1.NamespaceDefault))).To(Succeed())
		watchCtx, cancelWatch := context.WithTimeout(ctx, 10*time.Second)
		defer cancelWatch()
		stream, err := watcher.Watch(watchCtx, &api.RemoteConfigList{}, client.InNamespace(metav1.NamespaceDefault), &client.ListOptions{Raw: &metav1.ListOptions{ResourceVersion: list.ResourceVersion}})
		Expect(err).NotTo(HaveOccurred())
		defer stream.Stop()
		object := rawResource("RemoteConfig", "contract-apply", map[string]any{"value": "vendor: {field: true}"})
		Expect(applyObject(object, "gitops-a")).To(Succeed())
		DeferCleanup(func() { Expect(k8sClient.Delete(ctx, object)).To(Succeed()) })
		var event watch.Event
		Eventually(stream.ResultChan()).Should(Receive(&event))
		Expect(event.Type).To(Equal(watch.Added))
		Expect(event.Object.(*api.RemoteConfig).Name).To(Equal("contract-apply"))
		conflict := rawResource("RemoteConfig", "contract-apply", map[string]any{"value": "vendor: {field: false}"})
		Expect(apierrors.IsConflict(applyObject(conflict, "gitops-b"))).To(BeTrue())
	})

	It("measures the representative catalog with API storage and managedFields overhead", func() {
		data, err := os.ReadFile(filepath.Join("..", "..", "..", "configs", "apiserver", "initial", "remoteconfigschema", "otelcol-contrib-0.157.0.yaml"))
		Expect(err).NotTo(HaveOccurred())
		compact, err := yaml.YAMLToJSON(data)
		Expect(err).NotTo(HaveOccurred())
		var legacy map[string]any
		Expect(json.Unmarshal(compact, &legacy)).To(Succeed())
		catalog := legacy["spec"].(map[string]any)["components"]
		catalogJSON, err := json.Marshal(catalog)
		Expect(err).NotTo(HaveOccurred())
		// Probe the actual custom-resource representation without committing an inline catalog API.
		crd := &unstructured.Unstructured{}
		Expect(yaml.Unmarshal([]byte(catalogProbeCRD), &crd.Object)).To(Succeed())
		Expect(k8sClient.Create(ctx, crd)).To(Succeed())
		DeferCleanup(func() { Expect(k8sClient.Delete(ctx, crd)).To(Succeed()) })
		probe := rawResource("CatalogProbe", "catalog-size", map[string]any{"catalog": catalog})
		probe.SetAPIVersion("test.opampcommander.io/v1")
		Eventually(func() error { return applyObject(probe, "catalog-measure-a", client.DryRunAll) }, 10*time.Second, 100*time.Millisecond).Should(Succeed())
		dryRun, err := json.Marshal(probe.Object)
		Expect(err).NotTo(HaveOccurred())
		managed, err := json.Marshal(probe.GetManagedFields())
		Expect(err).NotTo(HaveOccurred())
		attempt := rawResource("CatalogProbe", "catalog-size", map[string]any{"catalog": catalog})
		attempt.SetAPIVersion("test.opampcommander.io/v1")
		err = applyObject(attempt, "catalog-measure-a")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("request is too large"))
		_, _ = fmt.Fprintf(GinkgoWriter, "catalog-size yaml=%d legacy-json=%d catalog-json=%d server-dry-run=%d managedFields=%d\n", len(data), len(compact), len(catalogJSON), len(dryRun), len(managed))
		smallCatalog := map[string]any{"exporters": map[string]any{"debug": map[string]any{"type": "debug"}}}
		small := rawResource("CatalogProbe", "small-catalog", map[string]any{"catalog": smallCatalog})
		small.SetAPIVersion("test.opampcommander.io/v1")
		Expect(applyObject(small, "catalog-measure-a")).To(Succeed())
		first, err := json.Marshal(small.Object)
		Expect(err).NotTo(HaveOccurred())
		second := rawResource("CatalogProbe", "small-catalog", map[string]any{"catalog": smallCatalog})
		second.SetAPIVersion("test.opampcommander.io/v1")
		Expect(applyObject(second, "catalog-measure-b")).To(Succeed())
		stored, err := json.Marshal(second.Object)
		Expect(err).NotTo(HaveOccurred())
		Expect(second.GetManagedFields()).To(HaveLen(2))
		Expect(len(stored)).To(BeNumerically(">", len(first)))
		inline := rawResource("RemoteConfig", "large-catalog", map[string]any{"value": string(catalogJSON)})
		Expect(apierrors.IsInvalid(k8sClient.Create(ctx, inline))).To(BeTrue())
	})
})

func rawResource(kind, name string, spec map[string]any) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{"apiVersion": "opampcommander.io/v1alpha1", "kind": kind, "metadata": map[string]any{"name": name, "namespace": metav1.NamespaceDefault}, "spec": spec}}
}

func applyObject(object *unstructured.Unstructured, owner string, options ...client.PatchOption) error {
	data, err := json.Marshal(object.Object)
	if err != nil {
		return err
	}
	return k8sClient.Patch(ctx, object, client.RawPatch(types.ApplyPatchType, data), append(options, client.FieldOwner(owner))...)
}

// The storage probe is installed only in envtest, never in the distributed bundle.
const catalogProbeCRD = `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: catalogprobes.test.opampcommander.io
spec:
  group: test.opampcommander.io
  scope: Namespaced
  names:
    plural: catalogprobes
    singular: catalogprobe
    kind: CatalogProbe
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                catalog:
                  type: object
                  x-kubernetes-preserve-unknown-fields: true
`
