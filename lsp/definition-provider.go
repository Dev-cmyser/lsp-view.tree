package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type DefinitionProvider struct {
	projectScanner *ProjectScanner
	parser         *ViewTreeParser
}

func NewDefinitionProvider(projectScanner *ProjectScanner) *DefinitionProvider {
	return &DefinitionProvider{
		projectScanner: projectScanner,
		parser:         NewViewTreeParser(),
	}
}

func (dp *DefinitionProvider) ProvideDefinition(document *TextDocument, position Position) ([]Location, error) {
	content := document.Text
	wordRange := dp.parser.GetWordRangeAtPosition(content, position)
	
	if wordRange == nil {
		return []Location{}, nil
	}
	
	nodeName := dp.getTextInRange(content, *wordRange)
	if nodeName == "" {
		return []Location{}, nil
	}
	
	nodeType := dp.getNodeType(content, position, *wordRange)
	documentURI := document.URI
	
	switch nodeType {
	case "root_class":
		return dp.findRootClassDefinition(documentURI, nodeName)
	case "class":
		return dp.findClassDefinition(nodeName)
	case "comp":
		return dp.findCompDefinition(documentURI, nodeName)
	case "prop":
		return dp.findPropDefinition(documentURI, nodeName)
	case "sub_prop":
		return dp.findSubPropDefinition(documentURI, position, nodeName)
	default:
		return []Location{}, nil
	}
}

func (dp *DefinitionProvider) getNodeType(content string, position Position, wordRange Range) string {
	// Root class - first line, first character after $
	if wordRange.Start.Character == 1 && wordRange.Start.Line == 0 {
		return "root_class"
	}
	
	lines := strings.Split(content, "\n")
	if position.Line >= len(lines) {
		return "sub_prop"
	}
	
	line := lines[position.Line]
	
	// Check if preceded by $
	if wordRange.Start.Character > 0 && wordRange.Start.Character-1 < len(line) {
		firstChar := line[wordRange.Start.Character-1]
		if firstChar == '$' {
			return "class"
		}
	}
	
	// Property at root level (character 1)
	if wordRange.Start.Character == 1 {
		return "prop"
	}
	
	// Check for binding operators before the word (translate -2, -1)
	if wordRange.Start.Character >= 2 && wordRange.Start.Character-2 < len(line) {
		leftNodeChar := line[wordRange.Start.Character-2]
		if leftNodeChar == '>' || leftNodeChar == '=' || leftNodeChar == '^' {
			return "prop"
		}
	}
	
	// Default to sub_prop for deeper nested items
	return "sub_prop"
}

func (dp *DefinitionProvider) findRootClassDefinition(documentURI, nodeName string) ([]Location, error) {
	// Find corresponding .ts file
	filePath := dp.uriToFilePath(documentURI)
	tsPath := strings.Replace(filePath, ".view.tree", ".ts", 1)
	tsURI := dp.filePathToURI(tsPath)
	
	// Check if .ts file exists
	if _, err := os.Stat(tsPath); err == nil {
		// Try to find class symbol in .ts file
		location, err := dp.findClassSymbolInFile(tsURI, "$"+nodeName)
		if err == nil && location != nil {
			return []Location{*location}, nil
		}
		
		// If no specific class found, return beginning of file
		locationRange := Range{
			Start: Position{Line: 0, Character: 0},
			End:   Position{Line: 0, Character: 0},
		}
		return []Location{{URI: tsURI, Range: locationRange}}, nil
	}
	
	// If .ts file doesn't exist, return empty
	return []Location{}, nil
}

func (dp *DefinitionProvider) findClassDefinition(nodeName string) ([]Location, error) {
	parts := strings.Split(nodeName, "_")
	workspaceRoot := dp.projectScanner.workspaceRoot
	
	// Try to find .view.tree file
	if len(parts) == 0 {
		return []Location{}, nil
	}
	
	lastPart := parts[len(parts)-1]
	
	possiblePaths := []string{
		filepath.Join(append([]string{workspaceRoot}, append(parts, lastPart+".view.tree")...)...),
		filepath.Join(append([]string{workspaceRoot}, append(parts, lastPart, lastPart+".view.tree")...)...),
	}
	
	for _, viewTreePath := range possiblePaths {
		if _, err := os.Stat(viewTreePath); err == nil {
			uri := dp.filePathToURI(viewTreePath)
			firstCharRange := Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: 0, Character: 0},
			}
			return []Location{{URI: uri, Range: firstCharRange}}, nil
		}
	}
	
	// Try to find in project data
	componentFile := dp.projectScanner.GetComponentFile(nodeName)
	if componentFile != "" {
		uri := dp.filePathToURI(componentFile)
		firstCharRange := Range{
			Start: Position{Line: 0, Character: 0},
			End:   Position{Line: 0, Character: 0},
		}
		return []Location{{URI: uri, Range: firstCharRange}}, nil
	}
	
	return []Location{}, nil
}

func (dp *DefinitionProvider) findCompDefinition(documentURI, nodeName string) ([]Location, error) {
	// Find corresponding .css.ts file
	filePath := dp.uriToFilePath(documentURI)
	cssPath := strings.Replace(filePath, ".view.tree", ".css.ts", 1)
	cssURI := dp.filePathToURI(cssPath)
	
	if _, err := os.Stat(cssPath); err == nil {
		// Try to find the CSS class definition
		content, err := os.ReadFile(cssPath)
		if err == nil {
			cssRule := dp.extractCssRule(string(content), nodeName)
			if cssRule != nil {
				return []Location{*cssRule}, nil
			}
		}
		
		// If no specific match, return beginning of file
		locationRange := Range{
			Start: Position{Line: 0, Character: 0},
			End:   Position{Line: 0, Character: 0},
		}
		return []Location{{URI: cssURI, Range: locationRange}}, nil
	}
	
	return []Location{}, nil
}

func (dp *DefinitionProvider) findPropDefinition(documentURI, nodeName string) ([]Location, error) {
	// Get the current component name
	content, err := dp.getDocumentContent(documentURI)
	if err != nil {
		return []Location{}, err
	}
	
	currentComponent := dp.getCurrentComponentFromContent(content)
	if currentComponent == "" {
		return []Location{}, nil
	}
	
	// Find corresponding .ts file
	filePath := dp.uriToFilePath(documentURI)
	tsPath := strings.Replace(filePath, ".view.tree", ".ts", 1)
	tsURI := dp.filePathToURI(tsPath)
	
	if _, err := os.Stat(tsPath); err == nil {
		// Find property in .ts file
		propLocation, err := dp.findPropertyInFile(tsURI, currentComponent, nodeName)
		if err == nil && propLocation != nil {
			return []Location{*propLocation}, nil
		}
		
		// Fallback to comp definition
		return dp.findCompDefinition(documentURI, nodeName)
	}
	
	return []Location{}, nil
}

func (dp *DefinitionProvider) findSubPropDefinition(documentURI string, position Position, nodeName string) ([]Location, error) {
	// This is a simplified version - in the original code this uses source maps
	// For now, we'll try to find it as a regular property
	return dp.findPropDefinition(documentURI, nodeName)
}

func (dp *DefinitionProvider) findClassSymbolInFile(fileURI, className string) (*Location, error) {
	filePath := dp.uriToFilePath(fileURI)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	
	// Look for class definition
	escapedClassName := regexp.QuoteMeta(className)
	classRegex := regexp.MustCompile(`class\s+` + escapedClassName + `\b`)
	match := classRegex.FindIndex(content)
	
	if match != nil {
		lines := strings.Split(string(content[:match[0]]), "\n")
		line := len(lines) - 1
		var character int
		if len(lines) > 0 {
			character = len(lines[len(lines)-1])
		}
		
		r := Range{
			Start: Position{Line: line, Character: character},
			End:   Position{Line: line, Character: character + len(className)},
		}
		
		return &Location{URI: fileURI, Range: r}, nil
	}
	
	return nil, nil
}

func (dp *DefinitionProvider) findPropertyInFile(fileURI, className, propertyName string) (*Location, error) {
	filePath := dp.uriToFilePath(fileURI)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	
	contentStr := string(content)
	
	// Look for property definition within class
	escapedClassName := regexp.QuoteMeta(className)
	classRegex := regexp.MustCompile(`class\s+` + escapedClassName + `[^{]*\{([^}]*(?:\{[^}]*\}[^}]*)*)\}`)
	classMatch := classRegex.FindStringSubmatch(contentStr)
	
	if len(classMatch) > 1 {
		classContent := classMatch[1]
		escapedPropertyName := regexp.QuoteMeta(propertyName)
		propRegex := regexp.MustCompile(`\b` + escapedPropertyName + `\s*[(:=]`)
		propMatch := propRegex.FindStringIndex(classContent)
		
		if propMatch != nil {
			// Find the position in the original content
			classStart := strings.Index(contentStr, classContent)
			propStart := classStart + propMatch[0]
			
			beforeMatch := contentStr[:propStart]
			lines := strings.Split(beforeMatch, "\n")
			line := len(lines) - 1
			var character int
			if len(lines) > 0 {
				character = len(lines[len(lines)-1])
			}
			
			r := Range{
				Start: Position{Line: line, Character: character},
				End:   Position{Line: line, Character: character + len(propertyName)},
			}
			
			return &Location{URI: fileURI, Range: r}, nil
		}
	}
	
	return nil, nil
}

func (dp *DefinitionProvider) extractCssRule(cssContent, className string) *Location {
	// Look for CSS class definition in TypeScript CSS-in-JS format
	escapedClassName := regexp.QuoteMeta(className)
	classRegex := regexp.MustCompile(escapedClassName + `\s*:\s*\{`)
	match := classRegex.FindStringIndex(cssContent)
	
	if match != nil {
		lines := strings.Split(cssContent[:match[0]], "\n")
		line := len(lines) - 1
		var character int
		if len(lines) > 0 {
			character = len(lines[len(lines)-1])
		}
		
		r := Range{
			Start: Position{Line: line, Character: character},
			End:   Position{Line: line, Character: character + len(className)},
		}
		
		// We need a file URI - this should be constructed from the CSS file path
		// For now, return a location with empty URI as we'd need the actual file URI
		return &Location{URI: "", Range: r}
	}
	
	return nil
}

func (dp *DefinitionProvider) getDocumentContent(uri string) (string, error) {
	filePath := dp.uriToFilePath(uri)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (dp *DefinitionProvider) getCurrentComponentFromContent(content string) string {
	lines := strings.Split(content, "\n")
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(line, "\t") && !strings.HasPrefix(line, " ") && strings.HasPrefix(trimmed, "$") {
			fields := strings.Fields(trimmed)
			if len(fields) > 0 && strings.HasPrefix(fields[0], "$") {
				return fields[0]
			}
		}
	}
	
	return ""
}

func (dp *DefinitionProvider) getTextInRange(content string, r Range) string {
	lines := strings.Split(content, "\n")
	if r.Start.Line >= len(lines) {
		return ""
	}
	
	line := lines[r.Start.Line]
	if r.Start.Character >= len(line) || r.End.Character > len(line) {
		return ""
	}
	
	return line[r.Start.Character:r.End.Character]
}

func (dp *DefinitionProvider) uriToFilePath(uri string) string {
	// Simple URI to file path conversion
	if strings.HasPrefix(uri, "file://") {
		return strings.TrimPrefix(uri, "file://")
	}
	return uri
}

func (dp *DefinitionProvider) filePathToURI(filePath string) string {
	// Simple file path to URI conversion
	if !strings.HasPrefix(filePath, "file://") {
		return "file://" + filePath
	}
	return filePath
}