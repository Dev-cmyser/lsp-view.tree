package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

type ProjectData struct {
	Components          map[string]bool            // Set of component names
	ComponentProperties map[string]map[string]bool // Map of component -> properties
	ComponentFiles      map[string]string          // Map of component -> file path
	FileComponents      map[string]map[string]bool // Map of file path -> components
	mutex               sync.RWMutex
}

func NewProjectData() *ProjectData {
	return &ProjectData{
		Components:          make(map[string]bool),
		ComponentProperties: make(map[string]map[string]bool),
		ComponentFiles:      make(map[string]string),
		FileComponents:      make(map[string]map[string]bool),
	}
}

type ProjectScanner struct {
	workspaceRoot string
	projectData   *ProjectData
}

func NewProjectScanner(workspaceRoot string) *ProjectScanner {
	return &ProjectScanner{
		workspaceRoot: workspaceRoot,
		projectData:   NewProjectData(),
	}
}

func (ps *ProjectScanner) ScanProject() error {
	log.Println("[view.tree] Starting project scan...")
	
	// Reset project data
	ps.projectData = NewProjectData()
	
	// Scan .view.tree files
	if err := ps.scanViewTreeFiles(); err != nil {
		log.Printf("[view.tree] Error scanning view.tree files: %v", err)
	}
	
	// Scan .ts files
	if err := ps.scanTsFiles(); err != nil {
		log.Printf("[view.tree] Error scanning ts files: %v", err)
	}
	
	ps.projectData.mutex.RLock()
	componentCount := len(ps.projectData.Components)
	propertiesCount := len(ps.projectData.ComponentProperties)
	ps.projectData.mutex.RUnlock()
	
	log.Printf("[view.tree] Scan complete: %d components, %d components with properties", componentCount, propertiesCount)
	
	var componentNames []string
	ps.projectData.mutex.RLock()
	for component := range ps.projectData.Components {
		componentNames = append(componentNames, component)
	}
	ps.projectData.mutex.RUnlock()
	
	if len(componentNames) > 0 {
		sort.Strings(componentNames)
		log.Printf("[view.tree] Components found: %s", strings.Join(componentNames, ", "))
	}
	
	return nil
}

func (ps *ProjectScanner) scanViewTreeFiles() error {
	viewTreeFiles, err := ps.findFiles("**/*.view.tree")
	if err != nil {
		return fmt.Errorf("failed to find view.tree files: %w", err)
	}
	
	log.Printf("[view.tree] Found %d .view.tree files", len(viewTreeFiles))
	
	for _, filePath := range viewTreeFiles {
		content, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("[view.tree] Error reading %s: %v", filePath, err)
			continue
		}
		
		ps.parseViewTreeFile(string(content), filePath)
	}
	
	return nil
}

func (ps *ProjectScanner) scanTsFiles() error {
	tsFiles, err := ps.findFiles("**/*.ts")
	if err != nil {
		return fmt.Errorf("failed to find ts files: %w", err)
	}
	
	log.Printf("[view.tree] Found %d .ts files", len(tsFiles))
	
	// Limit to first 100 files for performance
	if len(tsFiles) > 100 {
		tsFiles = tsFiles[:100]
	}
	
	for _, filePath := range tsFiles {
		content, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("[view.tree] Error reading %s: %v", filePath, err)
			continue
		}
		
		ps.parseTsFile(string(content), filePath)
	}
	
	return nil
}

func (ps *ProjectScanner) findFiles(pattern string) ([]string, error) {
	var files []string
	
	err := filepath.WalkDir(ps.workspaceRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors and continue
		}
		
		if d.IsDir() {
			// Skip hidden directories and node_modules
			if strings.HasPrefix(d.Name(), ".") || d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		
		if strings.Contains(pattern, "*.view.tree") && strings.HasSuffix(path, ".view.tree") {
			files = append(files, path)
		} else if strings.Contains(pattern, "*.ts") && strings.HasSuffix(path, ".ts") && !strings.HasSuffix(path, ".d.ts") {
			files = append(files, path)
		}
		
		return nil
	})
	
	return files, err
}

func (ps *ProjectScanner) parseViewTreeFile(content, filePath string) {
	lines := strings.Split(content, "\n")
	var currentComponent string
	
	ps.projectData.mutex.Lock()
	defer ps.projectData.mutex.Unlock()
	
	// Clear previous components for this file
	if components, exists := ps.projectData.FileComponents[filePath]; exists {
		for comp := range components {
			if ps.projectData.ComponentFiles[comp] == filePath {
				delete(ps.projectData.ComponentFiles, comp)
			}
		}
	}
	ps.projectData.FileComponents[filePath] = make(map[string]bool)
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Take only the first word from lines without indentation
		if !strings.HasPrefix(line, "\t") && !strings.HasPrefix(line, " ") && strings.HasPrefix(trimmed, "$") {
			fields := strings.Fields(trimmed)
			if len(fields) > 0 {
				firstWord := fields[0]
				if strings.HasPrefix(firstWord, "$") {
					currentComponent = firstWord
					ps.projectData.Components[firstWord] = true
					ps.projectData.ComponentFiles[firstWord] = filePath
					ps.projectData.FileComponents[filePath][firstWord] = true
					
					if _, exists := ps.projectData.ComponentProperties[firstWord]; !exists {
						ps.projectData.ComponentProperties[firstWord] = make(map[string]bool)
					}
				}
			}
		}
		
		// Look for properties (indented lines without <= and <=>)
		if currentComponent != "" {
			indentMatch := regexp.MustCompile(`^(\s+)([a-zA-Z_][a-zA-Z0-9_?*]*)\s*`).FindStringSubmatch(line)
			if len(indentMatch) > 2 && len(indentMatch[1]) > 0 && 
			   !strings.Contains(trimmed, "<=") && !strings.Contains(trimmed, "<=>") {
				property := indentMatch[2]
				if property != "" && !strings.HasPrefix(property, "$") && 
				   property != "null" && property != "true" && property != "false" {
					ps.projectData.ComponentProperties[currentComponent][property] = true
				}
			}
			
			// Look for properties in bindings: <= PropertyName
			bindingMatch := regexp.MustCompile(`<=\s+([a-zA-Z_][a-zA-Z0-9_?*]*)`).FindStringSubmatch(trimmed)
			if len(bindingMatch) > 1 {
				property := bindingMatch[1]
				if property != "" && !strings.HasPrefix(property, "$") {
					ps.projectData.ComponentProperties[currentComponent][property] = true
				}
			}
		}
	}
}

func (ps *ProjectScanner) parseTsFile(content, filePath string) {
	// Look for all $ components in TypeScript files
	componentRegex := regexp.MustCompile(`\$\w+`)
	matches := componentRegex.FindAllString(content, -1)
	
	if len(matches) == 0 {
		return
	}
	
	ps.projectData.mutex.Lock()
	defer ps.projectData.mutex.Unlock()
	
	// Clear previous components for this file
	if components, exists := ps.projectData.FileComponents[filePath]; exists {
		for comp := range components {
			if ps.projectData.ComponentFiles[comp] == filePath {
				delete(ps.projectData.ComponentFiles, comp)
			}
		}
	}
	ps.projectData.FileComponents[filePath] = make(map[string]bool)
	
	for _, match := range matches {
		ps.projectData.Components[match] = true
		// Only set file mapping if not already set by .view.tree file
		if _, exists := ps.projectData.ComponentFiles[match]; !exists {
			ps.projectData.ComponentFiles[match] = filePath
		}
		ps.projectData.FileComponents[filePath][match] = true
	}
}

func (ps *ProjectScanner) UpdateSingleFile(filePath, content string) {
	log.Printf("[view.tree] Updating single file: %s", filePath)
	
	if strings.HasSuffix(filePath, ".view.tree") {
		ps.parseViewTreeFile(content, filePath)
	} else if strings.HasSuffix(filePath, ".ts") {
		ps.parseTsFile(content, filePath)
	}
}

func (ps *ProjectScanner) GetProjectData() *ProjectData {
	return ps.projectData
}

func (ps *ProjectScanner) GetComponentsStartingWith(prefix string) []string {
	ps.projectData.mutex.RLock()
	defer ps.projectData.mutex.RUnlock()
	
	var components []string
	for component := range ps.projectData.Components {
		if strings.HasPrefix(component, prefix) {
			components = append(components, component)
		}
	}
	
	sort.Strings(components)
	return components
}

func (ps *ProjectScanner) GetPropertiesForComponent(component string) []string {
	ps.projectData.mutex.RLock()
	defer ps.projectData.mutex.RUnlock()
	
	properties, exists := ps.projectData.ComponentProperties[component]
	if !exists {
		return []string{}
	}
	
	var result []string
	for property := range properties {
		result = append(result, property)
	}
	
	sort.Strings(result)
	return result
}

func (ps *ProjectScanner) GetAllProperties() []string {
	ps.projectData.mutex.RLock()
	defer ps.projectData.mutex.RUnlock()
	
	allProperties := make(map[string]bool)
	for _, properties := range ps.projectData.ComponentProperties {
		for property := range properties {
			allProperties[property] = true
		}
	}
	
	var result []string
	for property := range allProperties {
		result = append(result, property)
	}
	
	sort.Strings(result)
	return result
}

func (ps *ProjectScanner) GetComponentFile(component string) string {
	ps.projectData.mutex.RLock()
	defer ps.projectData.mutex.RUnlock()
	
	return ps.projectData.ComponentFiles[component]
}

// GetComponents returns all components
func (ps *ProjectScanner) GetComponents() []string {
	ps.projectData.mutex.RLock()
	defer ps.projectData.mutex.RUnlock()
	
	var components []string
	for component := range ps.projectData.Components {
		components = append(components, component)
	}
	
	sort.Strings(components)
	return components
}

// HasComponent checks if a component exists
func (ps *ProjectScanner) HasComponent(component string) bool {
	ps.projectData.mutex.RLock()
	defer ps.projectData.mutex.RUnlock()
	
	return ps.projectData.Components[component]
}