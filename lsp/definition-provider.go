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
	lines := strings.Split(content, "\n")
	if position.Line >= len(lines) {
		return "sub_prop"
	}
	
	line := lines[position.Line]
	
	// Get the actual text of the word
	nodeText := dp.getTextInRange(content, wordRange)
	
	// Root class - first line, first character after $ (check before general component check)
	if position.Character == 1 && position.Line == 0 {
		return "root_class"
	}
	
	// Check if this is a component (starts with $)
	if strings.HasPrefix(nodeText, "$") {
		return "class"
	}
	
	// Check if preceded by $ (with possible spaces)
	beforeWord := line[:wordRange.Start.Character]
	if strings.Contains(beforeWord, "$") && strings.HasSuffix(strings.TrimSpace(beforeWord), "$") {
		return "class"
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
	tsPath := strings.Replace(filePath, ".tree", ".ts", 1)
	tsURI := dp.filePathToURI(tsPath)
	
	// Try to find class symbol in .ts file
	location, err := dp.findClassSymbolInFile(tsURI, "$"+nodeName)
	if err == nil && location != nil {
		return []Location{*location}, nil
	}
	
	// If no specific class found, return beginning of file (always return location like in reference)
	locationRange := Range{
		Start: Position{Line: 0, Character: 0},
		End:   Position{Line: 0, Character: 0},
	}
	return []Location{{URI: tsURI, Range: locationRange}}, nil
}

func (dp *DefinitionProvider) findClassDefinition(nodeName string) ([]Location, error) {
	parts := strings.Split(nodeName, "_")
	workspaceRoot := dp.projectScanner.workspaceRoot
	
	if len(parts) == 0 {
		return []Location{}, nil
	}
	
	lastPart := parts[len(parts)-1]
	firstCharRange := Range{
		Start: Position{Line: 0, Character: 0},
		End:   Position{Line: 0, Character: 0},
	}
	
	// First path: workspaceRoot/parts.join("/"), lastPart + ".view.tree"
	viewTreePath1 := filepath.Join(append([]string{workspaceRoot}, append(parts, lastPart+".view.tree")...)...)
	if _, err := os.Stat(viewTreePath1); err == nil {
		uri := dp.filePathToURI(viewTreePath1)
		return []Location{{URI: uri, Range: firstCharRange}}, nil
	}
	
	// Second path: workspaceRoot/[...parts, lastPart].join("/"), lastPart + ".view.tree"
	viewTreePath2 := filepath.Join(append([]string{workspaceRoot}, append(append(parts, lastPart), lastPart+".view.tree")...)...)
	if _, err := os.Stat(viewTreePath2); err == nil {
		uri := dp.filePathToURI(viewTreePath2)
		return []Location{{URI: uri, Range: firstCharRange}}, nil
	}
	
	// Try to find in project data (equivalent to workspace symbols)
	componentFile := dp.projectScanner.GetComponentFile(nodeName)
	if componentFile != "" {
		uri := dp.filePathToURI(componentFile)
		return []Location{{URI: uri, Range: firstCharRange}}, nil
	}
	
	// Always return first path location (even if file doesn't exist) like in reference
	uri := dp.filePathToURI(viewTreePath1)
	return []Location{{URI: uri, Range: firstCharRange}}, nil
}

func (dp *DefinitionProvider) findCompDefinition(documentURI, nodeName string) ([]Location, error) {
	// Find corresponding .css.ts file
	filePath := dp.uriToFilePath(documentURI)
	cssPath := strings.Replace(filePath, ".tree", ".css.ts", 1)
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
	// Get className from position 0,1 (like in reference)
	content, err := dp.getDocumentContent(documentURI)
	if err != nil {
		return []Location{}, err
	}
	
	// Get word at position 0,1 and add $ prefix
	className := dp.getClassNameAtPosition01(content)
	if className == "" {
		return []Location{}, nil
	}
	
	// Find corresponding .ts file
	filePath := dp.uriToFilePath(documentURI)
	tsPath := strings.Replace(filePath, ".tree", ".ts", 1)
	tsURI := dp.filePathToURI(tsPath)
	
	// Find property in .ts file
	propLocation, err := dp.findPropertyInFile(tsURI, className, nodeName)
	if err == nil && propLocation != nil {
		return []Location{*propLocation}, nil
	}
	
	// Always fallback to comp definition if no propSymbol found (like in reference)
	return dp.findCompDefinition(documentURI, nodeName)
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

func (dp *DefinitionProvider) getClassNameAtPosition01(content string) string {
	// Get word at position 0,1 and add $ prefix (like in reference)
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return ""
	}
	
	line := lines[0]
	if len(line) <= 1 {
		return ""
	}
	
	// Find word starting at position 1
	start := 1
	end := start
	
	// Move end forwards to find word end
	for end < len(line) && dp.isWordCharacter(rune(line[end])) {
		end++
	}
	
	if start == end {
		return ""
	}
	
	nodeName := line[start:end]
	return "$" + nodeName
}

func (dp *DefinitionProvider) isWordCharacter(char rune) bool {
	return (char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z') ||
		(char >= '0' && char <= '9') ||
		char == '_' || char == '$' || char == '?' || char == '*'
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