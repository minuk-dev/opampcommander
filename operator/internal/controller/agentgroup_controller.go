package controller

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	controlleroptions "sigs.k8s.io/controller-runtime/pkg/controller"

	opampcommanderv1alpha1 "github.com/minuk-dev/opampcommander/operator/api/v1alpha1"
)

// AgentGroupReconciler reconciles a AgentGroup object
type AgentGroupReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=opampcommander.io,resources=agentgroups,verbs=get;list;watch
// +kubebuilder:rbac:groups=opampcommander.io,resources=agentgroups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=opampcommander.io,resources=remoteconfigs;agentpackages;remoteconfigschemas;endpoints,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

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
	found, readErr := r.referencesFound(ctx, resource, make(map[dependency]bool))
	references := metav1.Condition{
		Type: "ReferencesFound", Status: metav1.ConditionTrue, ObservedGeneration: resource.Generation,
		Reason: "Found", Message: "Referenced objects exist; contents and Agent delivery have not been validated",
	}
	if !found {
		references.Status = metav1.ConditionFalse
		references.Reason = "DependencyMissing"
		references.Message = "A referenced object is missing or being deleted"
	}
	if readErr != nil {
		references.Status = metav1.ConditionUnknown
		references.Reason = "ReadFailed"
		references.Message = "Could not read referenced objects"
	}
	referencesChanged := meta.SetStatusCondition(&resource.Status.Conditions, references)
	changed := meta.SetStatusCondition(&resource.Status.Conditions, metav1.Condition{
		Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: resource.Generation,
		Reason: "NotImplemented", Message: "Agent configuration delivery is not implemented in this bootstrap controller",
	})
	if changed || referencesChanged {
		if err := r.Status().Patch(ctx, resource, client.MergeFromWithOptions(before, client.MergeFromWithOptimisticLock{})); err != nil {
			return ctrl.Result{}, err
		}
	}
	if readErr != nil {
		return ctrl.Result{}, readErr
	}
	// Repair missed dependency notifications with bounded workers and one timer per group.
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AgentGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	controller := ctrl.NewControllerManagedBy(mgr).
		For(&opampcommanderv1alpha1.AgentGroup{}, builder.WithPredicates(configurationChanged)).
		Named("agentgroup").
		WithOptions(controlleroptions.Options{MaxConcurrentReconciles: 2})
	if err := r.watchDependencies(mgr, controller); err != nil {
		return err
	}
	return controller.Complete(r)
}
