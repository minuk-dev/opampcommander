package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	opampcommanderv1alpha1 "github.com/minuk-dev/opampcommander/operator/api/v1alpha1"
)

// AgentGroupReconciler reconciles a AgentGroup object
type AgentGroupReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=opampcommander.io,resources=agentgroups,verbs=get;list;watch
// +kubebuilder:rbac:groups=opampcommander.io,resources=agentgroups/status,verbs=get;update;patch

// Reconcile records that this scaffold cannot yet apply Agent configuration.
// It deliberately never reports Ready=True before runtime integration exists.
func (r *AgentGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	resource := &opampcommanderv1alpha1.AgentGroup{}
	if err := r.Get(ctx, req.NamespacedName, resource); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if !resource.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}
	before := resource.DeepCopy()
	changed := meta.SetStatusCondition(&resource.Status.Conditions, metav1.Condition{
		Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: resource.Generation,
		Reason: "NotImplemented", Message: "Agent configuration delivery is not implemented in this bootstrap controller",
	})
	if !changed {
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, r.Status().Patch(ctx, resource, client.MergeFrom(before))
}

// SetupWithManager sets up the controller with the Manager.
func (r *AgentGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&opampcommanderv1alpha1.AgentGroup{}).
		Named("agentgroup").
		Complete(r)
}
