//go:build integration

package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	opampcommanderv1alpha1 "github.com/minuk-dev/opampcommander/operator/api/v1alpha1"
)

var _ = Describe("AgentGroup Controller", func() {
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
		agentgroup := &opampcommanderv1alpha1.AgentGroup{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind AgentGroup")
			err := k8sClient.Get(ctx, typeNamespacedName, agentgroup)
			if err != nil && errors.IsNotFound(err) {
				resource := &opampcommanderv1alpha1.AgentGroup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: resourceNamespace,
					},
					Spec: opampcommanderv1alpha1.AgentGroupSpec{Selector: opampcommanderv1alpha1.AgentSelector{}},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {

			resource := &opampcommanderv1alpha1.AgentGroup{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance AgentGroup")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &AgentGroupReconciler{
				Client:       k8sClient,
				Scheme:       k8sClient.Scheme(),
				Installation: testInstallation,
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			current := &opampcommanderv1alpha1.AgentGroup{}
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
			current.Spec.Priority++
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

var _ = Describe("AgentGroup schema", func() {
	It("preserves an empty OpAMP key and rejects duplicate empty keys", func() {
		resource := &opampcommanderv1alpha1.AgentGroup{
			ObjectMeta: metav1.ObjectMeta{Name: "empty-filename", Namespace: metav1.NamespaceDefault},
			Spec: opampcommanderv1alpha1.AgentGroupSpec{
				RemoteConfigs: []opampcommanderv1alpha1.RemoteConfigReference{{Name: "config", Filename: ""}},
			},
		}
		Expect(k8sClient.Create(ctx, resource)).To(Succeed())
		DeferCleanup(func() { Expect(k8sClient.Delete(ctx, resource)).To(Succeed()) })
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: resource.Name, Namespace: resource.Namespace}, resource)).To(Succeed())
		Expect(resource.Spec.RemoteConfigs).To(Equal([]opampcommanderv1alpha1.RemoteConfigReference{{Name: "config", Filename: ""}}))
		resource.Spec.RemoteConfigs = append(resource.Spec.RemoteConfigs, opampcommanderv1alpha1.RemoteConfigReference{Name: "other", Filename: ""})
		Expect(errors.IsInvalid(k8sClient.Update(ctx, resource))).To(BeTrue())
	})

	It("rejects an invalid spec at the API boundary", func() {
		resource := &opampcommanderv1alpha1.AgentGroup{
			ObjectMeta: metav1.ObjectMeta{Name: "invalid-agentgroup", Namespace: metav1.NamespaceDefault},
			Spec:       opampcommanderv1alpha1.AgentGroupSpec{RemoteConfigs: []opampcommanderv1alpha1.RemoteConfigReference{{Filename: "collector.yaml"}}},
		}
		Expect(errors.IsInvalid(k8sClient.Create(ctx, resource))).To(BeTrue())
	})
})
