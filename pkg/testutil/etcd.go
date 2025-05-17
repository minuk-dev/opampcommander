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

	"gotest.tools/icmd"
)

const (
	DefaultEtcdVersion = "v3.6.0"
)

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

func NewEtcd(b *Base) *Etcd {
	b.t.Helper()

	target := "etcd"
	binary, err := exec.LookPath("etcd")
	if err != nil {
		b.t.Logf("downloading etcd binary from cache directory: %s", b.CacheDir)
		binary, err = installEtcd(b.CacheDir, DefaultEtcdVersion)
		if err != nil {
			b.t.Logf("etcd binary not found in PATH & cannot install: %v", err)
		}
	}

	etcd := &Etcd{
		Base:   b,
		Target: target,
		Binary: binary,
	}

	return etcd
}

func getEtcdDownloadURL(version string) (string, error) {
	switch runtime.GOOS {
	case "linux":
		switch runtime.GOARCH {
		case "amd64", "arm64":
			return fmt.Sprintf("https://github.com/etcd-io/etcd/releases/download/%s/etcd-%s-linux-%s.tar.gz", version, version, runtime.GOARCH), nil
		default:
			return "", fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
		}
	case "darwin":
		switch runtime.GOARCH {
		case "amd64", "arm64":
			return fmt.Sprintf("https://github.com/etcd-io/etcd/releases/download/%s/etcd-%s-darwin-%s.zip", version, version, runtime.GOARCH), nil
		default:
			return "", fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
		}
	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

func installEtcd(cacheDir, version string) (string, error) {
	versionDir := filepath.Join(cacheDir, version)
	binaryFile := filepath.Join(versionDir, "etcd")

	if alreadyInstalled(binaryFile) {
		return binaryFile, nil
	}

	fileInfo, err := os.Stat(versionDir) // Ensure versionDir
	if errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(versionDir, 0o700)
		if err != nil {
			return "", fmt.Errorf("failed to create version: %v, err: %w", version, err)
		}
	}
	if !fileInfo.IsDir() {
		return "", fmt.Errorf("cacheDir %s is not a directory", cacheDir)
	}

	downloadURL, err := getEtcdDownloadURL(version)
	if err != nil {
		return "", fmt.Errorf("failed to get download URL for version %s: %w", version, err)
	}

	_, filename := path.Split(downloadURL)

	downloadFilename := filepath.Join(versionDir, filename)
	err = downloadFile(downloadFilename, downloadURL)
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

func unzipWithoutWrap(src string, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

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
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
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

		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}

func decompressTarGz(filename, destDir string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

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
			err := os.MkdirAll(filepath.Dir(target), 0o755)
			if err != nil {
				return err
			}
			outFile, err := os.Create(target)
			if err != nil {
				return err
			}
			_, err = io.Copy(outFile, tr)
			if err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}
}

func downloadFile(filename, url string) error {
	out, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to request: %w", err)
	}
	defer resp.Body.Close()

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

type Etcd struct {
	Base *Base

	Target string
	Binary string

	Endpoint *string

	result *icmd.Result
}

func (e *Etcd) Name() string {
	return "etcd"
}

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

func (e *Etcd) Start() {
	e.Base.t.Helper()

	// Start etcd.
	// This is a placeholder for actual start logic.
	e.Base.Logger.Info("Starting etcd")

	icmdCmd := icmd.Command(e.Binary)
	e.result = icmd.StartCmd(icmdCmd)

	e.WaitUntilReady(e.Base.ctx)
}

func (e *Etcd) IsAlive() bool {
	// livez is a liveness endpoint of etcd sinc v3.4.29
	// ref. https://etcd.io/docs/v3.4/op-guide/monitoring/#health-check
	resp, err := http.Get(must(url.JoinPath(*e.Endpoint, "/livez")))
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func (e *Etcd) IsReady() bool {
	// readyz is a readiness endpoint of etcd since v3.4.29
	// ref. https://etcd.io/docs/v3.4/op-guide/monitoring/#health-check
	resp, err := http.Get(must(url.JoinPath(*e.Endpoint, "/readyz")))
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(fmt.Sprintf("must: %v", err))
	}
	return v
}

func (e *Etcd) WaitUntilReady(ctx context.Context) error {
	e.Base.t.Helper()

	if e.Endpoint == nil {
		return fmt.Errorf("etcd endpoint is not set")
	}

	e.Base.Logger.Info("Waiting for etcd to be ready", "endpoint", *e.Endpoint)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if e.IsReady() {
				e.Base.Logger.Info("etcd is ready")
				return nil
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}

func (e *Etcd) Stop() {
	e.Base.t.Helper()

	// Stop etcd.
	// This is a placeholder for actual stop logic.
	e.Base.Logger.Info("Stopping etcd")

	if e.result != nil && e.result.Cmd != nil && e.result.Cmd.Process != nil {
		e.result.Cmd.Process.Signal(syscall.SIGTERM)

		ctx, cancel := context.WithTimeout(e.Base.ctx, 10*time.Second)
		defer cancel()

		stopCh := make(chan struct{})
		go func() {
			defer close(stopCh)
			e.result.Cmd.Process.Wait()
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
