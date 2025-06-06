package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNewServer(t *testing.T) {
	server := NewServer()
	if server == nil {
		t.Fatal("NewServer() returned nil")
	}
	
	if server.reader == nil {
		t.Error("Server reader is nil")
	}
	
	if server.writer == nil {
		t.Error("Server writer is nil")
	}
}

func TestUnmarshalParams(t *testing.T) {
	server := NewServer()
	
	// Test with valid params
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": "file:///test.view.tree",
		},
		"position": map[string]interface{}{
			"line": 5,
			"character": 10,
		},
	}
	
	var target TextDocumentPositionParams
	err := server.unmarshalParams(params, &target)
	if err != nil {
		t.Fatalf("unmarshalParams failed: %v", err)
	}
	
	if target.TextDocument.URI != "file:///test.view.tree" {
		t.Errorf("Expected URI 'file:///test.view.tree', got '%s'", target.TextDocument.URI)
	}
	
	if target.Position.Line != 5 {
		t.Errorf("Expected line 5, got %d", target.Position.Line)
	}
	
	if target.Position.Character != 10 {
		t.Errorf("Expected character 10, got %d", target.Position.Character)
	}
}

func TestPositionToOffset(t *testing.T) {
	server := NewServer()
	
	lines := []string{
		"$component",
		"\tproperty value",
		"\tsub /",
		"\t\titem",
	}
	
	// Test position at start of line 2
	pos := Position{Line: 2, Character: 0}
	offset := server.positionToOffset(lines, pos)
	expected := len("$component\n\tproperty value\n")
	if offset != expected {
		t.Errorf("Expected offset %d, got %d", expected, offset)
	}
	
	// Test position in middle of line 1
	pos = Position{Line: 1, Character: 5}
	offset = server.positionToOffset(lines, pos)
	expected = len("$component\n") + 5
	if offset != expected {
		t.Errorf("Expected offset %d, got %d", expected, offset)
	}
}

func TestApplyTextChange(t *testing.T) {
	server := NewServer()
	
	text := "$component\n\tproperty value\n\tsub /"
	changeRange := Range{
		Start: Position{Line: 1, Character: 1},
		End:   Position{Line: 1, Character: 9},
	}
	newText := "new_prop"
	
	result := server.applyTextChange(text, changeRange, newText)
	expected := "$component\n\tnew_prop value\n\tsub /"
	
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestURIConversion(t *testing.T) {
	server := NewServer()
	
	// Test URI to file path
	uri := "file:///path/to/file.view.tree"
	filePath := server.uriToFilePath(uri)
	expected := "/path/to/file.view.tree"
	if filePath != expected {
		t.Errorf("Expected '%s', got '%s'", expected, filePath)
	}
	
	// Test regular path
	regularPath := "/regular/path.view.tree"
	result := server.uriToFilePath(regularPath)
	if result != regularPath {
		t.Errorf("Expected '%s', got '%s'", regularPath, result)
	}
}

func TestProjectScannerBasic(t *testing.T) {
	scanner := NewProjectScanner(".")
	if scanner == nil {
		t.Fatal("NewProjectScanner returned nil")
	}
	
	if scanner.workspaceRoot != "." {
		t.Errorf("Expected workspace root '.', got '%s'", scanner.workspaceRoot)
	}
	
	projectData := scanner.GetProjectData()
	if projectData == nil {
		t.Error("GetProjectData returned nil")
	}
}

func TestParseViewTreeContent(t *testing.T) {
	scanner := NewProjectScanner(".")
	
	content := `$my_component
	property_name value
	binding_prop <= bound_value
	two_way_prop <=> bound_value`
	
	scanner.parseViewTreeFile(content, "/test/file.view.tree")
	
	// Check if component was parsed
	if !scanner.HasComponent("$my_component") {
		t.Error("Component $my_component not found")
	}
	
	// Check properties
	properties := scanner.GetPropertiesForComponent("$my_component")
	
	if len(properties) < 2 {
		t.Errorf("Expected at least 2 properties, got %d", len(properties))
	}
	
	// Check that we have property_name
	found := false
	for _, prop := range properties {
		if prop == "property_name" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Property 'property_name' not found")
	}
}

func TestViewTreeParser(t *testing.T) {
	parser := NewViewTreeParser()
	
	content := `$root_component
	property1 value1
	property2 <= binding

$child_component
	child_prop value`
	
	result := parser.Parse(content)
	
	// Check components
	if len(result.Components) != 2 {
		t.Errorf("Expected 2 components, got %d", len(result.Components))
	}
	
	if result.Components[0].Name != "$root_component" {
		t.Errorf("Expected first component '$root_component', got '%s'", result.Components[0].Name)
	}
	
	// Check properties
	if len(result.Components[0].Properties) < 2 {
		t.Errorf("Expected at least 2 properties for root component, got %d", len(result.Components[0].Properties))
	}
	
	// Check nodes
	if len(result.Nodes) == 0 {
		t.Error("No nodes parsed")
	}
}

func TestGetWordRangeAtPosition(t *testing.T) {
	parser := NewViewTreeParser()
	
	content := "$component_name\n\tproperty_value"
	
	// Test getting word at component name
	pos := Position{Line: 0, Character: 5}
	wordRange := parser.GetWordRangeAtPosition(content, pos)
	
	if wordRange == nil {
		t.Fatal("GetWordRangeAtPosition returned nil")
	}
	
	if wordRange.Start.Line != 0 || wordRange.Start.Character != 0 {
		t.Errorf("Expected start position (0,0), got (%d,%d)", 
			wordRange.Start.Line, wordRange.Start.Character)
	}
	
	if wordRange.End.Line != 0 || wordRange.End.Character != 15 {
		t.Errorf("Expected end position (0,15), got (%d,%d)", 
			wordRange.End.Line, wordRange.End.Character)
	}
}

func TestGetCurrentComponent(t *testing.T) {
	parser := NewViewTreeParser()
	
	content := `$main_component
	property1 value
	property2 <= binding
	
$second_component
	other_prop value`
	
	// Test position in first component
	pos := Position{Line: 2, Character: 5}
	component := parser.GetCurrentComponent(content, pos)
	if component != "$main_component" {
		t.Errorf("Expected '$main_component', got '%s'", component)
	}
	
	// Test position in second component
	pos = Position{Line: 5, Character: 5}
	component = parser.GetCurrentComponent(content, pos)
	if component != "$second_component" {
		t.Errorf("Expected '$second_component', got '%s'", component)
	}
}

func TestNestedComponentParsing(t *testing.T) {
	parser := NewViewTreeParser()
	
	content := `$my_app $mol_view
	sub /
		<= Button $mol_button_major
			title @ \Subscribe
			click? <=> submit? null
		<= Message $mol_status
			title @ \Status Message
	other_prop value`
	
	parseResult := parser.Parse(content)
	
	// Should have one root component
	if len(parseResult.Components) != 1 {
		t.Errorf("Expected 1 component, got %d", len(parseResult.Components))
	}
	
	rootComponent := parseResult.Components[0]
	if rootComponent.Name != "$my_app" {
		t.Errorf("Expected root component '$my_app', got '%s'", rootComponent.Name)
	}
	
	// Test that getCurrentComponent finds correct component for nested positions
	// Position at "title @ \Subscribe" should find $mol_button_major
	pos := Position{Line: 3, Character: 8}
	component := parser.GetCurrentComponent(content, pos)
	if component != "$mol_button_major" {
		t.Errorf("Expected '$mol_button_major' for nested position, got '%s'", component)
	}
	
	// Position at "title @ \Status Message" should find $mol_status
	pos = Position{Line: 6, Character: 8}
	component = parser.GetCurrentComponent(content, pos)
	if component != "$mol_status" {
		t.Errorf("Expected '$mol_status' for nested position, got '%s'", component)
	}
	
	// Position at "other_prop value" should find $my_app
	pos = Position{Line: 7, Character: 5}
	component = parser.GetCurrentComponent(content, pos)
	if component != "$my_app" {
		t.Errorf("Expected '$my_app' for root level property, got '%s'", component)
	}
}

func TestCompletionProvider(t *testing.T) {
	scanner := NewProjectScanner(".")
	provider := NewCompletionProvider(scanner)
	
	if provider == nil {
		t.Fatal("NewCompletionProvider returned nil")
	}
	
	// Add some test data
	scanner.parseViewTreeFile("$test_component\n\ttest_property value", "/test.view.tree")
	
	document := &TextDocument{
		URI:  "file:///test.view.tree",
		Text: "$test_component\n\t",
	}
	
	pos := Position{Line: 1, Character: 1}
	items, err := provider.ProvideCompletionItems(document, pos)
	
	if err != nil {
		t.Fatalf("ProvideCompletionItems failed: %v", err)
	}
	
	if len(items) == 0 {
		t.Error("No completion items returned")
	}
}

func TestDiagnosticProvider(t *testing.T) {
	scanner := NewProjectScanner(".")
	provider := NewDiagnosticProvider(scanner)
	
	if provider == nil {
		t.Fatal("NewDiagnosticProvider returned nil")
	}
	
	// Test valid content
	document := &TextDocument{
		URI:  "file:///test.view.tree",
		Text: "$valid_component\n\tvalid_property value",
	}
	
	diagnostics, err := provider.ProvideDiagnostics(document)
	if err != nil {
		t.Fatalf("ProvideDiagnostics failed: %v", err)
	}
	
	// Should have no diagnostics for valid content
	errorCount := 0
	for _, diag := range diagnostics {
		if diag.Severity == DiagnosticSeverityError {
			errorCount++
		}
	}
	
	if errorCount > 0 {
		t.Errorf("Expected no errors for valid content, got %d", errorCount)
	}
	
	// Test invalid content
	document.Text = "$invalid-component-name\n\t123invalid_property value"
	diagnostics, err = provider.ProvideDiagnostics(document)
	if err != nil {
		t.Fatalf("ProvideDiagnostics failed on invalid content: %v", err)
	}
	
	// Should have diagnostics for invalid content
	if len(diagnostics) == 0 {
		t.Error("Expected diagnostics for invalid content")
	}
}

func TestLSPMessageParsing(t *testing.T) {
	// Test valid LSP message
	msg := LSPMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]interface{}{
			"capabilities": map[string]interface{}{},
		},
	}
	
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal LSP message: %v", err)
	}
	
	var parsed LSPMessage
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Failed to unmarshal LSP message: %v", err)
	}
	
	if parsed.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC '2.0', got '%s'", parsed.JSONRPC)
	}
	
	if parsed.Method != "initialize" {
		t.Errorf("Expected method 'initialize', got '%s'", parsed.Method)
	}
}

func TestValidateSyntax(t *testing.T) {
	parser := NewViewTreeParser()
	
	// Test valid syntax
	validContent := "$component\n\tproperty value\n\tbinding <= bound"
	errors := parser.ValidateSyntax(validContent)
	
	errorCount := 0
	for _, err := range errors {
		if err.Severity == "error" {
			errorCount++
		}
	}
	
	if errorCount > 0 {
		t.Errorf("Expected no syntax errors for valid content, got %d", errorCount)
	}
	
	// Test invalid syntax - duplicate component
	invalidContent := "$component\n\tprop1 value\n$component\n\tprop2 value"
	errors = parser.ValidateSyntax(invalidContent)
	
	if len(errors) == 0 {
		t.Error("Expected syntax errors for duplicate components")
	}
}

func BenchmarkParseViewTree(b *testing.B) {
	parser := NewViewTreeParser()
	content := strings.Repeat("$component\n\tproperty value\n\tbinding <= bound\n", 100)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.Parse(content)
	}
}

func BenchmarkProjectScan(b *testing.B) {
	scanner := NewProjectScanner(".")
	content := strings.Repeat("$component\n\tproperty value\n", 50)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scanner.parseViewTreeFile(content, "/test.view.tree")
	}
}