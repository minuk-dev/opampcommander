package apiserver_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/minuk-dev/opampcommander/pkg/cmd/apiserver"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
	"github.com/stretchr/testify/assert"
)

func TestCommand(t *testing.T) {
	os.Setenv("OPAMP_COMMANDER_TESTING_DIR", "/Users/min-uklee/workspace/repos/opampcommander/tmp")
	base := testutil.NewBase(t)

	etcd := base.UseEtcd(nil)
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
		cmd.ExecuteContext(ctx)
	}()

	// then
	assert.Eventually(t, func() bool {
		pingURL := fmt.Sprintf("http://%s:%d/api/v1/ping", "localhost", port)
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
