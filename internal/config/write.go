package config

import (
	"fmt"
	"os"
	"strings"
)

func FindSection(lines []string, header string) (start, end int, found bool) {
	target := "[" + header + "]"
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == target {
			start = i
			for j := i + 1; j < len(lines); j++ {
				trimmed := strings.TrimSpace(lines[j])
				if strings.HasPrefix(trimmed, "[") {
					return start, j, true
				}
			}
			return start, len(lines), true
		}
	}
	return 0, 0, false
}

func AppendProject(path string, id string, project Project) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	content := string(data)
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	section := formatProjectSection(id, project)
	content += "\n" + section

	return os.WriteFile(path, []byte(content), 0644)
}

func RemoveProject(path string, id string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	header := "projects." + id
	start, end, found := FindSection(lines, header)
	if !found {
		return fmt.Errorf("project %q not found in config", id)
	}

	// Remove trailing blank lines before the next section
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	// Also remove one blank line before the section if present
	if start > 0 && strings.TrimSpace(lines[start-1]) == "" {
		start--
	}

	newLines := append(lines[:start], lines[end:]...)
	return os.WriteFile(path, []byte(strings.Join(newLines, "\n")), 0644)
}

func UpdateProject(path string, id string, updates map[string]string, unsets []string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	header := "projects." + id
	start, end, found := FindSection(lines, header)
	if !found {
		return fmt.Errorf("project %q not found in config", id)
	}

	lines = applyProjectUpdates(lines, start, end, updates, unsets)

	return os.WriteFile(path, []byte(strings.Join(collapseBlankLines(lines), "\n")), 0644)
}

func applyProjectUpdates(lines []string, start, end int, updates map[string]string, unsets []string) []string {
	pendingUpdates := cloneStringMap(updates)
	unsetSet := stringSet(unsets)

	lines = updateExistingProjectKeys(lines, start, end, pendingUpdates, unsetSet)
	return insertMissingProjectKeys(lines, start, end, pendingUpdates)
}

func updateExistingProjectKeys(lines []string, start, end int, updates map[string]string, unsets map[string]bool) []string {
	for i := start + 1; i < end; i++ {
		key := extractKey(lines[i])
		if key == "" {
			continue
		}
		if unsets[key] {
			lines[i] = ""
			continue
		}
		if val, ok := updates[key]; ok {
			lines[i] = formatKeyValue(key, val)
			delete(updates, key)
		}
	}
	return lines
}

func insertMissingProjectKeys(lines []string, start, end int, updates map[string]string) []string {
	var newLines []string
	for key, val := range updates {
		newLines = append(newLines, formatKeyValue(key, val))
	}
	if len(newLines) == 0 {
		return lines
	}

	insertAt := end
	for insertAt > start+1 && strings.TrimSpace(lines[insertAt-1]) == "" {
		insertAt--
	}
	result := make([]string, 0, len(lines)+len(newLines))
	result = append(result, lines[:insertAt]...)
	result = append(result, newLines...)
	result = append(result, lines[insertAt:]...)
	return result
}

func cloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func stringSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}

func collapseBlankLines(lines []string) []string {
	var cleaned []string
	prevBlank := false
	for _, line := range lines {
		blank := strings.TrimSpace(line) == ""
		if blank && prevBlank {
			continue
		}
		cleaned = append(cleaned, line)
		prevBlank = blank
	}
	return cleaned
}

func RenameSection(path string, oldID string, newID string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	oldHeader := "[projects." + oldID + "]"
	newHeader := "[projects." + newID + "]"

	lines := strings.Split(string(data), "\n")
	found := false
	for i, line := range lines {
		if strings.TrimSpace(line) == oldHeader {
			lines[i] = newHeader
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("project %q not found in config", oldID)
	}

	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
}

func formatProjectSection(id string, project Project) string {
	var b strings.Builder
	fmt.Fprintf(&b, "[projects.%s]\n", id)
	if project.Name != "" {
		fmt.Fprintf(&b, "name = %q\n", project.Name)
	}
	for _, s := range project.Sources() {
		writeStringList(&b, s.Name, s.Values)
	}
	return b.String()
}

func writeStringList(b *strings.Builder, key string, values StringList) {
	if len(values) == 0 {
		return
	}
	if len(values) == 1 {
		fmt.Fprintf(b, "%s = %q\n", key, values[0])
		return
	}
	fmt.Fprintf(b, "%s = [", key)
	for i, v := range values {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(b, "%q", v)
	}
	b.WriteString("]\n")
}

func formatKeyValue(key, val string) string {
	if key == "name" || !strings.Contains(val, ",") {
		return fmt.Sprintf("%s = %q", key, val)
	}
	var trimmed []string
	for _, p := range strings.Split(val, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			trimmed = append(trimmed, p)
		}
	}
	if len(trimmed) <= 1 {
		v := ""
		if len(trimmed) == 1 {
			v = trimmed[0]
		}
		return fmt.Sprintf("%s = %q", key, v)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s = [", key)
	for i, v := range trimmed {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%q", v)
	}
	b.WriteString("]")
	return b.String()
}

func extractKey(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "[") {
		return ""
	}
	parts := strings.SplitN(trimmed, "=", 2)
	if len(parts) != 2 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}
