package agentservice_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

var (
	errNotImplemented = errors.New("not implemented")
	errCascade        = errors.New("cascade failure")
)

// nsRecordingTxRunner records whether WithinTransaction was invoked and forwards
// the callback. It tags the context so callees can prove they observed the
// transactional ctx.
type nsRecordingTxRunner struct {
	calls int
}

type nsTxMarkerKey struct{}

func (r *nsRecordingTxRunner) WithinTransaction(
	ctx context.Context,
	fn func(context.Context) error,
) error {
	r.calls++
	txCtx := context.WithValue(ctx, nsTxMarkerKey{}, true)

	return fn(txCtx)
}

// --- fake persistence + usecases --------------------------------------

type nsFakeNamespacePersistence struct {
	stored      *agentmodel.Namespace
	getErr      error
	putCalls    int
	putCtxTxOK  bool
	lastPutBody *agentmodel.Namespace
}

func (f *nsFakeNamespacePersistence) GetNamespace(
	_ context.Context, _ string, _ *model.GetOptions,
) (*agentmodel.Namespace, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}

	return f.stored, nil
}

func (f *nsFakeNamespacePersistence) PutNamespace(
	ctx context.Context, namespace *agentmodel.Namespace,
) (*agentmodel.Namespace, error) {
	f.putCalls++
	f.lastPutBody = namespace

	if v, _ := ctx.Value(nsTxMarkerKey{}).(bool); v {
		f.putCtxTxOK = true
	}

	return namespace, nil
}

func (f *nsFakeNamespacePersistence) ListNamespaces(
	_ context.Context, _ *model.ListOptions,
) (*model.ListResponse[*agentmodel.Namespace], error) {
	return &model.ListResponse[*agentmodel.Namespace]{}, nil
}

type nsFakeAgentGroupUsecase struct {
	items     []*agentmodel.AgentGroup
	deleteErr error
}

func (f *nsFakeAgentGroupUsecase) GetAgentGroup(
	context.Context, string, string, *model.GetOptions,
) (*agentmodel.AgentGroup, error) {
	return nil, errNotImplemented
}

func (f *nsFakeAgentGroupUsecase) ListAgentGroups(
	context.Context, *model.ListOptions,
) (*model.ListResponse[*agentmodel.AgentGroup], error) {
	return &model.ListResponse[*agentmodel.AgentGroup]{Items: f.items}, nil
}

func (f *nsFakeAgentGroupUsecase) SaveAgentGroup(
	context.Context, string, string, *agentmodel.AgentGroup,
) (*agentmodel.AgentGroup, error) {
	return nil, errNotImplemented
}

func (f *nsFakeAgentGroupUsecase) DeleteAgentGroup(
	context.Context, string, string, time.Time, string,
) error {
	return f.deleteErr
}

func (f *nsFakeAgentGroupUsecase) GetAgentGroupsForAgent(
	context.Context, *agentmodel.Agent,
) ([]*agentmodel.AgentGroup, error) {
	return nil, errNotImplemented
}

func (f *nsFakeAgentGroupUsecase) PropagateAgentRemoteConfigChange(
	context.Context, string, string,
) error {
	return nil
}

func (f *nsFakeAgentGroupUsecase) ApplyMatchingAgentGroupsToAgent(
	context.Context, *agentmodel.Agent,
) error {
	return nil
}

func (f *nsFakeAgentGroupUsecase) ReconcileAgent(
	context.Context, *agentmodel.Agent,
) error {
	return nil
}

func (f *nsFakeAgentGroupUsecase) ReconcileAgentGroup(
	context.Context, string, string,
) error {
	return nil
}

type nsFakeCertificateUsecase struct{}

func (f *nsFakeCertificateUsecase) GetCertificate(
	context.Context, string, string, *model.GetOptions,
) (*agentmodel.Certificate, error) {
	return nil, errNotImplemented
}

func (f *nsFakeCertificateUsecase) SaveCertificate(
	context.Context, *agentmodel.Certificate,
) (*agentmodel.Certificate, error) {
	return nil, errNotImplemented
}

func (f *nsFakeCertificateUsecase) CreateCertificate(
	context.Context, *agentmodel.Certificate, string,
) (*agentmodel.Certificate, error) {
	return nil, errNotImplemented
}

func (f *nsFakeCertificateUsecase) UpdateCertificate(
	context.Context, string, string, *agentmodel.Certificate, string,
) (*agentmodel.Certificate, error) {
	return nil, errNotImplemented
}

func (f *nsFakeCertificateUsecase) ListCertificate(
	context.Context, *model.ListOptions,
) (*model.ListResponse[*agentmodel.Certificate], error) {
	return &model.ListResponse[*agentmodel.Certificate]{}, nil
}

func (f *nsFakeCertificateUsecase) DeleteCertificate(
	context.Context, string, string, time.Time, string,
) (*agentmodel.Certificate, error) {
	return nil, errNotImplemented
}

type nsFakeAgentPackageUsecase struct{}

func (f *nsFakeAgentPackageUsecase) GetAgentPackage(
	context.Context, string, string, *model.GetOptions,
) (*agentmodel.AgentPackage, error) {
	return nil, errNotImplemented
}

func (f *nsFakeAgentPackageUsecase) ListAgentPackages(
	context.Context, *model.ListOptions,
) (*model.ListResponse[*agentmodel.AgentPackage], error) {
	return &model.ListResponse[*agentmodel.AgentPackage]{}, nil
}

func (f *nsFakeAgentPackageUsecase) SaveAgentPackage(
	context.Context, *agentmodel.AgentPackage,
) (*agentmodel.AgentPackage, error) {
	return nil, errNotImplemented
}

func (f *nsFakeAgentPackageUsecase) CreateAgentPackage(
	context.Context, *agentmodel.AgentPackage, string,
) (*agentmodel.AgentPackage, error) {
	return nil, errNotImplemented
}

func (f *nsFakeAgentPackageUsecase) UpdateAgentPackage(
	context.Context, string, string, *agentmodel.AgentPackage,
) (*agentmodel.AgentPackage, error) {
	return nil, errNotImplemented
}

func (f *nsFakeAgentPackageUsecase) DeleteAgentPackage(
	context.Context, string, string, time.Time, string,
) error {
	return nil
}

type nsFakeAgentRemoteConfigUsecase struct{}

func (f *nsFakeAgentRemoteConfigUsecase) GetAgentRemoteConfig(
	context.Context, string, string, *model.GetOptions,
) (*agentmodel.AgentRemoteConfig, error) {
	return nil, errNotImplemented
}

func (f *nsFakeAgentRemoteConfigUsecase) ListAgentRemoteConfigs(
	context.Context, *model.ListOptions,
) (*model.ListResponse[*agentmodel.AgentRemoteConfig], error) {
	return &model.ListResponse[*agentmodel.AgentRemoteConfig]{}, nil
}

func (f *nsFakeAgentRemoteConfigUsecase) SaveAgentRemoteConfig(
	context.Context, *agentmodel.AgentRemoteConfig,
) (*agentmodel.AgentRemoteConfig, error) {
	return nil, errNotImplemented
}

func (f *nsFakeAgentRemoteConfigUsecase) CreateAgentRemoteConfig(
	context.Context, *agentmodel.AgentRemoteConfig, string,
) (*agentmodel.AgentRemoteConfig, error) {
	return nil, errNotImplemented
}

func (f *nsFakeAgentRemoteConfigUsecase) UpdateAgentRemoteConfig(
	context.Context, string, string, *agentmodel.AgentRemoteConfig,
) (*agentmodel.AgentRemoteConfig, error) {
	return nil, errNotImplemented
}

func (f *nsFakeAgentRemoteConfigUsecase) DeleteAgentRemoteConfig(
	context.Context, string, string, time.Time, string,
) error {
	return nil
}

func (f *nsFakeAgentRemoteConfigUsecase) ReconcileAgentRemoteConfig(
	context.Context, string, string,
) error {
	return nil
}

func newNamespaceService(
	persistence agentport.NamespacePersistencePort,
	groupUC agentport.AgentGroupUsecase,
	tx agentport.TransactionPort,
) *agentservice.NamespaceService {
	return agentservice.NewNamespaceService(
		persistence,
		groupUC,
		&nsFakeCertificateUsecase{},
		&nsFakeAgentPackageUsecase{},
		&nsFakeAgentRemoteConfigUsecase{},
		tx,
		agentmodel.DefaultNamespaceName,
	)
}

// --- tests ------------------------------------------------------------

func TestDeleteNamespace_RunsCascadeInsideTransaction(t *testing.T) {
	t.Parallel()

	runner := &nsRecordingTxRunner{}
	persistence := &nsFakeNamespacePersistence{stored: agentmodel.NewNamespace("team-a")}

	svc := newNamespaceService(persistence, &nsFakeAgentGroupUsecase{}, runner)

	err := svc.DeleteNamespace(t.Context(), "team-a", "tester")

	require.NoError(t, err)
	assert.Equal(t, 1, runner.calls, "WithinTransaction must be invoked exactly once")
	assert.Equal(t, 1, persistence.putCalls, "namespace itself should be soft-deleted at end of cascade")
	assert.True(t, persistence.putCtxTxOK, "cascade must run with the transactional context")
	require.NotNil(t, persistence.lastPutBody)
	assert.True(t, persistence.lastPutBody.IsDeleted(), "the persisted namespace must be marked deleted")
}

func TestDeleteNamespace_MidCascadeFailureAbortsNamespaceDelete(t *testing.T) {
	t.Parallel()

	runner := &nsRecordingTxRunner{}
	persistence := &nsFakeNamespacePersistence{stored: agentmodel.NewNamespace("team-a")}

	// Seed one agent group in the namespace and force its delete to fail.
	group := agentmodel.NewAgentGroup(
		"team-a",
		"grp",
		agentmodel.OfAttributes(nil),
		time.Now(),
		"tester",
	)
	groupUC := &nsFakeAgentGroupUsecase{
		items:     []*agentmodel.AgentGroup{group},
		deleteErr: errCascade,
	}

	svc := newNamespaceService(persistence, groupUC, runner)

	err := svc.DeleteNamespace(t.Context(), "team-a", "tester")

	require.Error(t, err)
	assert.Equal(t, 1, runner.calls)
	assert.Zero(t, persistence.putCalls,
		"namespace must NOT be deleted if a child cascade step fails — the transaction wrap is what protects partial state")
}

func TestDeleteNamespace_RejectsDefaultNamespace(t *testing.T) {
	t.Parallel()

	runner := &nsRecordingTxRunner{}

	svc := newNamespaceService(
		&nsFakeNamespacePersistence{}, &nsFakeAgentGroupUsecase{}, runner,
	)

	err := svc.DeleteNamespace(t.Context(), agentmodel.DefaultNamespaceName, "tester")

	require.ErrorIs(t, err, agentport.ErrDefaultNamespaceUndeletable)
	assert.Zero(t, runner.calls, "default-namespace guard must short-circuit before any transaction")
}

func TestCreateNamespace_RejectsDuplicate(t *testing.T) {
	t.Parallel()

	persistence := &nsFakeNamespacePersistence{stored: agentmodel.NewNamespace("team-a")}
	svc := newNamespaceService(persistence, &nsFakeAgentGroupUsecase{}, &nsRecordingTxRunner{})

	_, err := svc.CreateNamespace(t.Context(), agentmodel.NewNamespace("team-a"), "tester")

	require.ErrorIs(t, err, agentport.ErrNamespaceAlreadyExists)
	assert.Zero(t, persistence.putCalls, "a duplicate must not be persisted")
}

func TestCreateNamespace_StampsCreation(t *testing.T) {
	t.Parallel()

	// getErr forces the existence check to treat the namespace as not present.
	persistence := &nsFakeNamespacePersistence{getErr: errNotImplemented}
	svc := newNamespaceService(persistence, &nsFakeAgentGroupUsecase{}, &nsRecordingTxRunner{})

	created, err := svc.CreateNamespace(t.Context(), agentmodel.NewNamespace("team-a"), "tester")

	require.NoError(t, err)
	assert.Equal(t, 1, persistence.putCalls)
	require.NotEmpty(t, created.Status.Conditions, "creation must record a condition")

	cond := created.Status.Conditions[0]
	assert.Equal(t, model.ConditionTypeCreated, cond.Type)
	assert.Equal(t, "tester", cond.Reason, "the acting user must be stamped as the condition reason")
}

func TestUpdateNamespace_PreservesImmutableFields(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	stored := agentmodel.NewNamespace("team-a")
	stored.Metadata.CreatedAt = createdAt
	stored.MarkAsCreated(createdAt, "creator")

	persistence := &nsFakeNamespacePersistence{stored: stored}
	svc := newNamespaceService(persistence, &nsFakeAgentGroupUsecase{}, &nsRecordingTxRunner{})

	// The incoming model tries to mutate immutable fields and the labels.
	incoming := agentmodel.NewNamespace("team-a")
	incoming.Metadata.CreatedAt = time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	incoming.Metadata.Labels = map[string]string{"team": "a"}

	updated, err := svc.UpdateNamespace(t.Context(), "team-a", incoming)

	require.NoError(t, err)
	assert.Equal(t, createdAt, updated.Metadata.CreatedAt, "CreatedAt must be preserved from the stored namespace")
	assert.Equal(t, map[string]string{"team": "a"}, updated.Metadata.Labels, "mutable labels must be applied")
	assert.NotEmpty(t, updated.Status.Conditions, "existing lifecycle conditions must be preserved")
}
