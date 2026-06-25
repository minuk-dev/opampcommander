package reconcile_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/reconcile"
)

var errBoom = errors.New("boom")

// fakeReconciler records the last reconcile call for assertions.
type fakeReconciler struct {
	kind      string
	err       error
	gotNS     string
	gotName   string
	callCount int
}

func (f *fakeReconciler) Kind() string { return f.kind }

func (f *fakeReconciler) Reconcile(_ context.Context, namespace, name string) error {
	f.callCount++
	f.gotNS = namespace
	f.gotName = name

	return f.err
}

func TestService_Reconcile(t *testing.T) {
	t.Parallel()

	t.Run("dispatches to the reconciler registered for the kind", func(t *testing.T) {
		t.Parallel()

		target := &fakeReconciler{kind: "agentgroup"}
		other := &fakeReconciler{kind: "agent"}

		svc, err := reconcile.NewService([]reconcile.Reconciler{target, other})
		require.NoError(t, err)

		err = svc.Reconcile(context.Background(), "agentgroup", "default", "obs")

		require.NoError(t, err)
		assert.Equal(t, 1, target.callCount)
		assert.Equal(t, 0, other.callCount)
		assert.Equal(t, "default", target.gotNS)
		assert.Equal(t, "obs", target.gotName)
	})

	t.Run("returns ErrUnknownKind for an unregistered kind", func(t *testing.T) {
		t.Parallel()

		svc, err := reconcile.NewService([]reconcile.Reconciler{&fakeReconciler{kind: "agent"}})
		require.NoError(t, err)

		err = svc.Reconcile(context.Background(), "doesnotexist", "default", "x")

		require.Error(t, err)
		assert.ErrorIs(t, err, reconcile.ErrUnknownKind)
	})

	t.Run("propagates the reconciler's error", func(t *testing.T) {
		t.Parallel()

		svc, err := reconcile.NewService([]reconcile.Reconciler{&fakeReconciler{kind: "agent", err: errBoom}})
		require.NoError(t, err)

		err = svc.Reconcile(context.Background(), "agent", "default", "id")

		require.Error(t, err)
		assert.ErrorIs(t, err, errBoom)
	})

	t.Run("rejects duplicate kinds", func(t *testing.T) {
		t.Parallel()

		_, err := reconcile.NewService([]reconcile.Reconciler{
			&fakeReconciler{kind: "agent"},
			&fakeReconciler{kind: "agent"},
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, reconcile.ErrDuplicateKind)
	})
}

func TestService_Kinds(t *testing.T) {
	t.Parallel()

	svc, err := reconcile.NewService([]reconcile.Reconciler{
		&fakeReconciler{kind: "agentremoteconfig"},
		&fakeReconciler{kind: "agent"},
		&fakeReconciler{kind: "agentgroup"},
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"agent", "agentgroup", "agentremoteconfig"}, svc.Kinds())
}
