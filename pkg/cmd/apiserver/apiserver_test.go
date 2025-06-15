package apiserver_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"

	"github.com/minuk-dev/opampcommander/pkg/cmd/apiserver"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestCommand(t *testing.T) {
	t.Parallel()
	base := testutil.NewBase(t)

	etcd := base.UseEtcd(nil)
	//exhaustruct:ignore
	client := &http.Client{}

	// given
	cmd := apiserver.NewCommand(
		//exhaustruct:ignore
		apiserver.CommandOption{},
	)

	port := base.GetFreeTCPPort()

	cmd.SetArgs([]string{
		"--address", fmt.Sprintf("%s:%d", "localhost", port),
		"--database.endpoints", *etcd.Endpoint,
	})

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var waitGroup sync.WaitGroup

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

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

		defer func() {
			if closeErr := resp.Body.Close(); closeErr != nil {
				t.Logf("failed to close response body: %v", closeErr)
			}
		}()

		return resp.StatusCode == http.StatusOK
	}, 5*time.Second, 100*time.Millisecond, "API server should be ready")

	// Stop the server
	cancel()
	waitGroup.Wait()
}
