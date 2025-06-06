# Testing Guide for LSP View.Tree Server

This guide explains how to test the LSP View.Tree server functionality.

## Quick Start

```bash
# Build the project
npm run build

# Run automated tests
npm test

# Run simple connectivity test
npm run test:simple
```

## Testing Methods

### 1. Automated Testing (Recommended)

The automated test client thoroughly tests all LSP features:

```bash
npm test
```

**What it tests:**
- ✅ Server initialization and capabilities
- ✅ Document opening and processing
- ✅ Auto-completion at various positions
- ✅ Hover information
- ✅ Go-to-definition functionality
- ✅ Project scanning and component detection

**Expected output:**
```
🧪 LSP Server Test Client

🚀 Starting LSP server...
🔧 Initializing LSP server...
✅ Server capabilities: textDocumentSync, completionProvider, definitionProvider, hoverProvider
📄 Opening document: file:///path/to/test.view.tree

🧪 Running LSP feature tests...

🔍 Testing completion at 0:1...
✅ Completion items: 15
   First item: "$mol_page" (7)
   Second item: "$mol_view" (7)

💡 Testing hover at 0:5...
✅ Hover content available
   Preview: "**Component**: `$my_test_app`..."

🎯 Testing go-to-definition at 0:5...
✅ Found 1 definition(s)

🎉 All tests completed!
```

### 2. Simple Connectivity Test

Basic test to verify server startup and initialization:

```bash
npm run test:simple
```

This sends basic LSP messages and verifies the server responds correctly.

### 3. Manual Testing with VS Code

#### Setup VS Code Extension

1. Create a simple VS Code extension for testing:

```json
// package.json for VS Code extension
{
  "name": "view-tree-test",
  "version": "1.0.0",
  "engines": { "vscode": "^1.74.0" },
  "activationEvents": ["onLanguage:tree"],
  "main": "./extension.js",
  "contributes": {
    "languages": [{
      "id": "tree",
      "aliases": ["View Tree"],
      "extensions": [".view.tree"]
    }]
  }
}
```

```javascript
// extension.js
const vscode = require('vscode');
const { LanguageClient } = require('vscode-languageclient/node');

function activate(context) {
    const serverOptions = {
        command: 'node',
        args: ['/path/to/lsp-view.tree/lib/server.js', '--stdio']
    };

    const clientOptions = {
        documentSelector: [{ scheme: 'file', language: 'tree' }]
    };

    const client = new LanguageClient('viewtree', 'View Tree LSP', serverOptions, clientOptions);
    client.start();
}

module.exports = { activate };
```

#### Test Features

1. Create a `.view.tree` file
2. Test auto-completion: Type `$` and see component suggestions
3. Test hover: Hover over component names
4. Test go-to-definition: Ctrl+click on components

### 4. Manual Testing with Vim/Neovim

#### Using coc.nvim

Add to `coc-settings.json`:
```json
{
  "languageserver": {
    "viewtree": {
      "command": "node",
      "args": ["/path/to/lsp-view.tree/lib/server.js", "--stdio"],
      "filetypes": ["tree"]
    }
  }
}
```

#### Using built-in LSP (Neovim)

```lua
local lspconfig = require('lspconfig')
local configs = require('lspconfig.configs')

configs.viewtree = {
  default_config = {
    cmd = {'node', '/path/to/lsp-view.tree/lib/server.js', '--stdio'},
    filetypes = {'tree'},
    root_dir = lspconfig.util.root_pattern('.git', 'package.json'),
  },
}

lspconfig.viewtree.setup{}
```

### 5. Command Line Testing

#### Direct Server Communication

```bash
# Start server
node lib/server.js --stdio

# Send LSP messages manually (copy-paste each block)
```

**Initialize:**
```json
Content-Length: 245

{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"processId":null,"capabilities":{"textDocument":{"completion":{},"hover":{},"definition":{}}},"workspaceFolders":[{"uri":"file:///tmp","name":"test"}]}}
```

**Completion request:**
```json
Content-Length: 157

{"jsonrpc":"2.0","id":2,"method":"textDocument/completion","params":{"textDocument":{"uri":"file:///test.view.tree"},"position":{"line":0,"character":1}}}
```

## Test Files

### Sample .view.tree Files

Create these test files to verify different features:

**basic.view.tree:**
```tree
$my_app $mol_page
	title \My Application
	body /
		<= content
	
	content $mol_view
		sub /
			<= welcome_text
	
	welcome_text $mol_text
		text \Hello World
```

**complex.view.tree:**
```tree
$complex_component $mol_view
	dom_name \section
	attr *
		class \complex-section
		data-test \true
	
	sub /
		<= header
		<= main_content
		<= footer
	
	header $mol_view
		dom_name \header
		sub /
			<= title_text
	
	title_text $mol_text
		text <= page_title
	
	main_content $mol_list
		rows /
			<= items
	
	footer $mol_view
		dom_name \footer
		visible <= show_footer
```

## Expected Behavior

### Auto-completion

- **Component names**: Should suggest all `$` prefixed components
- **Properties**: Should suggest relevant properties like `title`, `body`, `sub`, etc.
- **Values**: Should suggest `null`, `true`, `false`, `\`, `@\`, `/`, `*`
- **Bindings**: Should suggest `<=`, `<=>`, `^`

### Hover Information

- **Components**: Should show component info, file location, properties
- **Properties**: Should show property type and description
- **CSS classes**: Should show CSS rules if `.css.ts` file exists

### Go-to-Definition

- **Components**: Should navigate to `.view.tree` or `.ts` file
- **Properties**: Should navigate to TypeScript property definition
- **CSS classes**: Should navigate to `.css.ts` file

### Diagnostics

- **Syntax errors**: Invalid component names, malformed bindings
- **Indentation**: Mixed tabs/spaces, incorrect nesting
- **Duplicates**: Duplicate components or properties
- **Missing files**: Warnings about missing TypeScript files

## Troubleshooting

### Common Issues

**Server not starting:**
- Check Node.js version (requires 18.0.0+)
- Verify `npm run build` completed successfully
- Check file permissions

**No completions:**
- Ensure file has `.view.tree` extension
- Check server is receiving requests in logs
- Verify workspace contains parseable files

**No hover/definitions:**
- Check corresponding `.ts` files exist
- Verify file paths are correct
- Check server logs for parsing errors

### Debug Mode

Enable debug logging:
```bash
NODE_ENV=development node lib/server.js --stdio
```

### Performance Testing

Test with large projects:
```bash
# Create many test files
for i in {1..100}; do
  echo "\$test_component_$i \$mol_view" > test_$i.view.tree
done

# Run completion test
npm test
```

## Continuous Integration

Add to your CI pipeline:

```yaml
# .github/workflows/test.yml
name: Test LSP Server
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
        with:
          node-version: '18'
      - run: npm install
      - run: npm test
```

## Contributing Tests

When adding new features:

1. Add test cases to `test-client.js`
2. Create sample `.view.tree` files
3. Update this testing guide
4. Verify all existing tests still pass

For bug reports, include:
- Sample `.view.tree` file that reproduces the issue
- Expected vs actual behavior
- Server logs with debug enabled