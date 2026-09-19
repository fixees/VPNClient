package update

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"myinternetvpn/client/internal/defaults"
	"myinternetvpn/client/internal/winutil"
)

// ApplyZip extracts a release zip next to the running executable and launches a
// hidden PowerShell helper that replaces files after this process exits.
func ApplyZip(zipPath, targetDir, exeName string) (helperPath string, err error) {
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
	exeName = strings.TrimSpace(exeName)
	if exeName == "" {
		exeName = defaults.ProductExe
	}
	exeName = filepath.Base(exeName)

	stage := filepath.Join(filepath.Dir(zipPath), "staging")
	_ = os.RemoveAll(stage)
	if err := os.MkdirAll(stage, 0o755); err != nil {
		return "", err
	}
	if err := ExtractZip(zipPath, stage); err != nil {
		return "", err
	}

	startExe := exeName
	if _, statErr := os.Stat(filepath.Join(stage, startExe)); statErr != nil {
		startExe = defaults.ProductExe
	}

	helperPath = filepath.Join(filepath.Dir(zipPath), "apply-update.ps1")
	script := buildApplyScript(os.Getpid(), stage, targetDir, startExe)
	if err := os.WriteFile(helperPath, []byte(script), 0o755); err != nil {
		return "", err
	}

	cmd := exec.Command(
		"powershell.exe",
		"-NoProfile",
		"-NonInteractive",
		"-ExecutionPolicy", "Bypass",
		"-WindowStyle", "Hidden",
		"-File", helperPath,
	)
	winutil.HideConsole(cmd)
	if err := cmd.Start(); err != nil {
		return "", err
	}
	return helperPath, nil
}

func buildApplyScript(pid int, stage, target, exeName string) string {
	var b strings.Builder
	b.WriteString("$ErrorActionPreference = 'Stop'\n")
	b.WriteString("$ProgressPreference = 'SilentlyContinue'\n")
	b.WriteString("$pidToWait = " + strconv.Itoa(pid) + "\n")
	b.WriteString("$stage = " + psQuote(stage) + "\n")
	b.WriteString("$target = " + psQuote(target) + "\n")
	b.WriteString("$exe = " + psQuote(exeName) + "\n")
	b.WriteString(`try { Wait-Process -Id $pidToWait -ErrorAction SilentlyContinue } catch {}
Start-Sleep -Milliseconds 400
if (-not (Test-Path -LiteralPath $stage)) { exit 1 }
New-Item -ItemType Directory -Force -Path $target | Out-Null
Copy-Item -LiteralPath (Join-Path $stage '*') -Destination $target -Recurse -Force
$launch = Join-Path $target $exe
if (Test-Path -LiteralPath $launch) {
  Start-Process -FilePath $launch
}
`)
	return b.String()
}

func psQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
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
