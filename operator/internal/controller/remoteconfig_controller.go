package controller

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	opampcommanderv1alpha1 "github.com/minuk-dev/opampcommander/operator/api/v1alpha1"
)

// RemoteConfigReconciler reconciles a RemoteConfig object
type RemoteConfigReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	Installation *Installation
}

// +kubebuilder:rbac:groups=opampcommander.io,resources=remoteconfigs,verbs=get;list;watch
// +kubebuilder:rbac:groups=opampcommander.io,resources=remoteconfigs/status,verbs=get;update;patch

// Reconcile records that this scaffold cannot yet apply Agent configuration.
// It deliberately never reports Ready=True before runtime integration exists.
func (r *RemoteConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	resource := &opampcommanderv1alpha1.RemoteConfig{}
	if err := r.Get(ctx, req.NamespacedName, resource); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if !resource.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}
	owned, err := r.Installation.ownedNamespace(ctx, req.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}
	if owned == nil {
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
	}
	before := resource.DeepCopy()
	changed := meta.SetStatusCondition(&resource.Status.Conditions, metav1.Condition{
		Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: resource.Generation,
		Reason: "NotImplemented", Message: "Agent configuration delivery is not implemented in this bootstrap controller",
	})
	if changed {
		if err := r.Status().Patch(ctx, resource, client.MergeFromWithOptions(before, client.MergeFromWithOptimisticLock{})); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RemoteConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&opampcommanderv1alpha1.RemoteConfig{}, builder.WithPredicates(configurationChanged)).
		Watches(&corev1.Namespace{}, handler.EnqueueRequestsFromMapFunc(enqueueNamespace(r.Client,
			func() client.ObjectList { return &opampcommanderv1alpha1.RemoteConfigList{} })), builder.WithPredicates(predicate.LabelChangedPredicate{})).
		Named("remoteconfig").
		Complete(r)
}
