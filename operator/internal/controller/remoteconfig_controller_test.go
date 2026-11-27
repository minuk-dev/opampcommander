//go:build integration

package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	opampcommanderv1alpha1 "github.com/minuk-dev/opampcommander/operator/api/v1alpha1"
)

var _ = Describe("RemoteConfig Controller", func() {
	Context("When reconciling a resource", func() {
		const (
			resourceName      = "test-resource"
			resourceNamespace = metav1.NamespaceDefault
		)

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: resourceNamespace,
		}
		remoteconfig := &opampcommanderv1alpha1.RemoteConfig{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind RemoteConfig")
			err := k8sClient.Get(ctx, typeNamespacedName, remoteconfig)
			if err != nil && errors.IsNotFound(err) {
				resource := &opampcommanderv1alpha1.RemoteConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: resourceNamespace,
					},
					Spec: opampcommanderv1alpha1.RemoteConfigSpec{Value: "service: {}"},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {

			resource := &opampcommanderv1alpha1.RemoteConfig{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance RemoteConfig")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("rejects a stale status patch after same-name recreation", func() {
			reader, err := client.NewWithWatch(cfg, client.Options{Scheme: k8sClient.Scheme()})
			Expect(err).NotTo(HaveOccurred())
			var replacementUID types.UID
			reconciler := &RemoteConfigReconciler{
				Client: interceptor.NewClient(reader, interceptor.Funcs{
					SubResourcePatch: func(ctx context.Context, c client.Client, subresource string, old client.Object,
						patch client.Patch, opts ...client.SubResourcePatchOption) error {
						Expect(c.Delete(ctx, old)).To(Succeed())
						replacement := &opampcommanderv1alpha1.RemoteConfig{
							ObjectMeta: metav1.ObjectMeta{Name: old.GetName(), Namespace: old.GetNamespace()},
							Spec:       opampcommanderv1alpha1.RemoteConfigSpec{Value: "service: {telemetry: {}}"},
						}
						Expect(c.Create(ctx, replacement)).To(Succeed())
						replacementUID = replacement.UID
						Expect(replacementUID).NotTo(Equal(old.GetUID()))
						return c.SubResource(subresource).Patch(ctx, old, patch, opts...)
					},
				}),
				Installation: testInstallation,
			}

			_, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(errors.IsConflict(err)).To(BeTrue())
			current := &opampcommanderv1alpha1.RemoteConfig{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, current)).To(Succeed())
			Expect(current.UID).To(Equal(replacementUID))
			Expect(current.Status.Conditions).To(BeEmpty())
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &RemoteConfigReconciler{
				Client:       k8sClient,
				Scheme:       k8sClient.Scheme(),
				Installation: testInstallation,
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			current := &opampcommanderv1alpha1.RemoteConfig{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, current)).To(Succeed())
			condition := meta.FindStatusCondition(current.Status.Conditions, "Ready")
			Expect(condition).NotTo(BeNil())
			Expect(condition.Status).To(Equal(metav1.ConditionUnknown))
			Expect(condition.Reason).To(Equal("NotImplemented"))
			Expect(condition.ObservedGeneration).To(Equal(current.Generation))
			version := current.ResourceVersion
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sClient.Get(ctx, typeNamespacedName, current)).To(Succeed())
			Expect(current.ResourceVersion).To(Equal(version), "no-op reconcile must not write status")
			current.Spec.Value = "service: {telemetry: {}}"
			Expect(k8sClient.Update(ctx, current)).To(Succeed())
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sClient.Get(ctx, typeNamespacedName, current)).To(Succeed())
			condition = meta.FindStatusCondition(current.Status.Conditions, "Ready")
			Expect(condition.ObservedGeneration).To(Equal(current.Generation))
			Expect(condition.Status).To(Equal(metav1.ConditionUnknown))

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: "absent", Namespace: resourceNamespace}})
			Expect(err).NotTo(HaveOccurred())

		})
	})
})

var _ = Describe("RemoteConfig schema", func() {
	It("rejects an invalid spec at the API boundary", func() {
		resource := &opampcommanderv1alpha1.RemoteConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "invalid-remoteconfig", Namespace: metav1.NamespaceDefault},
			Spec:       opampcommanderv1alpha1.RemoteConfigSpec{Value: ""},
		}
		Expect(errors.IsInvalid(k8sClient.Create(ctx, resource))).To(BeTrue())
	})
})
