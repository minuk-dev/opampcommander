package namespace_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	namespacesvc "github.com/minuk-dev/opampcommander/internal/application/service/namespace"
	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/agent/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

var (
	errNotImplemented = errors.New("not implemented")
	errCascade        = errors.New("cascade failure")
)

// recordingTxRunner records whether WithinTransaction was invoked and
// forwards the callback. It also tags the context so callees can prove they
// observed the transactional ctx.
type recordingTxRunner struct {
	calls int
}

type txMarkerKey struct{}

func (r *recordingTxRunner) WithinTransaction(
	ctx context.Context,
	fn func(context.Context) error,
) error {
	r.calls++
	txCtx := context.WithValue(ctx, txMarkerKey{}, true)

	return fn(txCtx)
}

// --- fake usecases -----------------------------------------------------

type fakeNamespaceUsecase struct {
	deleteCalls int
	deleteCtxOK bool
	deleteErr   error
}

func (f *fakeNamespaceUsecase) GetNamespace(
	context.Context, string, *model.GetOptions,
) (*agentmodel.Namespace, error) {
	return nil, errNotImplemented
}

func (f *fakeNamespaceUsecase) ListNamespaces(
	context.Context, *model.ListOptions,
) (*model.ListResponse[*agentmodel.Namespace], error) {
	return nil, errNotImplemented
}

func (f *fakeNamespaceUsecase) SaveNamespace(
	context.Context, *agentmodel.Namespace,
) (*agentmodel.Namespace, error) {
	return nil, errNotImplemented
}

func (f *fakeNamespaceUsecase) DeleteNamespace(
	ctx context.Context, _ string, _ time.Time, _ string,
) error {
	f.deleteCalls++

	if v, _ := ctx.Value(txMarkerKey{}).(bool); v {
		f.deleteCtxOK = true
	}

	return f.deleteErr
}

type fakeAgentGroupUsecase struct {
	items     []*agentmodel.AgentGroup
	deleteErr error
}

func (f *fakeAgentGroupUsecase) GetAgentGroup(
	context.Context, string, string, *model.GetOptions,
) (*agentmodel.AgentGroup, error) {
	return nil, errNotImplemented
}

func (f *fakeAgentGroupUsecase) ListAgentGroups(
	context.Context, *model.ListOptions,
) (*model.ListResponse[*agentmodel.AgentGroup], error) {
	return &model.ListResponse[*agentmodel.AgentGroup]{Items: f.items}, nil
}

func (f *fakeAgentGroupUsecase) SaveAgentGroup(
	context.Context, string, string, *agentmodel.AgentGroup,
) (*agentmodel.AgentGroup, error) {
	return nil, errNotImplemented
}

func (f *fakeAgentGroupUsecase) DeleteAgentGroup(
	context.Context, string, string, time.Time, string,
) error {
	return f.deleteErr
}

func (f *fakeAgentGroupUsecase) GetAgentGroupsForAgent(
	context.Context, *agentmodel.Agent,
) ([]*agentmodel.AgentGroup, error) {
	return nil, errNotImplemented
}

func (f *fakeAgentGroupUsecase) PropagateAgentRemoteConfigChange(
	context.Context, string, string,
) error {
	return nil
}

func (f *fakeAgentGroupUsecase) ApplyMatchingAgentGroupsToAgent(
	context.Context, *agentmodel.Agent,
) error {
	return nil
}

type fakeCertificateUsecase struct{}

func (f *fakeCertificateUsecase) GetCertificate(
	context.Context, string, string, *model.GetOptions,
) (*agentmodel.Certificate, error) {
	return nil, errNotImplemented
}

func (f *fakeCertificateUsecase) SaveCertificate(
	context.Context, *agentmodel.Certificate,
) (*agentmodel.Certificate, error) {
	return nil, errNotImplemented
}

func (f *fakeCertificateUsecase) ListCertificate(
	context.Context, *model.ListOptions,
) (*model.ListResponse[*agentmodel.Certificate], error) {
	return &model.ListResponse[*agentmodel.Certificate]{}, nil
}

func (f *fakeCertificateUsecase) DeleteCertificate(
	context.Context, string, string, time.Time, string,
) (*agentmodel.Certificate, error) {
	return nil, errNotImplemented
}

type fakeAgentPackageUsecase struct{}

func (f *fakeAgentPackageUsecase) GetAgentPackage(
	context.Context, string, string, *model.GetOptions,
) (*agentmodel.AgentPackage, error) {
	return nil, errNotImplemented
}

func (f *fakeAgentPackageUsecase) ListAgentPackages(
	context.Context, *model.ListOptions,
) (*model.ListResponse[*agentmodel.AgentPackage], error) {
	return &model.ListResponse[*agentmodel.AgentPackage]{}, nil
}

func (f *fakeAgentPackageUsecase) SaveAgentPackage(
	context.Context, *agentmodel.AgentPackage,
) (*agentmodel.AgentPackage, error) {
	return nil, errNotImplemented
}

func (f *fakeAgentPackageUsecase) DeleteAgentPackage(
	context.Context, string, string, time.Time, string,
) error {
	return nil
}

type fakeAgentRemoteConfigUsecase struct{}

func (f *fakeAgentRemoteConfigUsecase) GetAgentRemoteConfig(
	context.Context, string, string, *model.GetOptions,
) (*agentmodel.AgentRemoteConfig, error) {
	return nil, errNotImplemented
}

func (f *fakeAgentRemoteConfigUsecase) ListAgentRemoteConfigs(
	context.Context, *model.ListOptions,
) (*model.ListResponse[*agentmodel.AgentRemoteConfig], error) {
	return &model.ListResponse[*agentmodel.AgentRemoteConfig]{}, nil
}

func (f *fakeAgentRemoteConfigUsecase) SaveAgentRemoteConfig(
	context.Context, *agentmodel.AgentRemoteConfig,
) (*agentmodel.AgentRemoteConfig, error) {
	return nil, errNotImplemented
}

func (f *fakeAgentRemoteConfigUsecase) DeleteAgentRemoteConfig(
	context.Context, string, string, time.Time, string,
) error {
	return nil
}

// --- tests ------------------------------------------------------------

func TestDeleteNamespace_RunsCascadeInsideTransaction(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)
	runner := &recordingTxRunner{}
	nsUC := &fakeNamespaceUsecase{}

	svc := namespacesvc.NewNamespaceService(
		nsUC,
		&fakeAgentGroupUsecase{},
		&fakeCertificateUsecase{},
		&fakeAgentPackageUsecase{},
		&fakeAgentRemoteConfigUsecase{},
		runner,
		base.Logger,
	)

	err := svc.DeleteNamespace(t.Context(), "team-a")

	require.NoError(t, err)
	assert.Equal(t, 1, runner.calls, "WithinTransaction must be invoked exactly once")
	assert.Equal(t, 1, nsUC.deleteCalls, "namespace itself should be deleted at end of cascade")
	assert.True(t, nsUC.deleteCtxOK, "cascade must run with the transactional context")
}

func TestDeleteNamespace_MidCascadeFailureAbortsNamespaceDelete(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)
	runner := &recordingTxRunner{}
	nsUC := &fakeNamespaceUsecase{}

	// Seed one agent group in the namespace and force its delete to fail.
	group := agentmodel.NewAgentGroup(
		"team-a",
		"grp",
		agentmodel.OfAttributes(nil),
		time.Now(),
		"tester",
	)
	groupUC := &fakeAgentGroupUsecase{
		items:     []*agentmodel.AgentGroup{group},
		deleteErr: errCascade,
	}

	svc := namespacesvc.NewNamespaceService(
		nsUC,
		groupUC,
		&fakeCertificateUsecase{},
		&fakeAgentPackageUsecase{},
		&fakeAgentRemoteConfigUsecase{},
		runner,
		base.Logger,
	)

	err := svc.DeleteNamespace(t.Context(), "team-a")

	require.Error(t, err)
	assert.Equal(t, 1, runner.calls)
	assert.Zero(t, nsUC.deleteCalls,
		"namespace must NOT be deleted if a child cascade step fails — the transaction wrap is what protects partial state")
}

func TestDeleteNamespace_RejectsDefaultNamespace(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)
	runner := &recordingTxRunner{}

	svc := namespacesvc.NewNamespaceService(
		&fakeNamespaceUsecase{},
		&fakeAgentGroupUsecase{},
		&fakeCertificateUsecase{},
		&fakeAgentPackageUsecase{},
		&fakeAgentRemoteConfigUsecase{},
		runner,
		base.Logger,
	)

	err := svc.DeleteNamespace(t.Context(), agentmodel.DefaultNamespaceName)

	require.ErrorIs(t, err, namespacesvc.ErrDefaultNamespaceUndeletable)
	assert.Zero(t, runner.calls, "default-namespace guard must short-circuit before any transaction")
}
