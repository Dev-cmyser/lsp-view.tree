package main

import (
	"fmt"
	"regexp"
	"strings"
)

type DiagnosticProvider struct {
	projectScanner *ProjectScanner
	parser         *ViewTreeParser
}

func NewDiagnosticProvider(projectScanner *ProjectScanner) *DiagnosticProvider {
	return &DiagnosticProvider{
		projectScanner: projectScanner,
		parser:         NewViewTreeParser(),
	}
}

func (dp *DiagnosticProvider) ProvideDiagnostics(document *TextDocument) ([]Diagnostic, error) {
	content := document.Text
	var diagnostics []Diagnostic

	// Only process .view.tree files
	if !strings.HasSuffix(document.URI, ".view.tree") {
		return diagnostics, nil
	}

	// Parse the document
	parseResult := dp.parser.Parse(content)

	// Add parse errors
	for _, parseError := range parseResult.Errors {
		diagnostics = append(diagnostics, Diagnostic{
			Severity: dp.mapSeverity(parseError.Severity),
			Range:    parseError.Range,
			Message:  parseError.Message,
			Source:   "view.tree",
		})
	}

	// Validate syntax
	syntaxDiagnostics := dp.validateSyntax(content, document.URI)
	diagnostics = append(diagnostics, syntaxDiagnostics...)

	// Validate components
	componentDiagnostics := dp.validateComponents(parseResult.Components, document.URI)
	diagnostics = append(diagnostics, componentDiagnostics...)

	// Validate properties
	propertyDiagnostics := dp.validateProperties(parseResult.Components, content)
	diagnostics = append(diagnostics, propertyDiagnostics...)

	// Validate indentation
	indentationDiagnostics := dp.validateIndentation(content)
	diagnostics = append(diagnostics, indentationDiagnostics...)

	// Validate bindings
	bindingDiagnostics := dp.validateBindings(content)
	diagnostics = append(diagnostics, bindingDiagnostics...)

	return diagnostics, nil
}

func (dp *DiagnosticProvider) validateSyntax(content, documentURI string) []Diagnostic {
	var diagnostics []Diagnostic
	lines := strings.Split(content, "\n")

	for lineIndex, line := range lines {
		if line == "" {
			continue
		}
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}

		// Check for invalid characters in component names
		if strings.HasPrefix(trimmed, "$") {
			fields := strings.Fields(trimmed)
			if len(fields) > 0 {
				componentName := fields[0]
				matched, _ := regexp.MatchString(`^\$[a-zA-Z_][a-zA-Z0-9_]*$`, componentName)
				if !matched {
					startChar := strings.Index(line, componentName)
					r := Range{
						Start: Position{Line: lineIndex, Character: startChar},
						End:   Position{Line: lineIndex, Character: startChar + len(componentName)},
					}
					diagnostics = append(diagnostics, Diagnostic{
						Severity: DiagnosticSeverityError,
						Range:    r,
						Message:  fmt.Sprintf("Invalid component name: %s. Component names must start with $ followed by letters, numbers, or underscores.", componentName),
						Source:   "view.tree",
					})
				}
			}
		}

		// Check for mixing tabs and spaces
		if len(line) > 0 && !strings.HasPrefix(trimmed, "//") {
			leadingWhitespace := regexp.MustCompile(`^(\s*)`).FindString(line)
			hasTab := strings.Contains(leadingWhitespace, "\t")
			hasSpace := strings.Contains(leadingWhitespace, " ")

			if hasTab && hasSpace {
				r := Range{
					Start: Position{Line: lineIndex, Character: 0},
					End:   Position{Line: lineIndex, Character: len(leadingWhitespace)},
				}
				diagnostics = append(diagnostics, Diagnostic{
					Severity: DiagnosticSeverityWarning,
					Range:    r,
					Message:  "Mixed tabs and spaces in indentation. Use either tabs or spaces consistently.",
					Source:   "view.tree",
				})
			}
		}

		// Check for invalid binding syntax
		if strings.Contains(trimmed, "<=") || strings.Contains(trimmed, "<=>") {
			bindingMatch := regexp.MustCompile(`(<=?>?)\s*([a-zA-Z_$][a-zA-Z0-9_]*)?`).FindStringSubmatch(trimmed)
			if len(bindingMatch) >= 2 && (len(bindingMatch) < 3 || bindingMatch[2] == "") && bindingMatch[1] != "" {
				operatorIndex := strings.Index(line, bindingMatch[1])
				r := Range{
					Start: Position{Line: lineIndex, Character: operatorIndex},
					End:   Position{Line: lineIndex, Character: operatorIndex + len(bindingMatch[1])},
				}
				diagnostics = append(diagnostics, Diagnostic{
					Severity: DiagnosticSeverityError,
					Range:    r,
					Message:  "Binding operator must be followed by a property name.",
					Source:   "view.tree",
				})
			}
		}
	}

	return diagnostics
}

func (dp *DiagnosticProvider) validateComponents(components []ParsedComponent, documentURI string) []Diagnostic {
	var diagnostics []Diagnostic
	projectData := dp.projectScanner.GetProjectData()

	for _, component := range components {
		componentName := component.Name

		// Check if component exists in project
		projectData.mutex.RLock()
		hasComponent := projectData.Components[componentName]
		projectData.mutex.RUnlock()

		if !hasComponent && !strings.HasPrefix(componentName, "$mol_") {
			// Skip built-in $mol_ components for now
			diagnostics = append(diagnostics, Diagnostic{
				Severity: DiagnosticSeverityWarning,
				Range:    component.Range,
				Message:  fmt.Sprintf("Component '%s' not found in project. Consider defining it or check the spelling.", componentName),
				Source:   "view.tree",
			})
		}

		// Check for duplicate component definitions in same file
		duplicateCount := 0
		for _, otherComponent := range components {
			if otherComponent.Name == componentName {
				duplicateCount++
			}
		}

		if duplicateCount > 1 {
			// Mark all duplicates except the first one
			isFirst := true
			for _, otherComponent := range components {
				if otherComponent.Name == componentName {
					if isFirst {
						isFirst = false
						continue
					}
					diagnostics = append(diagnostics, Diagnostic{
						Severity: DiagnosticSeverityError,
						Range:    otherComponent.Range,
						Message:  fmt.Sprintf("Duplicate component definition: %s", componentName),
						Source:   "view.tree",
					})
				}
			}
		}
	}

	return diagnostics
}

func (dp *DiagnosticProvider) validateProperties(components []ParsedComponent, content string) []Diagnostic {
	var diagnostics []Diagnostic

	for _, component := range components {
		for _, property := range component.Properties {
			propertyName := property.Name

			// Check for invalid property names
			matched, _ := regexp.MatchString(`^[a-zA-Z_$][a-zA-Z0-9_?*]*$`, propertyName)
			if !matched {
				diagnostics = append(diagnostics, Diagnostic{
					Severity: DiagnosticSeverityError,
					Range:    property.Range,
					Message:  fmt.Sprintf("Invalid property name: %s. Property names must start with a letter, $, or underscore.", propertyName),
					Source:   "view.tree",
				})
			}

			// Check for reserved property names
			reservedNames := []string{"constructor", "prototype", "__proto__"}
			for _, reserved := range reservedNames {
				if propertyName == reserved {
					diagnostics = append(diagnostics, Diagnostic{
						Severity: DiagnosticSeverityError,
						Range:    property.Range,
						Message:  fmt.Sprintf("Reserved property name: %s. Choose a different name.", propertyName),
						Source:   "view.tree",
					})
					break
				}
			}

			// Check for duplicate properties within component
			duplicateCount := 0
			for _, otherProperty := range component.Properties {
				if otherProperty.Name == propertyName {
					duplicateCount++
				}
			}

			if duplicateCount > 1 {
				// Mark all duplicates except the first one
				isFirst := true
				for _, otherProperty := range component.Properties {
					if otherProperty.Name == propertyName {
						if isFirst {
							isFirst = false
							continue
						}
						diagnostics = append(diagnostics, Diagnostic{
							Severity: DiagnosticSeverityWarning,
							Range:    otherProperty.Range,
							Message:  fmt.Sprintf("Duplicate property: %s", propertyName),
							Source:   "view.tree",
						})
					}
				}
			}

			// Validate binding targets
			if property.IsBinding && property.Value != "" {
				bindingTarget := property.Value
				matched, _ := regexp.MatchString(`^[a-zA-Z_$][a-zA-Z0-9_?*]*$`, bindingTarget)
				if !matched {
					lines := strings.Split(content, "\n")
					if property.Line < len(lines) {
						line := lines[property.Line]
						bindingIndex := strings.Index(line, bindingTarget)
						if bindingIndex >= 0 {
							r := Range{
								Start: Position{Line: property.Line, Character: bindingIndex},
								End:   Position{Line: property.Line, Character: bindingIndex + len(bindingTarget)},
							}
							diagnostics = append(diagnostics, Diagnostic{
								Severity: DiagnosticSeverityError,
								Range:    r,
								Message:  fmt.Sprintf("Invalid binding target: %s", bindingTarget),
								Source:   "view.tree",
							})
						}
					}
				}
			}
		}
	}

	return diagnostics
}

func (dp *DiagnosticProvider) validateIndentation(content string) []Diagnostic {
	var diagnostics []Diagnostic
	lines := strings.Split(content, "\n")
	lastNonEmptyIndent := 0

	for lineIndex, line := range lines {
		if line == "" {
			continue
		}
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}

		currentIndent := dp.getIndentLevel(line)

		// Root level components should have no indentation
		if strings.HasPrefix(trimmed, "$") && currentIndent > 0 {
			r := Range{
				Start: Position{Line: lineIndex, Character: 0},
				End:   Position{Line: lineIndex, Character: currentIndent},
			}
			diagnostics = append(diagnostics, Diagnostic{
				Severity: DiagnosticSeverityError,
				Range:    r,
				Message:  "Component definitions should not be indented.",
				Source:   "view.tree",
			})
		}

		// Properties should be indented
		if !strings.HasPrefix(trimmed, "$") && currentIndent == 0 {
			r := Range{
				Start: Position{Line: lineIndex, Character: 0},
				End:   Position{Line: lineIndex, Character: 1},
			}
			diagnostics = append(diagnostics, Diagnostic{
				Severity: DiagnosticSeverityError,
				Range:    r,
				Message:  "Properties must be indented under their component.",
				Source:   "view.tree",
			})
		}

		// Check for excessive indentation jumps
		if currentIndent > lastNonEmptyIndent+1 {
			r := Range{
				Start: Position{Line: lineIndex, Character: 0},
				End:   Position{Line: lineIndex, Character: currentIndent},
			}
			diagnostics = append(diagnostics, Diagnostic{
				Severity: DiagnosticSeverityWarning,
				Range:    r,
				Message:  "Indentation increased by more than one level. This might indicate a structural issue.",
				Source:   "view.tree",
			})
		}

		if len(trimmed) > 0 {
			lastNonEmptyIndent = currentIndent
		}
	}

	return diagnostics
}

func (dp *DiagnosticProvider) validateBindings(content string) []Diagnostic {
	var diagnostics []Diagnostic
	lines := strings.Split(content, "\n")

	for lineIndex, line := range lines {
		if line == "" {
			continue
		}
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}

		// Check for malformed binding operators
		malformedBindings := []struct {
			pattern string
			message string
		}{
			{`[^<]=[^>]`, "Use <= or <=> for bindings, not ="},
			{`<[^=]`, "Incomplete binding operator. Use <= or <=>"},
			{`>[^=]`, "Invalid operator. Use <= or <=>"},
			{`<=\s*$`, "Binding operator <= must be followed by a property name"},
			{`<=>\s*$`, "Binding operator <=> must be followed by a property name"},
		}

		for _, check := range malformedBindings {
			matched, _ := regexp.MatchString(check.pattern, trimmed)
			if matched {
				re := regexp.MustCompile(check.pattern)
				match := re.FindString(trimmed)
				if match != "" {
					matchIndex := strings.Index(line, match)
					r := Range{
						Start: Position{Line: lineIndex, Character: matchIndex},
						End:   Position{Line: lineIndex, Character: matchIndex + len(match)},
					}
					diagnostics = append(diagnostics, Diagnostic{
						Severity: DiagnosticSeverityError,
						Range:    r,
						Message:  check.message,
						Source:   "view.tree",
					})
				}
			}
		}

		// Check for conflicting bindings - count actual distinct operators
		hasOneWayBinding := regexp.MustCompile(`[^<]<=\s`).MatchString(trimmed)
		hasTwoWayBinding := strings.Contains(trimmed, "<=>")
		
		if hasOneWayBinding && hasTwoWayBinding {
			r := Range{
				Start: Position{Line: lineIndex, Character: 0},
				End:   Position{Line: lineIndex, Character: len(line)},
			}
			diagnostics = append(diagnostics, Diagnostic{
				Severity: DiagnosticSeverityError,
				Range:    r,
				Message:  "Cannot use both <= and <=> operators in the same line.",
				Source:   "view.tree",
			})
		}
	}

	return diagnostics
}

func (dp *DiagnosticProvider) getIndentLevel(line string) int {
	indent := 0
	for _, char := range line {
		if char == '\t' {
			indent++
		} else if char == ' ' {
			indent++ // Could be adjusted for different space-to-tab ratios
		} else {
			break
		}
	}
	return indent
}

func (dp *DiagnosticProvider) mapSeverity(severity string) DiagnosticSeverity {
	switch severity {
	case "error":
		return DiagnosticSeverityError
	case "warning":
		return DiagnosticSeverityWarning
	case "info":
		return DiagnosticSeverityInformation
	default:
		return DiagnosticSeverityInformation
	}
}