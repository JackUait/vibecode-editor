package models

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Project represents a project entry
type Project struct {
	Name string
	Path string
}

// LoadProjects reads projects from file (name:path format)
func LoadProjects(filepath string) ([]Project, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open projects file: %w", err)
	}
	defer file.Close()

	var projects []Project
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse name:path format
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue // Skip malformed lines
		}

		projects = append(projects, Project{
			Name: strings.TrimSpace(parts[0]),
			Path: strings.TrimSpace(parts[1]),
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read projects file: %w", err)
	}

	return projects, nil
}
