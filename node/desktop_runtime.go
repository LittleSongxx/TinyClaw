package node

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

var (
	windowsReadableTempDirOnce sync.Once
	windowsReadableTempDirPath string
	windowsReadableTempDirErr  error
)

func nodeCapabilitiesForRuntime() []NodeCapability {
	capabilities := []NodeCapability{
		{Name: "system.exec", Category: "system", Description: "Execute a command on the paired PC node"},
		{Name: "fs.list", Category: "fs", Description: "List files in a directory on the paired PC node"},
		{Name: "fs.read", Category: "fs", Description: "Read a file from the paired PC node"},
		{Name: "fs.write", Category: "fs", Description: "Write a file on the paired PC node"},
		{Name: "screen.snapshot", Category: "screen", Description: "Capture a screenshot on the paired PC node"},
		{Name: "browser.open", Category: "browser", Description: "Open a URL with the default browser"},
		{Name: "app.launch", Category: "app", Description: "Launch an application on the paired PC node"},
	}
	if supportsWindowsDesktopAutomation() {
		capabilities = append(capabilities,
			NodeCapability{Name: "input.keyboard.type", Category: "input", Description: "Type text into the active window on the paired PC node"},
			NodeCapability{Name: "input.keyboard.key", Category: "input", Description: "Press a key on the paired PC node"},
			NodeCapability{Name: "input.keyboard.hotkey", Category: "input", Description: "Trigger a hotkey combination on the paired PC node"},
			NodeCapability{Name: "input.mouse.move", Category: "input", Description: "Move the mouse cursor on the paired PC node"},
			NodeCapability{Name: "input.mouse.click", Category: "input", Description: "Click the mouse on the paired PC node"},
			NodeCapability{Name: "input.mouse.double_click", Category: "input", Description: "Double click the mouse on the paired PC node"},
			NodeCapability{Name: "input.mouse.right_click", Category: "input", Description: "Right click the mouse on the paired PC node"},
			NodeCapability{Name: "input.mouse.drag", Category: "input", Description: "Drag the mouse on the paired PC node"},
			NodeCapability{Name: "window.list", Category: "window", Description: "List desktop windows on the paired PC node"},
			NodeCapability{Name: "window.focus", Category: "window", Description: "Focus a desktop window on the paired PC node"},
			NodeCapability{Name: "ui.inspect", Category: "ui", Description: "Inspect the currently focused desktop UI element or a point on screen"},
			NodeCapability{Name: "ui.find", Category: "ui", Description: "Find desktop UI elements inside the current window on the paired PC node"},
			NodeCapability{Name: "ui.focus", Category: "ui", Description: "Focus a desktop UI element on the paired PC node"},
		)
	}
	return capabilities
}

func supportsWindowsDesktopAutomation() bool {
	return runtime.GOOS == "windows" || isWSLRuntime()
}

func isWSLRuntime() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	if os.Getenv("WSL_DISTRO_NAME") != "" || os.Getenv("WSL_INTEROP") != "" {
		return true
	}
	release, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(release)), "microsoft")
}

func powerShellExecutable() (string, error) {
	candidates := []string{"powershell", "powershell.exe", "pwsh", "pwsh.exe"}
	if isWSLRuntime() {
		candidates = []string{"powershell.exe", "pwsh.exe", "powershell", "pwsh"}
	}
	for _, candidate := range candidates {
		path, err := exec.LookPath(candidate)
		if err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("powershell executable is not available on %s", runtime.GOOS)
}

func desktopTempFilePath(name string) (string, string, error) {
	if !isWSLRuntime() {
		path := filepath.Join(os.TempDir(), name)
		return path, path, nil
	}
	tempDir, err := windowsReadableTempDir()
	if err != nil {
		return "", "", err
	}
	localPath := filepath.Join(tempDir, name)
	windowsPath, err := wslPath("w", localPath)
	if err != nil {
		return "", "", err
	}
	return localPath, windowsPath, nil
}

func windowsReadableTempDir() (string, error) {
	windowsReadableTempDirOnce.Do(func() {
		tempDir, err := queryWindowsTempDir()
		if err == nil {
			windowsReadableTempDirPath, err = wslPath("u", tempDir)
		}
		if err != nil {
			fallback := "/mnt/c/Windows/Temp"
			if statErr := os.MkdirAll(fallback, 0755); statErr == nil {
				windowsReadableTempDirPath = fallback
				windowsReadableTempDirErr = nil
				return
			}
			windowsReadableTempDirErr = err
			return
		}
		windowsReadableTempDirErr = os.MkdirAll(windowsReadableTempDirPath, 0755)
	})
	return windowsReadableTempDirPath, windowsReadableTempDirErr
}

func queryWindowsTempDir() (string, error) {
	executable, err := powerShellExecutable()
	if err != nil {
		return "", err
	}
	output, err := exec.Command(executable, "-NoProfile", "-Command", "[System.IO.Path]::GetTempPath()").CombinedOutput()
	if err != nil {
		return "", formatCommandError(err, output)
	}
	tempDir := strings.TrimSpace(string(output))
	if tempDir == "" {
		return "", fmt.Errorf("powershell returned empty temp directory")
	}
	return tempDir, nil
}

func wslPath(mode, path string) (string, error) {
	output, err := exec.Command("wslpath", "-"+mode, path).CombinedOutput()
	if err != nil {
		return "", formatCommandError(err, output)
	}
	converted := strings.TrimSpace(string(output))
	if converted == "" {
		return "", fmt.Errorf("wslpath returned empty path for %s", path)
	}
	return converted, nil
}

func formatCommandError(err error, output []byte) error {
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return err
	}
	return fmt.Errorf("%w: %s", err, trimmed)
}
