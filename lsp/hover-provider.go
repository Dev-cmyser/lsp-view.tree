package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type HoverProvider struct {
	projectScanner *ProjectScanner
	parser         *ViewTreeParser
}

func NewHoverProvider(projectScanner *ProjectScanner) *HoverProvider {
	return &HoverProvider{
		projectScanner: projectScanner,
		parser:         NewViewTreeParser(),
	}
}

func (hp *HoverProvider) ProvideHover(document *TextDocument, position Position) (*Hover, error) {
	content := document.Text
	wordRange := hp.parser.GetWordRangeAtPosition(content, position)
	
	if wordRange == nil {
		return nil, nil
	}
	
	nodeName := hp.getTextInRange(content, *wordRange)
	if nodeName == "" {
		return nil, nil
	}
	
	nodeType := hp.getNodeType(content, position, *wordRange)
	documentURI := document.URI
	
	var hoverContent *MarkupContent
	var err error
	
	switch nodeType {
	case "root_class":
		hoverContent, err = hp.getComponentHover(nodeName, documentURI)
	case "class":
		hoverContent, err = hp.getComponentHover(nodeName, "")
	case "comp":
		hoverContent, err = hp.getCssClassHover(nodeName, documentURI)
	case "prop":
		hoverContent = hp.getPropertyHover(nodeName, content)
	case "sub_prop":
		hoverContent = hp.getSubPropertyHover(nodeName, content)
	default:
		hoverContent = hp.getGenericHover(nodeName)
	}
	
	if err != nil {
		log.Printf("[view.tree] Error providing hover: %v", err)
		return nil, err
	}
	
	if hoverContent == nil {
		return nil, nil
	}
	
	return &Hover{
		Contents: *hoverContent,
		Range:    wordRange,
	}, nil
}

func (hp *HoverProvider) getNodeType(content string, position Position, wordRange Range) string {
	lines := strings.Split(content, "\n")
	if position.Line >= len(lines) {
		return "sub_prop"
	}
	
	line := lines[position.Line]
	
	// Get the actual text of the word
	nodeText := hp.getTextInRange(content, wordRange)
	
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

func (hp *HoverProvider) getComponentHover(componentName, documentURI string) (*MarkupContent, error) {
	projectData := hp.projectScanner.GetProjectData()
	
	projectData.mutex.RLock()
	hasComponent := projectData.Components[componentName]
	projectData.mutex.RUnlock()
	
	var markdownContent []string
	
	// Component header
	markdownContent = append(markdownContent, fmt.Sprintf("**Component**: `%s`", componentName))
	markdownContent = append(markdownContent, "")
	
	// Show inheritance if available (parse from component name pattern)
	if strings.HasPrefix(componentName, "$mol_") {
		markdownContent = append(markdownContent, "**Framework**: MOL Framework")
		markdownContent = append(markdownContent, "")
	}
	
	if !hasComponent {
		markdownContent = append(markdownContent, "*External component - not found in current project*")
		markdownContent = append(markdownContent, "")
		
		// Try to infer file path from component name
		if strings.HasPrefix(componentName, "$") {
			parts := strings.Split(componentName[1:], "_")
			if len(parts) > 0 {
				lastPart := parts[len(parts)-1]
				expectedPath := strings.Join(parts, "/") + "/" + lastPart + ".view.tree"
				markdownContent = append(markdownContent, fmt.Sprintf("**Expected path**: `%s`", expectedPath))
				markdownContent = append(markdownContent, "")
			}
		}
		
		return &MarkupContent{
			Kind:  MarkupKindMarkdown,
			Value: strings.Join(markdownContent, "\n"),
		}, nil
	}
	
	// Component file location
	componentFile := hp.projectScanner.GetComponentFile(componentName)
	if componentFile != "" {
		relativePath := hp.getRelativePath(componentFile)
		markdownContent = append(markdownContent, fmt.Sprintf("**File**: `%s`", relativePath))
		markdownContent = append(markdownContent, "")
	}
	
	// Component properties
	properties := hp.projectScanner.GetPropertiesForComponent(componentName)
	if len(properties) > 0 {
		markdownContent = append(markdownContent, "**Properties**:")
		maxProps := 10
		if len(properties) > maxProps {
			for _, prop := range properties[:maxProps] {
				markdownContent = append(markdownContent, fmt.Sprintf("- `%s`", prop))
			}
			markdownContent = append(markdownContent, fmt.Sprintf("- ... and %d more", len(properties)-maxProps))
		} else {
			for _, prop := range properties {
				markdownContent = append(markdownContent, fmt.Sprintf("- `%s`", prop))
			}
		}
		markdownContent = append(markdownContent, "")
	}
	
	// Component documentation from TypeScript file
	if documentURI != "" {
		tsDoc, err := hp.getTypeScriptDocumentation(componentName, documentURI)
		if err == nil && tsDoc != "" {
			markdownContent = append(markdownContent, "**Documentation**:")
			markdownContent = append(markdownContent, tsDoc)
			markdownContent = append(markdownContent, "")
		}
	}
	
	// Usage information
	markdownContent = append(markdownContent, "**Usage**:")
	markdownContent = append(markdownContent, "```tree")
	markdownContent = append(markdownContent, componentName)
	if len(properties) > 0 {
		markdownContent = append(markdownContent, "\tproperty <= value")
	}
	markdownContent = append(markdownContent, "```")
	
	return &MarkupContent{
		Kind:  MarkupKindMarkdown,
		Value: strings.Join(markdownContent, "\n"),
	}, nil
}

func (hp *HoverProvider) getCssClassHover(className, documentURI string) (*MarkupContent, error) {
	var markdownContent []string
	
	markdownContent = append(markdownContent, fmt.Sprintf("**CSS Class**: `%s`", className))
	markdownContent = append(markdownContent, "")
	
	// Try to find CSS definition
	filePath := hp.uriToFilePath(documentURI)
	cssPath := strings.Replace(filePath, ".view.tree", ".css.ts", 1)
	
	if _, err := os.Stat(cssPath); err == nil {
		relativePath := hp.getRelativePath(cssPath)
		markdownContent = append(markdownContent, fmt.Sprintf("**Defined in**: `%s`", relativePath))
		markdownContent = append(markdownContent, "")
		
		// Try to extract CSS rules
		cssContent, err := os.ReadFile(cssPath)
		if err == nil {
			cssRule := hp.extractCssRule(string(cssContent), className)
			if cssRule != "" {
				markdownContent = append(markdownContent, "**CSS Rules**:")
				markdownContent = append(markdownContent, "```css")
				markdownContent = append(markdownContent, cssRule)
				markdownContent = append(markdownContent, "```")
			}
		}
	} else {
		markdownContent = append(markdownContent, "*CSS file not found*")
	}
	
	return &MarkupContent{
		Kind:  MarkupKindMarkdown,
		Value: strings.Join(markdownContent, "\n"),
	}, nil
}

func (hp *HoverProvider) getPropertyHover(propertyName, content string) *MarkupContent {
	currentComponent := hp.parser.GetCurrentComponent(content, Position{Line: 0, Character: 0})
	var markdownContent []string
	
	markdownContent = append(markdownContent, fmt.Sprintf("**Property**: `%s`", propertyName))
	markdownContent = append(markdownContent, "")
	
	if currentComponent != "" {
		markdownContent = append(markdownContent, fmt.Sprintf("**Component**: `%s`", currentComponent))
		markdownContent = append(markdownContent, "")
	}
	
	// Find property context in the current file
	propertyContext := hp.findPropertyContext(propertyName, content)
	if propertyContext != nil {
		if propertyContext.BindingType != "" {
			markdownContent = append(markdownContent, fmt.Sprintf("**Binding**: `%s`", propertyContext.BindingType))
			markdownContent = append(markdownContent, "")
		}
		if propertyContext.Value != "" {
			markdownContent = append(markdownContent, fmt.Sprintf("**Value**: `%s`", propertyContext.Value))
			markdownContent = append(markdownContent, "")
		}
		if propertyContext.BoundProperty != "" {
			markdownContent = append(markdownContent, fmt.Sprintf("**Bound to**: `%s`", propertyContext.BoundProperty))
			markdownContent = append(markdownContent, "")
		}
	}
	
	// Common property descriptions
	propertyDesc := hp.getCommonPropertyDescription(propertyName)
	if propertyDesc != "" {
		markdownContent = append(markdownContent, fmt.Sprintf("**Description**: %s", propertyDesc))
		markdownContent = append(markdownContent, "")
	}
	
	// Usage examples
	usageExample := hp.getPropertyUsageExample(propertyName)
	if usageExample != "" {
		markdownContent = append(markdownContent, "**Usage**:")
		markdownContent = append(markdownContent, "```tree")
		markdownContent = append(markdownContent, usageExample)
		markdownContent = append(markdownContent, "```")
	}
	
	return &MarkupContent{
		Kind:  MarkupKindMarkdown,
		Value: strings.Join(markdownContent, "\n"),
	}
}

func (hp *HoverProvider) getSubPropertyHover(propertyName, content string) *MarkupContent {
	// For sub-properties, provide similar information as regular properties
	return hp.getPropertyHover(propertyName, content)
}

type PropertyContext struct {
	BindingType    string // "<=", "<=>", "=>", "^", ""
	Value          string
	BoundProperty  string
}

func (hp *HoverProvider) findPropertyContext(propertyName, content string) *PropertyContext {
	lines := strings.Split(content, "\n")
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Look for property definitions
		if strings.Contains(line, propertyName) {
			// Check for different binding types
			if match := regexp.MustCompile(fmt.Sprintf(`%s\s*<=>\s*([^\\s]+)`, regexp.QuoteMeta(propertyName))).FindStringSubmatch(trimmed); len(match) > 1 {
				return &PropertyContext{
					BindingType:   "<=>",
					BoundProperty: match[1],
				}
			}
			if match := regexp.MustCompile(fmt.Sprintf(`%s\s*<=\s*([^\\s]+)`, regexp.QuoteMeta(propertyName))).FindStringSubmatch(trimmed); len(match) > 1 {
				return &PropertyContext{
					BindingType:   "<=",
					BoundProperty: match[1],
				}
			}
			if match := regexp.MustCompile(fmt.Sprintf(`%s\s*=>\s*([^\\s]+)`, regexp.QuoteMeta(propertyName))).FindStringSubmatch(trimmed); len(match) > 1 {
				return &PropertyContext{
					BindingType:   "=>",
					BoundProperty: match[1],
				}
			}
			if match := regexp.MustCompile(fmt.Sprintf(`%s\s*\^\s*([^\\s]+)`, regexp.QuoteMeta(propertyName))).FindStringSubmatch(trimmed); len(match) > 1 {
				return &PropertyContext{
					BindingType: "^",
					Value:       match[1],
				}
			}
			// Check for direct value assignment
			if match := regexp.MustCompile(fmt.Sprintf(`%s\s+(.+)`, regexp.QuoteMeta(propertyName))).FindStringSubmatch(trimmed); len(match) > 1 {
				value := strings.TrimSpace(match[1])
				if !strings.HasPrefix(value, "<=") && !strings.HasPrefix(value, "=>") && !strings.HasPrefix(value, "^") {
					return &PropertyContext{
						Value: value,
					}
				}
			}
		}
	}
	
	return nil
}

func (hp *HoverProvider) getCommonPropertyDescription(propertyName string) string {
	commonProps := map[string]string{
		"title":       "Display text or label for the component",
		"hint":        "Placeholder or helper text",
		"value":       "Current value of the component",
		"enabled":     "Whether the component is enabled/disabled",
		"visible":     "Whether the component is visible",
		"click":       "Click event handler",
		"change":      "Change event handler",
		"focus":       "Focus event handler",
		"blur":        "Blur event handler",
		"sub":         "Sub-components or child elements",
		"content":     "Content area of the component",
		"plugins":     "Plugin configurations",
		"attr":        "HTML attributes",
		"field":       "Form field configuration",
		"uri":         "URL or URI reference",
		"rows":        "List of row items",
		"dom_name":    "HTML tag name",
		"dom_name_space": "HTML namespace",
	}
	
	return commonProps[propertyName]
}

func (hp *HoverProvider) getPropertyUsageExample(propertyName string) string {
	examples := map[string]string{
		"title":   fmt.Sprintf("\t%s @ \\Display Text", propertyName),
		"hint":    fmt.Sprintf("\t%s @ \\Placeholder text", propertyName),
		"value":   fmt.Sprintf("\t%s? <=> bound_property? \\default", propertyName),
		"enabled": fmt.Sprintf("\t%s <= is_enabled", propertyName),
		"click":   fmt.Sprintf("\t%s? <=> on_click? null", propertyName),
		"sub":     fmt.Sprintf("\t%s /\n\t\t<= Item $component", propertyName),
		"content": fmt.Sprintf("\t%s /\n\t\t<= Child $component", propertyName),
	}
	
	if example, exists := examples[propertyName]; exists {
		return example
	}
	
	return fmt.Sprintf("\t%s <= some_value", propertyName)
}

func (hp *HoverProvider) getGenericHover(nodeName string) *MarkupContent {
	var markdownContent []string
	
	markdownContent = append(markdownContent, fmt.Sprintf("**Element**: `%s`", nodeName))
	markdownContent = append(markdownContent, "")
	
	// Check if it's a special value
	specialValueInfo := hp.getSpecialValueInfo(nodeName)
	if specialValueInfo != nil {
		markdownContent = append(markdownContent, fmt.Sprintf("**Type**: %s", specialValueInfo.Type))
		markdownContent = append(markdownContent, "")
		markdownContent = append(markdownContent, fmt.Sprintf("**Description**: %s", specialValueInfo.Description))
		markdownContent = append(markdownContent, "")
	}
	
	if len(markdownContent) <= 2 {
		return nil // No useful information to show
	}
	
	return &MarkupContent{
		Kind:  MarkupKindMarkdown,
		Value: strings.Join(markdownContent, "\n"),
	}
}

type PropertyTypeInfo struct {
	Type        string
	Description string
}

func (hp *HoverProvider) getPropertyTypeInfo(propertyName string) *PropertyTypeInfo {
	propertyTypes := map[string]PropertyTypeInfo{
		"dom_name": {
			Type:        "string",
			Description: "HTML tag name for the DOM element",
		},
		"dom_name_space": {
			Type:        "string",
			Description: "XML namespace for the DOM element",
		},
		"attr": {
			Type:        "Dictionary<string>",
			Description: "HTML attributes for the DOM element",
		},
		"field": {
			Type:        "any",
			Description: "Form field value binding",
		},
		"value": {
			Type:        "any",
			Description: "Element value or content",
		},
		"enabled": {
			Type:        "boolean",
			Description: "Whether the element is enabled",
		},
		"visible": {
			Type:        "boolean",
			Description: "Whether the element is visible",
		},
		"title": {
			Type:        "string",
			Description: "Element title or tooltip text",
		},
		"hint": {
			Type:        "string",
			Description: "Hint text for the element",
		},
		"sub": {
			Type:        "Array<$mol_view>",
			Description: "Child elements or components",
		},
		"event": {
			Type:        "Dictionary<Function>",
			Description: "Event handlers",
		},
		"plugins": {
			Type:        "Array<$mol_plugin>",
			Description: "Plugins to apply to the element",
		},
	}
	
	if info, exists := propertyTypes[propertyName]; exists {
		return &info
	}
	return nil
}

func (hp *HoverProvider) getPropertyUsageExamples(propertyName string) []string {
	examples := map[string][]string{
		"dom_name": {
			"\tdom_name \\div",
			"\tdom_name \\span",
		},
		"attr": {
			"\tattr *",
			"\t\tclass \\my-class",
			"\t\tid \\my-id",
		},
		"field": {
			"\tfield <= value",
			"\tfield <=> current_value",
		},
		"value": {
			"\tvalue \\Hello World",
			"\tvalue <= text",
		},
		"enabled": {
			"\tenabled <= is_active",
			"\tenabled true",
		},
		"visible": {
			"\tvisible <= show_element",
			"\tvisible false",
		},
		"sub": {
			"\tsub /",
			"\t\t<= items",
			"\t\t$my_component",
		},
		"event": {
			"\tevent *",
			"\t\tclick <= handle_click",
		},
	}
	
	return examples[propertyName]
}

type SpecialValueInfo struct {
	Type        string
	Description string
}

func (hp *HoverProvider) getSpecialValueInfo(value string) *SpecialValueInfo {
	specialValues := map[string]SpecialValueInfo{
		"null": {
			Type:        "null",
			Description: "Represents an empty or undefined value",
		},
		"true": {
			Type:        "boolean",
			Description: "Boolean true value",
		},
		"false": {
			Type:        "boolean",
			Description: "Boolean false value",
		},
		"/": {
			Type:        "list",
			Description: "Empty list marker",
		},
		"*": {
			Type:        "dictionary",
			Description: "Dictionary marker for key-value pairs",
		},
		"\\": {
			Type:        "string",
			Description: "String literal marker",
		},
		"@\\": {
			Type:        "localized string",
			Description: "Localized string literal marker",
		},
	}
	
	if info, exists := specialValues[value]; exists {
		return &info
	}
	return nil
}

func (hp *HoverProvider) getTypeScriptDocumentation(componentName, documentURI string) (string, error) {
	filePath := hp.uriToFilePath(documentURI)
	tsPath := strings.Replace(filePath, ".view.tree", ".ts", 1)
	
	content, err := os.ReadFile(tsPath)
	if err != nil {
		return "", err
	}
	
	// Look for JSDoc comments before class definition
	escapedComponentName := regexp.QuoteMeta(componentName)
	classRegex := regexp.MustCompile(`/\*\*([\s\S]*?)\*/\s*export\s+class\s+` + escapedComponentName)
	match := classRegex.FindStringSubmatch(string(content))
	
	if len(match) > 1 {
		docComment := match[1]
		lines := strings.Split(docComment, "\n")
		var docLines []string
		
		for _, line := range lines {
			cleaned := regexp.MustCompile(`^\s*\*\s?`).ReplaceAllString(line, "")
			cleaned = strings.TrimSpace(cleaned)
			if cleaned != "" {
				docLines = append(docLines, cleaned)
			}
		}
		
		return strings.Join(docLines, "\n"), nil
	}
	
	return "", nil
}

func (hp *HoverProvider) extractCssRule(cssContent, className string) string {
	// Look for CSS class definition in TypeScript CSS-in-JS format
	escapedClassName := regexp.QuoteMeta(className)
	classRegex := regexp.MustCompile(escapedClassName + `\s*:\s*\{([^}]+)\}`)
	match := classRegex.FindStringSubmatch(cssContent)
	
	if len(match) > 1 {
		rules := match[1]
		lines := strings.Split(rules, "\n")
		var cleanedLines []string
		
		for _, line := range lines {
			cleaned := strings.TrimSpace(line)
			if cleaned != "" {
				cleanedLines = append(cleanedLines, cleaned)
			}
		}
		
		return strings.Join(cleanedLines, "\n")
	}
	
	return ""
}

func (hp *HoverProvider) getRelativePath(filePath string) string {
	workspaceRoot := hp.projectScanner.workspaceRoot
	relPath, err := filepath.Rel(workspaceRoot, filePath)
	if err != nil {
		return filePath
	}
	return relPath
}

func (hp *HoverProvider) getTextInRange(content string, r Range) string {
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

func (hp *HoverProvider) uriToFilePath(uri string) string {
	// Simple URI to file path conversion
	if strings.HasPrefix(uri, "file://") {
		return strings.TrimPrefix(uri, "file://")
	}
	return uri
}