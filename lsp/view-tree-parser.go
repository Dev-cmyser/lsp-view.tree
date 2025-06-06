package main

import (
	"regexp"
	"strings"
)

type ParsedComponent struct {
	Name       string          `json:"name"`
	Range      Range           `json:"range"`
	Properties []ParsedProperty `json:"properties"`
	StartLine  int             `json:"startLine"`
	EndLine    int             `json:"endLine"`
}

type ParsedProperty struct {
	Name        string  `json:"name"`
	Range       Range   `json:"range"`
	Line        int     `json:"line"`
	IndentLevel int     `json:"indentLevel"`
	IsBinding   bool    `json:"isBinding"`
	BindingType string  `json:"bindingType,omitempty"` // "one-way", "two-way", "override"
	Value       string  `json:"value,omitempty"`
}

type ParsedNode struct {
	Type        string `json:"type"` // "root_class", "class", "comp", "prop", "sub_prop"
	Name        string `json:"name"`
	Range       Range  `json:"range"`
	Line        int    `json:"line"`
	IndentLevel int    `json:"indentLevel"`
}

type ParseResult struct {
	Components []ParsedComponent `json:"components"`
	Nodes      []ParsedNode      `json:"nodes"`
	Errors     []ParseError      `json:"errors"`
}

type ParseError struct {
	Message  string             `json:"message"`
	Range    Range              `json:"range"`
	Severity string             `json:"severity"` // "error", "warning", "info"
}

type ViewTreeParser struct {
	lines []string
}

func NewViewTreeParser() *ViewTreeParser {
	return &ViewTreeParser{}
}

func (vtp *ViewTreeParser) Parse(content string) ParseResult {
	vtp.lines = strings.Split(content, "\n")

	result := ParseResult{
		Components: []ParsedComponent{},
		Nodes:      []ParsedNode{},
		Errors:     []ParseError{},
	}

	// Stack to track components by indentation level
	componentStack := make(map[int]*ParsedComponent)
	var rootComponent *ParsedComponent

	for lineIndex, line := range vtp.lines {
		if line == "" {
			continue
		}
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}

		indentLevel := vtp.getIndentLevel(line)

		// Root level component definition
		if indentLevel == 0 && strings.HasPrefix(trimmed, "$") {
			// Finish previous root component
			if rootComponent != nil {
				rootComponent.EndLine = lineIndex - 1
				result.Components = append(result.Components, *rootComponent)
			}

			// Clear component stack for new root component
			componentStack = make(map[int]*ParsedComponent)

			// Start new root component
			fields := strings.Fields(trimmed)
			if len(fields) == 0 {
				continue
			}
			firstWord := fields[0]
			wordRange := vtp.getWordRange(lineIndex, strings.Index(line, firstWord), firstWord)

			rootComponent = &ParsedComponent{
				Name:       firstWord,
				Range:      wordRange,
				Properties: []ParsedProperty{},
				StartLine:  lineIndex,
				EndLine:    lineIndex,
			}
			componentStack[0] = rootComponent

			// Add node for root class
			nodeType := "class"
			if lineIndex == 0 && wordRange.Start.Character == 1 {
				nodeType = "root_class"
			}
			result.Nodes = append(result.Nodes, ParsedNode{
				Type:        nodeType,
				Name:        firstWord,
				Range:       wordRange,
				Line:        lineIndex,
				IndentLevel: 0,
			})
		} else if indentLevel > 0 {
			// Check if this line contains a component reference
			componentRef := vtp.extractComponentReference(line)
			if componentRef != "" {
				// Create new component entry for this indentation level
				wordRange := vtp.getWordRange(lineIndex, strings.Index(line, componentRef), componentRef)
				newComponent := &ParsedComponent{
					Name:       componentRef,
					Range:      wordRange,
					Properties: []ParsedProperty{},
					StartLine:  lineIndex,
					EndLine:    lineIndex,
				}
				componentStack[indentLevel] = newComponent
			}

			// Find the current component for this indentation level
			var currentComponent *ParsedComponent
			for level := indentLevel; level >= 0; level-- {
				if comp, exists := componentStack[level]; exists {
					currentComponent = comp
					break
				}
			}

			if currentComponent != nil {
				// Property or sub-component
				wordMatch := regexp.MustCompile(`^(\s+)([a-zA-Z_$][a-zA-Z0-9_?*]*)`).FindStringSubmatch(line)
				if len(wordMatch) > 2 {
					propertyName := wordMatch[2]
					if propertyName == "" {
						continue
					}
					propertyStart := strings.Index(line, propertyName)
					wordRange := vtp.getWordRange(lineIndex, propertyStart, propertyName)

					// Determine if it's a binding
					isBinding := strings.Contains(trimmed, "<=") || strings.Contains(trimmed, "<=>")
					var bindingType string
					var value string

					if isBinding {
						if strings.Contains(trimmed, "<=>") {
							bindingType = "two-way"
						} else if strings.Contains(trimmed, "<=") {
							bindingType = "one-way"
						}

						// Extract bound property name
						bindingMatch := regexp.MustCompile(`<=>\s*([a-zA-Z_][a-zA-Z0-9_?*]*)|<=\s*([a-zA-Z_][a-zA-Z0-9_?*]*)`).FindStringSubmatch(trimmed)
						if len(bindingMatch) > 1 {
							if bindingMatch[1] != "" {
								value = bindingMatch[1]
							} else if len(bindingMatch) > 2 {
								value = bindingMatch[2]
							}
						}
					} else if strings.Contains(trimmed, "^") {
						bindingType = "override"
					} else {
						// Extract value after property name
						valueMatch := regexp.MustCompile(`^[a-zA-Z_$][a-zA-Z0-9_?*]*\s+(.+)$`).FindStringSubmatch(trimmed)
						if len(valueMatch) > 1 {
							value = strings.TrimSpace(valueMatch[1])
						}
					}

					property := ParsedProperty{
						Name:        propertyName,
						Range:       wordRange,
						Line:        lineIndex,
						IndentLevel: indentLevel,
						IsBinding:   isBinding,
						BindingType: bindingType,
						Value:       value,
					}

					currentComponent.Properties = append(currentComponent.Properties, property)

					// Determine node type
					var nodeType string
					if strings.HasPrefix(propertyName, "$") {
						nodeType = "comp"
					} else if indentLevel == 1 {
						nodeType = "prop"
					} else {
						nodeType = "sub_prop"
					}

					result.Nodes = append(result.Nodes, ParsedNode{
						Type:        nodeType,
						Name:        propertyName,
						Range:       wordRange,
						Line:        lineIndex,
						IndentLevel: indentLevel,
					})
				}
			} else if indentLevel > 0 {
				// Error: indented line without current component
				errorRange := Range{
					Start: Position{Line: lineIndex, Character: 0},
					End:   Position{Line: lineIndex, Character: len(line)},
				}
				result.Errors = append(result.Errors, ParseError{
					Message:  "Property defined outside of component",
					Range:    errorRange,
					Severity: "error",
				})
			}
		}
	}

	// Finish last root component
	if rootComponent != nil {
		rootComponent.EndLine = len(vtp.lines) - 1
		result.Components = append(result.Components, *rootComponent)
	}

	return result
}

func (vtp *ViewTreeParser) GetNodeAtPosition(content string, position Position) *ParsedNode {
	parseResult := vtp.Parse(content)

	for _, node := range parseResult.Nodes {
		if vtp.isPositionInRange(position, node.Range) {
			return &node
		}
	}

	return nil
}

func (vtp *ViewTreeParser) GetWordRangeAtPosition(content string, position Position) *Range {
	vtp.lines = strings.Split(content, "\n")

	if position.Line >= len(vtp.lines) {
		return nil
	}

	line := vtp.lines[position.Line]
	if line == "" {
		return nil
	}
	character := position.Character

	// Find word boundaries
	start := character
	end := character

	// Move start backwards to find word start
	for start > 0 && start-1 < len(line) && vtp.isWordCharacter(rune(line[start-1])) {
		start--
	}

	// Move end forwards to find word end
	for end < len(line) && vtp.isWordCharacter(rune(line[end])) {
		end++
	}

	if start == end {
		return nil
	}

	return &Range{
		Start: Position{Line: position.Line, Character: start},
		End:   Position{Line: position.Line, Character: end},
	}
}

func (vtp *ViewTreeParser) GetCurrentComponent(content string, position Position) string {
	vtp.lines = strings.Split(content, "\n")

	if position.Line >= len(vtp.lines) {
		return ""
	}

	// First, check if current line contains a component reference
	currentLine := vtp.lines[position.Line]
	if componentInLine := vtp.extractComponentFromLine(currentLine); componentInLine != "" {
		return componentInLine
	}

	// Look backwards to find the closest component that owns this position
	currentIndent := vtp.getIndentLevel(currentLine)
	
	for i := position.Line - 1; i >= 0; i-- {
		line := vtp.lines[i]
		if line == "" {
			continue
		}
		
		lineIndent := vtp.getIndentLevel(line)
		
		// If we find a line with less indentation, check if it contains a component
		if lineIndent < currentIndent {
			if componentInLine := vtp.extractComponentFromLine(line); componentInLine != "" {
				return componentInLine
			}
		}
		
		// If line has no indentation and starts with $, it's a root component
		if lineIndent == 0 {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "$") {
				fields := strings.Fields(trimmed)
				if len(fields) > 0 && strings.HasPrefix(fields[0], "$") {
					return fields[0]
				}
			}
		}
	}

	return ""
}

func (vtp *ViewTreeParser) extractComponentFromLine(line string) string {
	// Look for component references like "<= Button $mol_button_major"
	trimmed := strings.TrimSpace(line)
	
	// Check for binding patterns with components
	patterns := []string{
		`<=\s+\w+\s+(\$\w+)`,
		`=>\s+\w+\s+(\$\w+)`,
		`<=>\s+\w+\s+(\$\w+)`,
		`^\s*(\$\w+)`, // Direct component reference
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(trimmed); len(matches) > 1 {
			return matches[1]
		}
	}
	
	return ""
}

func (vtp *ViewTreeParser) extractComponentReference(line string) string {
	// Extract component reference from binding lines like "<= Button $mol_button_major"
	trimmed := strings.TrimSpace(line)
	
	// Check for component references in bindings
	patterns := []string{
		`<=\s+\w+\s+(\$\w+)`,
		`=>\s+\w+\s+(\$\w+)`,
		`<=>\s+\w+\s+(\$\w+)`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(trimmed); len(matches) > 1 {
			return matches[1]
		}
	}
	
	return ""
}

func (vtp *ViewTreeParser) getIndentLevel(line string) int {
	indent := 0
	for _, char := range line {
		if char == '\t' {
			indent++
		} else {
			break
		}
	}
	return indent
}

func (vtp *ViewTreeParser) getWordRange(line, start int, word string) Range {
	return Range{
		Start: Position{Line: line, Character: start},
		End:   Position{Line: line, Character: start + len(word)},
	}
}

func (vtp *ViewTreeParser) isPositionInRange(position Position, r Range) bool {
	if position.Line < r.Start.Line || position.Line > r.End.Line {
		return false
	}

	if position.Line == r.Start.Line && position.Character < r.Start.Character {
		return false
	}

	if position.Line == r.End.Line && position.Character > r.End.Character {
		return false
	}

	return true
}

func (vtp *ViewTreeParser) isWordCharacter(char rune) bool {
	return (char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z') ||
		(char >= '0' && char <= '9') ||
		char == '_' || char == '$' || char == '?' || char == '*'
}

// ValidateSyntax validates view.tree syntax
func (vtp *ViewTreeParser) ValidateSyntax(content string) []ParseError {
	parseResult := vtp.Parse(content)
	errors := make([]ParseError, len(parseResult.Errors))
	copy(errors, parseResult.Errors)

	// Add additional validation rules
	componentNames := make(map[string][]ParsedComponent)
	for _, component := range parseResult.Components {
		componentNames[component.Name] = append(componentNames[component.Name], component)
	}

	// Check for duplicate component names
	for name, components := range componentNames {
		if len(components) > 1 {
			for i := 1; i < len(components); i++ {
				errors = append(errors, ParseError{
					Message:  "Duplicate component name: " + name,
					Range:    components[i].Range,
					Severity: "warning",
				})
			}
		}
	}

	// Check for invalid property names
	for _, component := range parseResult.Components {
		for _, property := range component.Properties {
			if !vtp.isValidPropertyName(property.Name) {
				errors = append(errors, ParseError{
					Message:  "Invalid property name: " + property.Name,
					Range:    property.Range,
					Severity: "error",
				})
			}
		}
	}

	return errors
}

func (vtp *ViewTreeParser) isValidPropertyName(name string) bool {
	// Basic validation - starts with letter or underscore, contains only alphanumeric, underscore, ?, *
	matched, _ := regexp.MatchString(`^[a-zA-Z_$][a-zA-Z0-9_?*]*$`, name)
	return matched
}