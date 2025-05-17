package apiserver_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/cmd/apiserver"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestCommand(t *testing.T) {
	t.Parallel()

	err := os.Setenv("OPAMP_COMMANDER_TESTING_DIR", "/Users/min-uklee/workspace/repos/opampcommander/tmp")
	require.NoError(t, err)

	base := testutil.NewBase(t)

	etcd := base.UseEtcd(nil)
	//exhaustruct:ignore
	client := &http.Client{}

	// given
	cmd := apiserver.NewCommand(apiserver.CommandOption{})

	port := base.GetFreeTCPPort()

	cmd.SetArgs([]string{
		"--addr", fmt.Sprintf("%s:%d", "localhost", port),
		"--db-host", *etcd.Endpoint,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()

		// when
		err := cmd.ExecuteContext(ctx)
		assert.NoError(t, err)
	}()

	// then
	assert.Eventually(t, func() bool {
		base := net.JoinHostPort("localhost", strconv.Itoa(port))
		pingURL := fmt.Sprintf("http://%s/api/v1/ping", base)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, pingURL, nil)
		if err != nil {
			return false
		}

		resp, err := client.Do(req)

		if err != nil {
			return false
		}

		defer resp.Body.Close()

		return resp.StatusCode == http.StatusOK
	}, 5*time.Second, 100*time.Millisecond, "API server should be ready")

	// Stop the server
	cancel()
	wg.Wait()
}
