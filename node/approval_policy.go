package node

import (
	"encoding/json"
	"path"
	"strings"
)

const (
	metadataKindKey                   = "kind"
	metadataKindWindows               = "windows"
	metadataKindWSL                   = "wsl"
	metadataParentNodeIDKey           = "parent_node_id"
	metadataWSLDistroKey              = "wsl_distro"
	metadataAllowCommandPrefixesKey   = "approval_allow_command_prefixes"
	metadataAllowWritePathPrefixesKey = "approval_allow_write_path_prefixes"
)

func encodeMetadataStringSlice(values []string) string {
	if len(values) == 0 {
		return ""
	}
	content, err := json.Marshal(values)
	if err != nil {
		return ""
	}
	return string(content)
}

func decodeMetadataStringSlice(metadata map[string]string, key string) []string {
	if len(metadata) == 0 {
		return nil
	}
	raw := strings.TrimSpace(metadata[key])
	if raw == "" {
		return nil
	}

	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err == nil {
		out := make([]string, 0, len(values))
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value != "" {
				out = append(out, value)
			}
		}
		return out
	}

	lines := strings.Split(raw, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func normalizeApprovalCommand(command string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(command)), " ")
}

func normalizeApprovalPath(pathText string) string {
	pathText = strings.TrimSpace(pathText)
	if pathText == "" {
		return ""
	}
	return path.Clean(pathText)
}

func shouldRequireNodeApproval(desc NodeDescriptor, req NodeCommandRequest) bool {
	if !req.RequireApproval {
		return false
	}
	if desc.Metadata[metadataKindKey] != metadataKindWSL {
		return true
	}

	switch req.Capability {
	case "wsl.exec":
		command := normalizeApprovalCommand(buildApprovalCommandLine(req.Arguments))
		for _, prefix := range decodeMetadataStringSlice(desc.Metadata, metadataAllowCommandPrefixesKey) {
			if strings.HasPrefix(command, prefix) {
				return false
			}
		}
	case "wsl.fs.write":
		target := normalizeApprovalPath(stringArg(req.Arguments, "path"))
		for _, prefix := range decodeMetadataStringSlice(desc.Metadata, metadataAllowWritePathPrefixesKey) {
			if strings.HasPrefix(target, prefix) {
				return false
			}
		}
	}
	return true
}

func buildApprovalCommandLine(arguments map[string]interface{}) string {
	command := strings.TrimSpace(stringArg(arguments, "command"))
	args := normalizeStringSlice(stringSliceArg(arguments, "args"))
	if command == "" {
		return ""
	}
	if len(args) == 0 {
		return command
	}
	return command + " " + strings.Join(args, " ")
}

func wslDistroName(desc NodeDescriptor) string {
	if len(desc.Metadata) == 0 {
		return ""
	}
	return strings.TrimSpace(desc.Metadata[metadataWSLDistroKey])
}
