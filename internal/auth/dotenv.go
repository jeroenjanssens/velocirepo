package auth

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func UpsertEnvFile(path string, updates map[string]string) error {
	lines, err := readLines(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", path, err)
	}

	remaining := make(map[string]bool, len(updates))
	for k := range updates {
		remaining[k] = true
	}

	var result []string
	for _, line := range lines {
		key := envKey(line)
		if key != "" && remaining[key] {
			result = append(result, key+"="+quoteEnvValue(updates[key]))
			delete(remaining, key)
		} else {
			result = append(result, line)
		}
	}

	for key, val := range updates {
		if remaining[key] {
			result = append(result, key+"="+quoteEnvValue(val))
		} else {
			_ = val
		}
	}

	content := strings.Join(result, "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	return os.WriteFile(path, []byte(content), 0600)
}

func ReadEnvValue(path, key string) string {
	lines, err := readLines(path)
	if err != nil {
		return ""
	}
	for _, line := range lines {
		k := envKey(line)
		if k == key {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				return unquoteEnvValue(parts[1])
			}
		}
	}
	return ""
}

func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func envKey(line string) string {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return ""
	}
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}

func quoteEnvValue(val string) string {
	if strings.ContainsAny(val, " \t#=\"'\\$`!") {
		r := strings.NewReplacer(
			"\\", "\\\\",
			"\"", "\\\"",
			"$", "\\$",
			"`", "\\`",
			"!", "\\!",
		)
		return "\"" + r.Replace(val) + "\""
	}
	return val
}

func unquoteEnvValue(val string) string {
	if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
		val = val[1 : len(val)-1]
		r := strings.NewReplacer(
			"\\\"", "\"",
			"\\$", "$",
			"\\`", "`",
			"\\!", "!",
			"\\\\", "\\",
		)
		return r.Replace(val)
	}
	return val
}
