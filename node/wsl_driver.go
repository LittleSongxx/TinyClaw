package node

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"os/exec"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"
)

type WSLDriverConfig struct {
	DistroName string
	DefaultCWD string
}

type WSLDriver struct {
	distroName string
	defaultCWD string
}

func NewWSLDriver(cfg WSLDriverConfig) *WSLDriver {
	return &WSLDriver{
		distroName: strings.TrimSpace(cfg.DistroName),
		defaultCWD: strings.TrimSpace(cfg.DefaultCWD),
	}
}

func (d *WSLDriver) Capabilities() []NodeCapability {
	return []NodeCapability{
		{Name: "wsl.exec", Category: "wsl", Description: "Execute a shell command inside the configured WSL distro"},
		{Name: "wsl.fs.list", Category: "wsl", Description: "List files in a directory inside the configured WSL distro"},
		{Name: "wsl.fs.read", Category: "wsl", Description: "Read a file from the configured WSL distro"},
		{Name: "wsl.fs.write", Category: "wsl", Description: "Write a file inside the configured WSL distro"},
	}
}

func (d *WSLDriver) Execute(ctx context.Context, req NodeCommandRequest) (*NodeCommandResult, error) {
	startedAt := time.Now().Unix()
	result := &NodeCommandResult{
		ID:         req.ID,
		NodeID:     req.NodeID,
		Capability: req.Capability,
		StartedAt:  startedAt,
		Success:    false,
	}

	var err error
	switch req.Capability {
	case "wsl.exec":
		result, err = d.execCommand(ctx, req, startedAt)
	case "wsl.fs.list":
		result, err = d.listFiles(ctx, req, startedAt)
	case "wsl.fs.read":
		result, err = d.readFile(ctx, req, startedAt)
	case "wsl.fs.write":
		result, err = d.writeFile(ctx, req, startedAt)
	default:
		err = errors.New("unsupported wsl node capability")
	}

	if err != nil {
		if result == nil {
			result = &NodeCommandResult{
				ID:         req.ID,
				NodeID:     req.NodeID,
				Capability: req.Capability,
				StartedAt:  startedAt,
			}
		}
		result.Success = false
		result.Error = err.Error()
		result.CompletedAt = time.Now().Unix()
		return result, nil
	}

	result.Success = true
	if result.CompletedAt == 0 {
		result.CompletedAt = time.Now().Unix()
	}
	return result, nil
}

func (d *WSLDriver) execCommand(ctx context.Context, req NodeCommandRequest, startedAt int64) (*NodeCommandResult, error) {
	command := strings.TrimSpace(stringArg(req.Arguments, "command"))
	if command == "" {
		return nil, errors.New("wsl.exec requires command")
	}
	args := normalizeStringSlice(stringSliceArg(req.Arguments, "args"))
	envMap := mapArg(req.Arguments, "env")
	commandScript, err := buildWSLCommandScript(command, args, envMap)
	if err != nil {
		return nil, err
	}

	runCtx, cancel := wslTimeoutContext(ctx, req.TimeoutSec)
	defer cancel()

	workingDir := stringArg(req.Arguments, "cwd")
	stdout, stderr, effectiveCWD, err := d.runWSLScript(runCtx, workingDir, commandScript, nil)
	combinedOutput := append(append([]byte{}, stdout...), stderr...)

	return &NodeCommandResult{
		ID:          req.ID,
		NodeID:      req.NodeID,
		Capability:  req.Capability,
		Output:      decodeWSLText(combinedOutput),
		StartedAt:   startedAt,
		CompletedAt: time.Now().Unix(),
		Data: map[string]interface{}{
			"command": command,
			"args":    args,
			"cwd":     effectiveCWD,
			"distro":  d.distroName,
		},
	}, formatCommandError(err, stderr)
}

func (d *WSLDriver) listFiles(ctx context.Context, req NodeCommandRequest, startedAt int64) (*NodeCommandResult, error) {
	target := normalizeWSLPath(stringArg(req.Arguments, "path"), ".")
	runCtx, cancel := wslTimeoutContext(ctx, req.TimeoutSec)
	defer cancel()

	script := `set -euo pipefail
target=` + shellSingleQuote(target) + `
if [ ! -d "$target" ]; then
  echo "directory not found: $target" >&2
  exit 1
fi
find "$target" -mindepth 1 -maxdepth 1 -printf '%f\0%y\0%s\0%T@\0'
`
	stdout, stderr, effectiveCWD, err := d.runWSLScript(runCtx, "", script, nil)
	if err != nil {
		return nil, formatCommandError(err, stderr)
	}

	entries, err := parseWSLListOutput(stdout)
	if err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		left, _ := entries[i]["name"].(string)
		right, _ := entries[j]["name"].(string)
		return left < right
	})

	return &NodeCommandResult{
		ID:          req.ID,
		NodeID:      req.NodeID,
		Capability:  req.Capability,
		StartedAt:   startedAt,
		CompletedAt: time.Now().Unix(),
		Data: map[string]interface{}{
			"path":    target,
			"entries": entries,
			"cwd":     effectiveCWD,
			"distro":  d.distroName,
		},
	}, nil
}

func (d *WSLDriver) readFile(ctx context.Context, req NodeCommandRequest, startedAt int64) (*NodeCommandResult, error) {
	target := strings.TrimSpace(stringArg(req.Arguments, "path"))
	if target == "" {
		return nil, errors.New("wsl.fs.read requires path")
	}
	target = normalizeWSLPath(target, "")

	runCtx, cancel := wslTimeoutContext(ctx, req.TimeoutSec)
	defer cancel()

	script := `set -euo pipefail
target=` + shellSingleQuote(target) + `
if [ ! -f "$target" ]; then
  echo "file not found: $target" >&2
  exit 1
fi
cat -- "$target"
`
	stdout, stderr, effectiveCWD, err := d.runWSLScript(runCtx, "", script, nil)
	if err != nil {
		return nil, formatCommandError(err, stderr)
	}

	result := &NodeCommandResult{
		ID:          req.ID,
		NodeID:      req.NodeID,
		Capability:  req.Capability,
		StartedAt:   startedAt,
		CompletedAt: time.Now().Unix(),
		Data: map[string]interface{}{
			"path":   target,
			"size":   len(stdout),
			"cwd":    effectiveCWD,
			"distro": d.distroName,
		},
	}
	if utf8.Valid(stdout) {
		result.Output = string(stdout)
	} else {
		result.Data["base64"] = base64.StdEncoding.EncodeToString(stdout)
	}
	return result, nil
}

func (d *WSLDriver) writeFile(ctx context.Context, req NodeCommandRequest, startedAt int64) (*NodeCommandResult, error) {
	target := strings.TrimSpace(stringArg(req.Arguments, "path"))
	if target == "" {
		return nil, errors.New("wsl.fs.write requires path")
	}
	target = normalizeWSLPath(target, "")
	appendMode := boolArg(req.Arguments, "append")
	encoding := stringArg(req.Arguments, "encoding")
	content := stringArg(req.Arguments, "content")

	payload := ""
	writtenSize := 0
	if encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(content)
		if err != nil {
			return nil, fmt.Errorf("decode base64 content: %w", err)
		}
		payload = base64.StdEncoding.EncodeToString(decoded)
		writtenSize = len(decoded)
	} else {
		payload = base64.StdEncoding.EncodeToString([]byte(content))
		writtenSize = len([]byte(content))
	}

	writeTarget := ">"
	if appendMode {
		writeTarget = ">>"
	}
	script := `set -euo pipefail
target=` + shellSingleQuote(target) + `
payload=` + shellSingleQuote(payload) + `
mkdir -p -- "$(dirname -- "$target")"
printf '%s' "$payload" | base64 -d ` + writeTarget + ` "$target"
`

	runCtx, cancel := wslTimeoutContext(ctx, req.TimeoutSec)
	defer cancel()

	_, stderr, effectiveCWD, err := d.runWSLScript(runCtx, "", script, nil)
	if err != nil {
		return nil, formatCommandError(err, stderr)
	}

	return &NodeCommandResult{
		ID:          req.ID,
		NodeID:      req.NodeID,
		Capability:  req.Capability,
		StartedAt:   startedAt,
		CompletedAt: time.Now().Unix(),
		Data: map[string]interface{}{
			"path":     target,
			"append":   appendMode,
			"size":     writtenSize,
			"encoding": encoding,
			"cwd":      effectiveCWD,
			"distro":   d.distroName,
		},
	}, nil
}

func ListWSLDistros(ctx context.Context) ([]string, error) {
	executable, err := wslExecutable()
	if err != nil {
		return nil, err
	}
	output, err := exec.CommandContext(ctx, executable, "-l", "-q").CombinedOutput()
	if err != nil {
		return nil, formatCommandError(err, output)
	}

	text := strings.ReplaceAll(decodeWSLText(output), "\u0000", "")
	lines := strings.Split(text, "\n")
	distros := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		distros = append(distros, line)
	}
	return distros, nil
}

func wslExecutable() (string, error) {
	candidates := []string{"wsl.exe", "wsl"}
	for _, candidate := range candidates {
		path, err := exec.LookPath(candidate)
		if err == nil {
			return path, nil
		}
	}
	return "", errors.New("wsl.exe is not available on this host")
}

func wslTimeoutContext(ctx context.Context, timeoutSec int) (context.Context, context.CancelFunc) {
	if timeoutSec <= 0 {
		timeoutSec = 30
	}
	return context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
}

func (d *WSLDriver) runWSLScript(ctx context.Context, cwd, script string, stdin []byte) ([]byte, []byte, string, error) {
	executable, err := wslExecutable()
	if err != nil {
		return nil, nil, "", err
	}

	effectiveCWD := strings.TrimSpace(cwd)
	if effectiveCWD == "" {
		effectiveCWD = strings.TrimSpace(d.defaultCWD)
	}

	commandArgs := make([]string, 0, 8)
	if d.distroName != "" {
		commandArgs = append(commandArgs, "-d", d.distroName)
	}
	if effectiveCWD != "" {
		commandArgs = append(commandArgs, "--cd", effectiveCWD)
	}
	commandArgs = append(commandArgs, "--exec", "bash", "-lc", script)

	cmd := exec.CommandContext(ctx, executable, commandArgs...)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), effectiveCWD, err
}

func buildWSLCommandScript(command string, args []string, env map[string]string) (string, error) {
	var builder strings.Builder
	builder.WriteString("set -euo pipefail\n")
	if len(env) > 0 {
		keys := make([]string, 0, len(env))
		for key := range env {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if !isValidShellEnvKey(key) {
				return "", fmt.Errorf("invalid environment variable name: %s", key)
			}
			builder.WriteString("export ")
			builder.WriteString(key)
			builder.WriteString("=")
			builder.WriteString(shellSingleQuote(env[key]))
			builder.WriteString("\n")
		}
	}
	builder.WriteString("exec ")
	builder.WriteString(shellSingleQuote(command))
	for _, arg := range args {
		builder.WriteString(" ")
		builder.WriteString(shellSingleQuote(arg))
	}
	return builder.String(), nil
}

func isValidShellEnvKey(key string) bool {
	if key == "" {
		return false
	}
	for i, char := range key {
		if i == 0 {
			if (char < 'A' || char > 'Z') && (char < 'a' || char > 'z') && char != '_' {
				return false
			}
			continue
		}
		if (char < 'A' || char > 'Z') && (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '_' {
			return false
		}
	}
	return true
}

func shellSingleQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func normalizeWSLPath(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		trimmed = fallback
	}
	if trimmed == "" {
		return ""
	}
	return path.Clean(trimmed)
}

func parseWSLListOutput(raw []byte) ([]map[string]interface{}, error) {
	trimmed := bytes.TrimSuffix(raw, []byte{0})
	if len(trimmed) == 0 {
		return nil, nil
	}

	fields := bytes.Split(trimmed, []byte{0})
	if len(fields)%4 != 0 {
		return nil, errors.New("unexpected wsl fs.list response shape")
	}

	items := make([]map[string]interface{}, 0, len(fields)/4)
	for index := 0; index < len(fields); index += 4 {
		size, err := strconv.ParseInt(string(fields[index+2]), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse wsl entry size: %w", err)
		}
		modtimeFloat, err := strconv.ParseFloat(string(fields[index+3]), 64)
		if err != nil {
			return nil, fmt.Errorf("parse wsl entry modtime: %w", err)
		}
		items = append(items, map[string]interface{}{
			"name":    string(fields[index]),
			"is_dir":  string(fields[index+1]) == "d",
			"size":    size,
			"modtime": int64(modtimeFloat),
		})
	}
	return items, nil
}

func decodeWSLText(raw []byte) string {
	raw = bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF})
	if utf8.Valid(raw) && !bytes.Contains(raw, []byte{0}) {
		return string(raw)
	}
	if len(raw)%2 != 0 {
		return string(raw)
	}

	zeroHighBytes := 0
	units := make([]uint16, len(raw)/2)
	for index := 0; index < len(raw); index += 2 {
		units[index/2] = binary.LittleEndian.Uint16(raw[index : index+2])
		if raw[index+1] == 0 {
			zeroHighBytes++
		}
	}
	if zeroHighBytes < len(units)/4 {
		return string(raw)
	}
	return string(utf16.Decode(units))
}
