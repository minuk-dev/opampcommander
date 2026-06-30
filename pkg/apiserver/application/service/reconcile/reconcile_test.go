package reconcile_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	reconcilesvc "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/reconcile"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	domainreconcile "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/reconcile"
)

var errReconcile = errors.New("reconcile failed")

// fakeReconciler is a domainreconcile.Reconciler whose Reconcile returns reconcileErr,
// recording the last namespace/name it was called with.
type fakeReconciler struct {
	kind          string
	reconcileErr  error
	lastNamespace string
	lastName      string
}

func (f *fakeReconciler) Kind() string { return f.kind }

func (f *fakeReconciler) Reconcile(_ context.Context, namespace, name string) error {
	f.lastNamespace = namespace
	f.lastName = name

	return f.reconcileErr
}

func newSvc(t *testing.T, reconcilers ...domainreconcile.Reconciler) *reconcilesvc.Service {
	t.Helper()

	registry, err := domainreconcile.NewService(reconcilers)
	require.NoError(t, err)

	return reconcilesvc.New(registry)
}

func TestService_Reconcile(t *testing.T) {
	t.Parallel()

	t.Run("success delegates to the matching reconciler", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		rec := &fakeReconciler{kind: "AgentGroup"}
		svc := newSvc(t, rec)

		err := svc.Reconcile(ctx, "AgentGroup", "default", "group-1")

		require.NoError(t, err)
		assert.Equal(t, "default", rec.lastNamespace)
		assert.Equal(t, "group-1", rec.lastName)
	})

	t.Run("unknown kind maps to ErrInvalidArgument", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		svc := newSvc(t, &fakeReconciler{kind: "AgentGroup"})

		err := svc.Reconcile(ctx, "Nonexistent", "default", "x")

		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrInvalidArgument)
		assert.ErrorIs(t, err, domainreconcile.ErrUnknownKind)
	})

	t.Run("reconciler error is wrapped", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		svc := newSvc(t, &fakeReconciler{kind: "AgentGroup", reconcileErr: errReconcile})

		err := svc.Reconcile(ctx, "AgentGroup", "default", "group-1")

		require.Error(t, err)
		assert.ErrorIs(t, err, errReconcile)
		assert.NotErrorIs(t, err, model.ErrInvalidArgument)
	})
}

func TestService_ReconcileKinds(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	svc := newSvc(t, &fakeReconciler{kind: "AgentGroup"}, &fakeReconciler{kind: "AgentRemoteConfig"})

	kinds := svc.ReconcileKinds(ctx)

	assert.Equal(t, []string{"AgentGroup", "AgentRemoteConfig"}, kinds)
}
