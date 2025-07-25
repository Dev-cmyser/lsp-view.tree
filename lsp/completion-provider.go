package main

import (
	"fmt"
	"log"
	"strings"
)

type InternalCompletionContext struct {
	Type             string // "component_name", "component_extends", "property_name", "property_binding", "value"
	IndentLevel      int
	CurrentComponent string
}

type CompletionProvider struct {
	projectScanner *ProjectScanner
	parser         *ViewTreeParser
}

func NewCompletionProvider(projectScanner *ProjectScanner) *CompletionProvider {
	return &CompletionProvider{
		projectScanner: projectScanner,
		parser:         NewViewTreeParser(),
	}
}

func (cp *CompletionProvider) ProvideCompletionItems(document *TextDocument, position Position) ([]CompletionItem, error) {
	log.Printf("[completion] Request at %d:%d", position.Line, position.Character)

	content := document.Text
	lines := strings.Split(content, "\n")

	if position.Line >= len(lines) {
		return []CompletionItem{}, nil
	}

	line := lines[position.Line]
	beforeCursor := ""
	if position.Character <= len(line) {
		beforeCursor = line[:position.Character]
	}

	log.Printf("[completion] Line: \"%s\", Before cursor: \"%s\"", line, beforeCursor)

	var items []CompletionItem
	completionContext := cp.getCompletionContext(content, position, beforeCursor)
	log.Printf("[completion] Context: %s, indent: %d", completionContext.Type, completionContext.IndentLevel)

	switch completionContext.Type {
	case "component_name":
		log.Println("[completion] Adding component completions")
		cp.addComponentCompletions(&items)
	case "component_extends":
		log.Println("[completion] Adding component extends completions")
		cp.addComponentCompletions(&items)
	case "property_name":
		log.Printf("[completion] Adding property completions for component: %s", completionContext.CurrentComponent)
		cp.addPropertyCompletions(&items, completionContext.CurrentComponent)
	case "property_binding":
		log.Println("[completion] Adding binding completions")
		cp.addBindingCompletions(&items)
	case "value":
		log.Println("[completion] Adding value completions")
		cp.addValueCompletions(&items)
	}

	log.Printf("[completion] Returning %d items", len(items))
	return items, nil
}

func (cp *CompletionProvider) getCompletionContext(content string, position Position, beforeCursor string) InternalCompletionContext {
	trimmed := strings.TrimSpace(beforeCursor)
	// Match reference logic: beforeCursor.length - beforeCursor.trimStart().length
	trimStart := strings.TrimLeftFunc(beforeCursor, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
	indentLevel := len(beforeCursor) - len(trimStart)

	// If starts with $ anywhere - it's a component
	if strings.HasPrefix(trimmed, "$") {
		return InternalCompletionContext{Type: "component_name", IndentLevel: indentLevel, CurrentComponent: ""}
	}

	// If at root level and no space - it's a component
	if indentLevel == 0 && !strings.Contains(trimmed, " ") {
		return InternalCompletionContext{Type: "component_name", IndentLevel: indentLevel, CurrentComponent: ""}
	}

	// If at root level and has space - it's inheritance
	if indentLevel == 0 && strings.Contains(trimmed, " ") {
		return InternalCompletionContext{Type: "component_extends", IndentLevel: indentLevel, CurrentComponent: ""}
	}

	// If has binding operators
	if strings.Contains(trimmed, "<=") {
		return InternalCompletionContext{Type: "property_binding", IndentLevel: indentLevel, CurrentComponent: ""}
	}

	// If indented - it's a property
	if indentLevel > 0 {
		currentComponent := cp.getCurrentComponent(content, position)
		return InternalCompletionContext{Type: "property_name", IndentLevel: indentLevel, CurrentComponent: currentComponent}
	}

	return InternalCompletionContext{Type: "value", IndentLevel: indentLevel, CurrentComponent: ""}
}

func (cp *CompletionProvider) getCurrentComponent(content string, position Position) string {
	return cp.parser.GetCurrentComponent(content, position)
}

func (cp *CompletionProvider) addComponentCompletions(items *[]CompletionItem) {
	projectData := cp.projectScanner.GetProjectData()

	projectData.mutex.RLock()
	componentCount := len(projectData.Components)
	projectData.mutex.RUnlock()

	log.Printf("[completion] Project has %d components", componentCount)

	// Add components from project
	projectData.mutex.RLock()
	for component := range projectData.Components {
		item := CompletionItem{
			Label:         component,
			Kind:          CompletionItemKindClass,
			InsertText:    component,
			SortText:      "1" + component,
			Detail:        "Component",
			Documentation: fmt.Sprintf("Component: %s", component),
		}
		*items = append(*items, item)
	}
	projectData.mutex.RUnlock()

	log.Printf("[completion] Added %d component completions", componentCount)
}

func (cp *CompletionProvider) addPropertyCompletions(items *[]CompletionItem, currentComponent string) {
	projectData := cp.projectScanner.GetProjectData()

	// Add properties for current component
	if currentComponent != "" {
		projectData.mutex.RLock()
		if properties, exists := projectData.ComponentProperties[currentComponent]; exists {
			for property := range properties {
				item := CompletionItem{
					Label:         property,
					Kind:          CompletionItemKindProperty,
					InsertText:    property,
					SortText:      "1" + property,
					Detail:        fmt.Sprintf("Property of %s", currentComponent),
					Documentation: fmt.Sprintf("Property from component %s", currentComponent),
				}
				*items = append(*items, item)
			}
		}
		projectData.mutex.RUnlock()
	}

	// Add common properties if component not found
	if currentComponent == "" {
		allProperties := cp.projectScanner.GetAllProperties()
		for _, property := range allProperties {
			item := CompletionItem{
				Label:         property,
				Kind:          CompletionItemKindProperty,
				InsertText:    property,
				SortText:      "2" + property,
				Detail:        "Property",
				Documentation: "Property from project",
			}
			*items = append(*items, item)
		}
	}

	// Add list marker
	listItem := CompletionItem{
		Label:         "/",
		Kind:          CompletionItemKindOperator,
		InsertText:    "/",
		SortText:      "0/",
		Detail:        "Empty list",
		Documentation: "Creates an empty list",
	}
	*items = append(*items, listItem)
}

func (cp *CompletionProvider) addBindingCompletions(items *[]CompletionItem) {
	operators := []struct {
		text          string
		detail        string
		documentation string
	}{
		{"<=", "One-way binding", "Binds property value from parent to child (one direction)"},
		{"<=>", "Two-way binding", "Binds property value between parent and child (both directions)"},
		{"^", "Override", "Overrides property in parent class"},
		{"*", "Multi-property marker", "Marks property as accepting multiple values"},
	}

	for _, op := range operators {
		item := CompletionItem{
			Label:         op.text,
			Kind:          CompletionItemKindOperator,
			InsertText:    op.text,
			SortText:      "0" + op.text,
			Detail:        op.detail,
			Documentation: op.documentation,
		}
		*items = append(*items, item)
	}
}

func (cp *CompletionProvider) addValueCompletions(items *[]CompletionItem) {
	specialValues := []struct {
		text          string
		detail        string
		insertText    string
		documentation string
	}{
		{"null", "Null value", "null", "Represents empty/null value"},
		{"true", "Boolean true", "true", "Boolean true value"},
		{"false", "Boolean false", "false", "Boolean false value"},
		{"\\", "String literal", "\\\n\t\\", "Multi-line string literal"},
		{"@\\", "Localized string", "@\\\n\t\\", "Localized multi-line string"},
		{"*", "Dictionary marker", "*", "Marks property as dictionary"},
	}

	for _, value := range specialValues {
		insertText := value.insertText
		if insertText == "" {
			insertText = value.text
		}

		item := CompletionItem{
			Label:         value.text,
			Kind:          CompletionItemKindValue,
			InsertText:    insertText,
			SortText:      "0" + value.text,
			Detail:        value.detail,
			Documentation: value.documentation,
		}

		if strings.Contains(insertText, "\n") {
			item.InsertTextFormat = InsertTextFormatSnippet
		}

		*items = append(*items, item)
	}

	cp.addComponentCompletions(items)
}
