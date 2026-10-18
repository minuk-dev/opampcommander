package inmemory_test

import (
	"testing"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/inmemory"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
	"github.com/minuk-dev/opampcommander/pkg/testutil/resourcepatchtest"
)

func TestConditionalResourceWrites(t *testing.T) {
	t.Parallel()
	testutil.CheckResourceWrites(t,
		inmemory.NewAgentGroupRepository(inmemory.NewAgentRepository()),
		inmemory.NewAgentPackageRepository(), inmemory.NewAgentRemoteConfigRepository(), inmemory.NewNamespaceRepository())
}

func TestResourcePatches(t *testing.T) {
	t.Parallel()
	resourcepatchtest.CheckResourcePatches(t,
		inmemory.NewAgentGroupRepository(inmemory.NewAgentRepository()),
		inmemory.NewAgentPackageRepository(), inmemory.NewAgentRemoteConfigRepository(), inmemory.NewNamespaceRepository())
}
