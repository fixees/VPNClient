package update

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"myinternetvpn/client/internal/defaults"
)

// ApplyZip extracts a release zip next to the running executable and launches a
// Windows helper script that replaces files after the process exits.
func ApplyZip(zipPath, targetDir string) (helperPath string, err error) {
	if runtime.GOOS != "windows" {
		return "", fmt.Errorf("apply update is only implemented for Windows")
	}
	zipPath, err = filepath.Abs(zipPath)
	if err != nil {
		return "", err
	}
	targetDir, err = filepath.Abs(targetDir)
	if err != nil {
		return "", err
	}
	stage := filepath.Join(filepath.Dir(zipPath), "staging")
	_ = os.RemoveAll(stage)
	if err := os.MkdirAll(stage, 0o755); err != nil {
		return "", err
	}
	if err := ExtractZip(zipPath, stage); err != nil {
		return "", err
	}

	pid := os.Getpid()
	helperPath = filepath.Join(filepath.Dir(zipPath), "apply-update.bat")
	script := fmt.Sprintf(`@echo off
setlocal
set PID=%d
set STAGE=%s
set TARGET=%s
:wait
tasklist /FI "PID eq %%PID%%" | find "%d" >nul
if not errorlevel 1 (
  timeout /t 1 /nobreak >nul
  goto wait
)
xcopy /E /Y /I "%%STAGE%%\*" "%%TARGET%%\" >nul
start "" "%%TARGET%%\%s"
`, pid, stage, targetDir, pid, defaults.ProductExe)

	if err := os.WriteFile(helperPath, []byte(script), 0o755); err != nil {
		return "", err
	}
	cmd := exec.Command("cmd", "/C", "start", "", helperPath)
	if err := cmd.Start(); err != nil {
		return "", err
	}
	return helperPath, nil
}

func ExtractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		name := filepath.Clean(f.Name)
		if strings.Contains(name, "..") {
			return fmt.Errorf("invalid zip entry %q", f.Name)
		}
		path := filepath.Join(dest, name)
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) && path != filepath.Clean(dest) {
			return fmt.Errorf("zip entry escapes target: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(path, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			rc.Close()
			return err
		}
		_, copyErr := io.Copy(out, rc)
		closeErr := out.Close()
		rc.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}
