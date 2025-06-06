# View.Tree LSP Server (Go Implementation)

A Language Server Protocol (LSP) implementation for the view.tree language, written in Go 1.24. This is a complete port of the TypeScript LSP server with identical functionality.

## Features

- **Syntax Highlighting Support**: Full parsing and validation of view.tree syntax
- **Auto-completion**: Context-aware completion for:
  - Component names (`$component_name`)
  - Property names based on current component context
  - Binding operators (`<=`, `<=>`, `^`)
  - Special values (`null`, `true`, `false`, `*`, `/`, etc.)
  - CSS classes and event handlers
- **Go-to-Definition**: Navigate to component and property definitions
- **Hover Information**: Rich hover tooltips with component and property documentation
- **Real-time Diagnostics**: Error checking and validation including:
  - Syntax errors
  - Invalid component/property names
  - Indentation issues
  - Binding validation
  - Duplicate definitions
- **Project-wide Analysis**: Scans `.view.tree` and `.ts` files for comprehensive project understanding

## Building

```bash
cd lsp
go mod tidy
go build
```

This will create the `lsp-view-tree` executable (or `lsp-view-tree.exe` on Windows).

## Usage

The LSP server communicates via stdin/stdout using the Language Server Protocol. It's designed to be used with editors and IDEs that support LSP.

### Running the Server

```bash
./lsp-view-tree
```

The server will:
1. Listen for LSP messages on stdin
2. Send responses via stdout
3. Log debug information to stderr

### Editor Integration

Configure your editor to use this LSP server for `.view.tree` files. The server supports:

- `textDocument/completion` - Auto-completion
- `textDocument/definition` - Go-to-definition
- `textDocument/hover` - Hover information
- `textDocument/publishDiagnostics` - Error reporting

## Architecture

The Go implementation follows the exact same architecture as the TypeScript version:

```
main.go                 -> Entry point
server.go              -> Main LSP server and protocol handling
project-scanner.go     -> Scans and indexes .view.tree and .ts files
view-tree-parser.go    -> Parses view.tree syntax and structure
completion-provider.go -> Provides auto-completion functionality
definition-provider.go -> Handles go-to-definition requests
hover-provider.go      -> Generates hover information
diagnostic-provider.go -> Validates code and reports errors
```

### Key Components

- **ProjectScanner**: Recursively scans the workspace for `.view.tree` and `.ts` files, extracting component definitions and properties
- **ViewTreeParser**: Parses view.tree syntax into structured AST nodes
- **Providers**: Implement specific LSP features using the parsed project data

## View.Tree Language Support

The server understands the complete view.tree syntax:

```tree
$my_component extends $parent_component
    dom_name \div
    attr *
        class \my-class
        id \element-id
    
    property_name <= bound_value
    two_way_property <=> bound_value
    override_property ^ parent_value
    
    sub /
        <= items
        $child_component
            value <= item_value
```

## Comparison with TypeScript Version

This Go implementation provides 100% feature parity with the TypeScript version:

- ✅ Identical LSP protocol support
- ✅ Same completion algorithms and triggers
- ✅ Equivalent parsing and validation logic
- ✅ Identical diagnostic messages and severity levels
- ✅ Same hover information formatting
- ✅ Compatible project scanning behavior

### Performance Characteristics

- **Memory Usage**: Generally lower memory footprint than Node.js version
- **Startup Time**: Faster cold start times
- **Concurrency**: Better handling of concurrent requests via Go's goroutines
- **Binary Size**: Single self-contained executable (~10-15MB)

## Development

### Project Structure

- All code is in the `main` package for simplicity
- LSP protocol structures are defined in `server.go`
- Each provider is in its own file with clear responsibilities
- Thread-safe project data structures with `sync.RWMutex`

### Adding Features

1. Define new LSP capabilities in `server.go`
2. Add request handlers in the main message router
3. Implement provider logic in appropriate files
4. Update project scanner if new file types are needed

### Debugging

The server logs extensively to stderr. Set log level with:

```go
log.SetFlags(log.LstdFlags | log.Lshortfile)
```

## License

Same license as the parent project (Apache 2.0).