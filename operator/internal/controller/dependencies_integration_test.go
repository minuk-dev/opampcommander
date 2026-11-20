//go:build integration

package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	controllerconfig "sigs.k8s.io/controller-runtime/pkg/config"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	api "github.com/minuk-dev/opampcommander/operator/api/v1alpha1"
)

var _ = Describe("Dependency watches", func() {
	It("recovers missing references through native writes and a fresh manager cache", func() {
		namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{GenerateName: "dependency-watch-"}}
		Expect(k8sClient.Create(ctx, namespace)).To(Succeed())
		DeferCleanup(func() { Expect(k8sClient.Delete(ctx, namespace)).To(Succeed()) })
		group := &api.AgentGroup{ObjectMeta: metav1.ObjectMeta{Name: "group", Namespace: namespace.Name}, Spec: api.AgentGroupSpec{
			RemoteConfigs: []api.RemoteConfigReference{{Name: "config", Filename: ""}},
		}}
		Expect(k8sClient.Create(ctx, group)).To(Succeed())
		start := func() func() {
			mgr, err := ctrl.NewManager(cfg, ctrl.Options{
				Scheme: k8sClient.Scheme(), Metrics: metricsserver.Options{BindAddress: "0"}, HealthProbeBindAddress: "0",
				Cache:      cache.Options{ReaderFailOnMissingInformer: true, DefaultNamespaces: map[string]cache.Config{namespace.Name: {}}},
				Controller: controllerconfig.Controller{SkipNameValidation: new(true)},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect((&AgentGroupReconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme(), Installation: testInstallation}).SetupWithManager(mgr)).To(Succeed())
			runCtx, stop := context.WithCancel(ctx)
			done := make(chan error, 1)
			go func() { done <- mgr.Start(runCtx) }()
			stopped := false
			shutdown := func() {
				if !stopped {
					stop()
					Eventually(done, 10*time.Second).Should(Receive(Succeed()))
					stopped = true
				}
			}
			DeferCleanup(shutdown)
			syncCtx, cancelSync := context.WithTimeout(ctx, 10*time.Second)
			defer cancelSync()
			Expect(mgr.GetCache().WaitForCacheSync(syncCtx)).To(BeTrue())
			return shutdown
		}
		expectReferences := func(status metav1.ConditionStatus) {
			Eventually(func(g Gomega) {
				current := &api.AgentGroup{}
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(group), current)).To(Succeed())
				condition := meta.FindStatusCondition(current.Status.Conditions, "ReferencesFound")
				g.Expect(condition).NotTo(BeNil())
				g.Expect(condition.Status).To(Equal(status))
				g.Expect(meta.FindStatusCondition(current.Status.Conditions, "Ready").Status).To(Equal(metav1.ConditionUnknown))
			}, 10*time.Second, 50*time.Millisecond).Should(Succeed())
		}
		stop := start()
		Consistently(func(g Gomega) {
			current := &api.AgentGroup{}
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(group), current)).To(Succeed())
			g.Expect(current.Status.Conditions).To(BeEmpty())
		}, 300*time.Millisecond, 50*time.Millisecond).Should(Succeed())
		namespace.Labels = map[string]string{InstallationLabel: string(testInstallation.UID)}
		Expect(k8sClient.Update(ctx, namespace)).To(Succeed())
		expectReferences(metav1.ConditionFalse)
		config := &api.RemoteConfig{ObjectMeta: metav1.ObjectMeta{Name: "config", Namespace: namespace.Name}, Spec: api.RemoteConfigSpec{Value: "service: {}"}}
		Expect(k8sClient.Create(ctx, config)).To(Succeed())
		expectReferences(metav1.ConditionTrue)
		config.Spec.Value = ""
		config.Spec.ValueFrom = &api.SecretKeyReference{Name: "private", Key: "config"}
		Expect(k8sClient.Update(ctx, config)).To(Succeed())
		expectReferences(metav1.ConditionFalse)
		secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "private", Namespace: namespace.Name}, Data: map[string][]byte{"config": []byte("service: {}")}}
		Expect(k8sClient.Create(ctx, secret)).To(Succeed())
		expectReferences(metav1.ConditionTrue)
		delete(namespace.Labels, InstallationLabel)
		Expect(k8sClient.Update(ctx, namespace)).To(Succeed())
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(group), group)).To(Succeed())
		group.Spec.Priority++
		Expect(k8sClient.Update(ctx, group)).To(Succeed())
		Consistently(func(g Gomega) {
			current := &api.AgentGroup{}
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(group), current)).To(Succeed())
			g.Expect(meta.FindStatusCondition(current.Status.Conditions, "Ready").ObservedGeneration).NotTo(Equal(current.Generation))
		}, 300*time.Millisecond, 50*time.Millisecond).Should(Succeed())
		namespace.Labels[InstallationLabel] = string(testInstallation.UID)
		Expect(k8sClient.Update(ctx, namespace)).To(Succeed())
		Eventually(func(g Gomega) {
			current := &api.AgentGroup{}
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(group), current)).To(Succeed())
			g.Expect(meta.FindStatusCondition(current.Status.Conditions, "Ready").ObservedGeneration).To(Equal(current.Generation))
		}, 10*time.Second, 50*time.Millisecond).Should(Succeed())
		stop()
		Expect(k8sClient.Delete(ctx, secret)).To(Succeed())
		start()
		expectReferences(metav1.ConditionFalse)
	})
})
