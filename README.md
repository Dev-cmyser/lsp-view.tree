# LSP View.Tree Server

Language Server Protocol implementation for `.view.tree` files used in the $mol framework.

## Features

- **Syntax Highlighting Support**: Provides language server capabilities for `.view.tree` files
- **Auto-completion**: Intelligent completion for components, properties, and values
- **Go to Definition**: Navigate to component and property definitions
- **Hover Information**: Rich hover tooltips with documentation
- **Diagnostics**: Real-time error detection and warnings
- **Incremental Parsing**: Efficient file watching and parsing
- **Project-wide Analysis**: Scans entire workspace for components and properties

## Installation

### Prerequisites

- Node.js 18.0.0 or higher
- npm or yarn package manager

### From Source

```bash
git clone <repository-url>
cd lsp-view.tree
npm install
npm run build
```

## Usage

### With VS Code

Create a VS Code extension or configure a generic LSP client:

```json
{
  "contributes": {
    "languages": [{
      "id": "tree",
      "aliases": ["View Tree", "tree"],
      "extensions": [".view.tree"],
      "configuration": "./language-configuration.json"
    }]
  }
}
```

### With Vim/Neovim (using coc.nvim)

Add to your `coc-settings.json`:

```json
{
  "languageserver": {
    "view-tree": {
      "command": "node",
      "args": ["/path/to/lsp-view.tree/lib/server.js", "--stdio"],
      "filetypes": ["tree"],
      "rootPatterns": [".git", "package.json"]
    }
  }
}
```

### With Vim/Neovim (using built-in LSP)

```lua
local lspconfig = require('lspconfig')

local configs = require('lspconfig.configs')
if not configs.viewtree then
  configs.viewtree = {
    default_config = {
      cmd = {'node', '/path/to/lsp-view.tree/lib/server.js', '--stdio'},
      filetypes = {'tree'},
      root_dir = lspconfig.util.root_pattern('.git', 'package.json'),
      single_file_support = true,
    },
  }
end

lspconfig.viewtree.setup{}
```

### With Emacs (using lsp-mode)

```elisp
(add-to-list 'lsp-language-id-configuration '(tree-mode . "tree"))

(lsp-register-client
 (make-lsp-client :new-connection (lsp-stdio-connection '("node" "/path/to/lsp-view.tree/lib/server.js" "--stdio"))
                  :major-modes '(tree-mode)
                  :server-id 'view-tree-ls))
```

### With Sublime Text (using LSP package)

Add to your LSP settings:

```json
{
  "clients": {
    "view-tree": {
      "enabled": true,
      "command": ["node", "/path/to/lsp-view.tree/lib/server.js", "--stdio"],
      "selector": "source.tree"
    }
  }
}
```

### Standalone Usage

```bash
# Start the server
npm start

# Or run directly
node lib/server.js --stdio
```

## Configuration

The LSP server automatically detects `.view.tree` and `.ts` files in your workspace. No additional configuration is required for basic functionality.

### Advanced Configuration

You can customize the behavior by creating a `.view-tree-lsp.json` file in your workspace root:

```json
{
  "maxTsFiles": 100,
  "diagnostics": {
    "enabled": true,
    "validateIndentation": true,
    "validateBindings": true
  },
  "completion": {
    "maxSuggestions": 50,
    "includeBuiltins": true
  }
}
```

## Language Features

### Auto-completion

The server provides intelligent completion for:

- **Components**: All `$` prefixed components found in the project
- **Properties**: Context-aware property suggestions
- **Binding Operators**: `<=`, `<=>`, `^`, `*`
- **Special Values**: `null`, `true`, `false`, `\`, `@\`, `/`
- **Common Properties**: `dom_name`, `attr`, `field`, `value`, `sub`, etc.

### Go to Definition

Navigate to definitions for:

- **Root Classes**: Jump to corresponding `.ts` file
- **Components**: Navigate to component `.view.tree` file
- **CSS Classes**: Jump to `.css.ts` file definitions
- **Properties**: Navigate to property definitions in TypeScript

### Hover Information

Rich hover tooltips provide:

- Component documentation and file location
- Property type information and descriptions
- CSS class definitions and rules
- Usage examples and syntax help

### Diagnostics

Real-time error detection for:

- **Syntax Errors**: Invalid component names, malformed bindings
- **Indentation Issues**: Mixed tabs/spaces, incorrect nesting
- **Duplicate Definitions**: Duplicate components or properties
- **Missing Files**: Missing TypeScript implementations
- **Invalid Bindings**: Malformed binding operators

## File Structure

```
.view.tree files structure:
$component_name $parent_component
    property_name value
    binding_property <= source_property
    two_way_binding <=> target_property
    sub_component $other_component
        nested_property value
```

### Supported Syntax

- **Components**: `$component_name`
- **Inheritance**: `$child_component $parent_component`
- **Properties**: `property_name value`
- **One-way Binding**: `property <= source`
- **Two-way Binding**: `property <=> target`
- **Override**: `property ^ value`
- **Lists**: `/` (empty list)
- **Dictionaries**: `*` (key-value pairs)
- **Strings**: `\` and `@\` (localized)
- **Booleans**: `true`, `false`
- **Null**: `null`

## Development

### Project Structure

```
src/
├── server.ts              # Main LSP server
├── project-scanner.ts     # Project file scanning
├── view-tree-parser.ts    # .view.tree syntax parser
├── completion-provider.ts # Auto-completion logic
├── definition-provider.ts # Go-to-definition logic
├── hover-provider.ts      # Hover information
└── diagnostic-provider.ts # Error detection
```

### Building

```bash
# Install dependencies
npm install

# Build TypeScript
npm run build

# Watch mode for development
npm run watch

# Start development server
npm run dev
```

### Testing

```bash
# Run tests (if available)
npm test

# Manual testing with sample files
echo '$my_component $mol_view
    title \Hello World
    sub /
        $mol_button
            title \Click me' > test.view.tree
```

## API

### LSP Methods Supported

- `textDocument/completion` - Auto-completion
- `textDocument/definition` - Go to definition
- `textDocument/hover` - Hover information
- `textDocument/publishDiagnostics` - Error reporting
- `textDocument/didOpen` - Document opened
- `textDocument/didChange` - Document changed
- `textDocument/didClose` - Document closed

### Initialization

The server supports standard LSP initialization with workspace folder detection and capability negotiation.

## Troubleshooting

### Common Issues

**Server not starting**
- Check Node.js version (requires 18.0.0+)
- Verify file paths in configuration
- Check server logs for error messages

**No completions showing**
- Ensure file extension is `.view.tree`
- Check that workspace contains `.view.tree` or `.ts` files
- Verify language mode is set correctly in your editor

**Go to definition not working**
- Ensure corresponding `.ts` files exist
- Check file naming conventions match component names
- Verify workspace root is set correctly

**Diagnostics not appearing**
- Check if diagnostics are enabled in client
- Verify file is saved (some clients only show diagnostics on save)
- Check server console for parsing errors

### Debug Mode

Start the server with debug logging:

```bash
NODE_ENV=development node lib/server.js --stdio
```

### Log Files

Server logs are written to the LSP client's output channel. Check your editor's LSP logs for detailed information.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

### Code Style

- Use TypeScript strict mode
- Follow existing code formatting
- Add JSDoc comments for public APIs
- Use meaningful variable and function names

## License

MIT License - see LICENSE file for details.

## Related Projects

- [$mol Framework](https://github.com/hyoo-ru/mam_mol) - The framework this LSP server supports
- [Tree Language Specification](https://github.com/hyoo-ru/mam_mol/tree/master/tree) - Official .view.tree format documentation

## Changelog

### v1.0.0
- Initial release
- Basic LSP functionality
- Auto-completion support
- Go to definition
- Hover information
- Diagnostics
- Project-wide scanning