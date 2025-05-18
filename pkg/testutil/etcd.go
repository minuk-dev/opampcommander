package testutil

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/stretchr/testify/require"
	"gotest.tools/icmd"
)

const (
	// DefaultEtcdVersion is the default version of etcd to use if not specified.
	DefaultEtcdVersion = "v3.6.0"
)

const (
	// DefaultEtcdStartWaitTime is the default wait time for etcd to start.
	DefaultEtcdStartWaitTime = 10 * time.Second

	// DefaultEtcdReadyCheckInterval is the default interval for checking if etcd is ready.
	DefaultEtcdReadyCheckInterval = 500 * time.Millisecond

	// DefaultEtcdDownloadTimeout is the default timeout for downloading etcd.
	DefaultEtcdDownloadTimeout = 3 * time.Minute
)

// UnsupportedPlatformError is an error type that indicates the platform is not supported for etcd installation.
type UnsupportedPlatformError struct {
	OS   string
	Arch string
}

// UnsupportedPlatformError implements the error interface for UnsupportedPlatformError.
func (e *UnsupportedPlatformError) Error() string {
	return fmt.Sprintf("unsupported platform: %s/%s", e.OS, e.Arch)
}

// UseEtcd is a method that initializes and starts an etcd instance for testing purposes.
func (b *Base) UseEtcd(config any) *Etcd {
	b.t.Helper()

	if _, ok := b.Dependencies["etcd"]; ok {
		b.t.Fatalf("etcd already exists")
	}

	etcd := NewEtcd(b)
	etcd.Configure(config)
	etcd.Start()

	b.Dependencies["etcd"] = etcd

	b.t.Cleanup(func() {
		etcd.Stop()
	})

	return etcd
}

// NewEtcd creates a new instance of Etcd.
func NewEtcd(base *Base) *Etcd {
	base.t.Helper()

	target := "etcd"

	binary, err := exec.LookPath("etcd")
	if err != nil {
		base.t.Logf("downloading etcd binary from cache directory: %s", base.CacheDir)

		binary, err = installEtcd(base.ctx, base.CacheDir, DefaultEtcdVersion)
		if err != nil {
			base.t.Logf("etcd binary not found in PATH & cannot install: %v", err)
		}
	}

	etcd := &Etcd{
		Base:   base,
		Target: target,
		Binary: binary,

		Endpoint: nil,
		result:   nil,
	}

	return etcd
}

func getEtcdDownloadURL(version string) (string, error) {
	switch runtime.GOOS {
	case "linux":
		switch runtime.GOARCH {
		case "amd64", "arm64":
			return fmt.Sprintf("https://github.com/etcd-io/etcd/releases/download/%s/etcd-%s-linux-%s.tar.gz",
				version, version, runtime.GOARCH), nil
		default:
			return "", &UnsupportedPlatformError{
				OS:   runtime.GOOS,
				Arch: runtime.GOARCH,
			}
		}
	case "darwin":
		switch runtime.GOARCH {
		case "amd64", "arm64":
			return fmt.Sprintf("https://github.com/etcd-io/etcd/releases/download/%s/etcd-%s-darwin-%s.zip",
				version, version, runtime.GOARCH), nil
		default:
			return "", &UnsupportedPlatformError{
				OS:   runtime.GOOS,
				Arch: runtime.GOARCH,
			}
		}
	default:
		return "", &UnsupportedPlatformError{
			OS:   runtime.GOOS,
			Arch: runtime.GOARCH,
		}
	}
}

//nolint:err113,mnd
func installEtcd(ctx context.Context, cacheDir, version string) (string, error) {
	versionDir := filepath.Join(cacheDir, version)
	binaryFile := filepath.Join(versionDir, "etcd")

	if alreadyInstalled(binaryFile) {
		return binaryFile, nil
	}

	fileInfo, err := os.Stat(versionDir) // Ensure versionDir
	if errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(versionDir, 0700)
		if err != nil {
			return "", fmt.Errorf("failed to create version: %v, err: %w", version, err)
		}
	} else if !fileInfo.IsDir() {
		return "", fmt.Errorf("cacheDir %s is not a directory", cacheDir)
	}

	downloadURL, err := getEtcdDownloadURL(version)
	if err != nil {
		return "", fmt.Errorf("failed to get download URL for version %s: %w", version, err)
	}

	_, filename := path.Split(downloadURL)

	downloadFilename := filepath.Join(versionDir, filename)

	err = downloadFile(ctx, downloadFilename, downloadURL)
	if err != nil {
		return "", fmt.Errorf("failed to download: %w", err)
	}

	switch filepath.Ext(downloadFilename) {
	case ".zip":
		err = unzipWithoutWrap(downloadFilename, versionDir)
		if err != nil {
			return "", fmt.Errorf("failed to unzip file: %w", err)
		}
	case ".tar.gz", ".tgz":
		err = decompressTarGz(downloadFilename, versionDir)
		if err != nil {
			return "", fmt.Errorf("failed to decompress file: %w", err)
		}
	default:
		return "", fmt.Errorf("unsupported file extension: %s", filepath.Ext(downloadFilename))
	}

	_, err = exec.LookPath(binaryFile)
	if err != nil {
		return "", fmt.Errorf("failed to find binary file: %w", err)
	}

	return binaryFile, nil
}

func closeSilently(closer io.Closer) {
	_ = closer.Close() // Ignore error
}

//nolint:wrapcheck,varnamelen,err113,mnd,gosec,funlen
func unzipWithoutWrap(src string, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer closeSilently(r)

	var rootDir string
	// Guess the root directory from the first file
	for _, f := range r.File {
		parts := strings.Split(f.Name, "/")
		if len(parts) > 1 {
			rootDir = parts[0]

			break
		}
	}

	for _, f := range r.File {
		relativePath := strings.TrimPrefix(f.Name, rootDir+"/")
		if relativePath == "" {
			continue
		}

		fpath := filepath.Join(dest, relativePath)

		// security check to prevent Zip Slip vulnerability
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			err := os.MkdirAll(fpath, 0750)
			if err != nil {
				return err
			}

			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), 0700); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		_, err = io.Copy(outFile, rc)

		closeSilently(outFile)
		closeSilently(rc)

		if err != nil {
			return err
		}
	}

	return nil
}

//nolint:wrapcheck,varnamelen,mnd,gosec
func decompressTarGz(filename, destDir string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer closeSilently(file)

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer closeSilently(gzr)

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()

		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		case header == nil:
			continue
		}

		target := filepath.Join(destDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			err := os.MkdirAll(target, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
		case tar.TypeReg:
			err := os.MkdirAll(filepath.Dir(target), 0700)
			if err != nil {
				return err
			}

			outFile, err := os.Create(target)
			if err != nil {
				return err
			}

			_, err = io.Copy(outFile, tr)
			if err != nil {
				closeSilently(outFile)

				return err
			}

			closeSilently(outFile)
		}
	}
}

//nolint:gosec
func downloadFile(ctx context.Context, filename, url string) error {
	client := &http.Client{
		Transport:     nil,
		CheckRedirect: nil,
		Jar:           nil,
		Timeout:       DefaultEtcdDownloadTimeout,
	}
	// Ensure idle connections are closed after use.
	// target download URL can use http2, so connection can be left.
	// This causes goleak to fail with "leaked goroutine" error.
	defer client.CloseIdleConnections()

	out, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer closeSilently(out)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to request: %w", err)
	}
	defer closeSilently(resp.Body)

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}

	return nil
}

func alreadyInstalled(binaryFile string) bool {
	_, err := exec.LookPath(binaryFile)

	return err == nil
}

// Etcd represents an etcd instance for testing purposes.
// It provides methods to start, stop, and check the status of the etcd instance.
type Etcd struct {
	Base *Base

	Target string
	Binary string

	Endpoint *string

	result *icmd.Result
}

// Name returns the name of the etcd instance.
func (e *Etcd) Name() string {
	return "etcd"
}

// Info returns information about the etcd instance.
func (e *Etcd) Info() map[string]string {
	e.Base.t.Helper()

	info := make(map[string]string)
	if e.Endpoint != nil {
		info["endpoint"] = *e.Endpoint
	} else {
		info["endpoint"] = "not set"
	}

	return info
}

// Configure configures the etcd instance with the provided configuration.
func (e *Etcd) Configure(config any) {
	e.Base.t.Helper()

	if config == nil {
		e.Base.t.Log("use default etcd configuration")

		config = map[string]string{
			"endpoint": "http://localhost:2379",
		}
	}

	// Configure etcd with the provided config.
	// This is a placeholder for actual configuration logic.
	configMap, ok := config.(map[string]string)

	if !ok {
		e.Base.t.Fatalf("expected config to be a map[string]string, got %T", config)
	}

	if endpoint, ok := configMap["endpoint"]; ok {
		e.Endpoint = &endpoint
	} else {
		e.Endpoint = P("http://localhost:2379")
	}
}

// Start starts the etcd instance.
func (e *Etcd) Start() {
	e.Base.t.Helper()

	// Start etcd.
	// This is a placeholder for actual start logic.
	e.Base.Logger.Info("Starting etcd")

	icmdCmd := icmd.Command(e.Binary)
	e.result = icmd.StartCmd(icmdCmd)

	err := e.WaitUntilReady(e.Base.ctx)
	if err != nil {
		e.Base.Logger.Warn("etcd is not ready", "error", err)
	}
}

// IsAlive checks if etcd is alive.
// livez is a liveness endpoint of etcd sinc v3.4.29
// ref. https://etcd.io/docs/v3.4/op-guide/monitoring/#health-check
func (e *Etcd) IsAlive(ctx context.Context) bool {
	liveEndpoint := must(url.JoinPath(*e.Endpoint, "/livez"))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, liveEndpoint, nil)
	if err != nil {
		e.Base.Logger.Error("failed to create request for livez endpoint", "error", err)
		e.Base.t.Fail()
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}

	defer closeSilently(resp.Body)

	return resp.StatusCode == http.StatusOK
}

// IsReady checks if etcd is ready to serve requests.
// readyz is a readiness endpoint of etcd since v3.4.29
// ref. https://etcd.io/docs/v3.4/op-guide/monitoring/#health-check
func (e *Etcd) IsReady() bool {
	readyEndpoint := must(url.JoinPath(*e.Endpoint, "/readyz"))

	req, err := http.NewRequestWithContext(e.Base.ctx, http.MethodGet, readyEndpoint, nil)
	if err != nil {
		e.Base.Logger.Error("failed to create request for readyz endpoint", "error", err)
		e.Base.t.Fail()
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}

	defer closeSilently(resp.Body)

	return resp.StatusCode == http.StatusOK
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(fmt.Sprintf("must: %v", err))
	}

	return v
}

// WaitUntilReady waits until etcd is ready.
//
//nolint:err113
func (e *Etcd) WaitUntilReady(ctx context.Context) error {
	e.Base.t.Helper()

	if e.Endpoint == nil {
		return errors.New("etcd endpoint is not set")
	}

	e.Base.Logger.Info("Waiting for etcd to be ready", "endpoint", *e.Endpoint)

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled while waiting for etcd to be ready: %w", ctx.Err())
		default:
			if e.IsReady() {
				e.Base.Logger.Info("etcd is ready")

				return nil
			}

			time.Sleep(DefaultEtcdReadyCheckInterval)
		}
	}
}

// Stop stops the etcd instance.
func (e *Etcd) Stop() {
	e.Base.t.Helper()

	// Stop etcd.
	// This is a placeholder for actual stop logic.
	e.Base.Logger.Info("Stopping etcd")

	if e.result != nil && e.result.Cmd != nil && e.result.Cmd.Process != nil {
		err := e.result.Cmd.Process.Signal(syscall.SIGTERM)
		require.NoError(e.Base.t, err)

		ctx, cancel := context.WithTimeout(e.Base.ctx, DefaultEtcdStartWaitTime)
		defer cancel()

		stopCh := make(chan struct{})
		go func() {
			defer close(stopCh)

			_, err := e.result.Cmd.Process.Wait()
			if err != nil {
				e.Base.Logger.Error("etcd process wait failed", "error", err)
			} else {
				e.Base.Logger.Info("etcd process exited successfully")
			}
		}()

		select {
		case <-ctx.Done():
			e.Base.Logger.Warn("etcd stop timed out, force killing process")

			if err := e.result.Cmd.Process.Kill(); err != nil {
				e.Base.Logger.Error("failed to kill etcd process", "error", err)
			}
		case <-stopCh:
			e.Base.Logger.Info("etcd stopped successfully")
		}
	}
}
