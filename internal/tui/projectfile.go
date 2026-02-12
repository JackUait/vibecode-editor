package tui

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackuait/ghost-tab/internal/models"
)

// AppendProject appends a name:path entry to the projects file.
func AppendProject(name, path, filePath string) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(name + ":" + path + "\n")
	return err
}

// RemoveProject removes an exact line from the projects file.
func RemoveProject(line, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var kept []string
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		if scanner.Text() != line {
			kept = append(kept, scanner.Text())
		}
	}

	var result string
	if len(kept) > 0 {
		result = strings.Join(kept, "\n") + "\n"
	}
	return os.WriteFile(filePath, []byte(result), 0644)
}

// IsDuplicateProject checks if an expanded path already exists in the project list.
func IsDuplicateProject(expandedPath string, projects []models.Project) bool {
	cleaned := strings.TrimRight(expandedPath, "/")
	for _, p := range projects {
		if strings.TrimRight(p.Path, "/") == cleaned {
			return true
		}
	}
	return false
}
