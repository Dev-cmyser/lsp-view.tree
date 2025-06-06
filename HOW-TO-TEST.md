# How to Test the LSP View.Tree Server

This guide provides step-by-step instructions for testing the LSP server functionality.

## ✅ Quick Verification

First, ensure the server builds and starts correctly:

```bash
# 1. Build the project
npm run build

# 2. Quick connectivity test (30 seconds)
npm run test:simple
```

**Expected output:**
```
✅ Server initialized with capabilities: [
  'textDocumentSync',
  'completionProvider', 
  'definitionProvider',
  'hoverProvider',
  'diagnosticProvider'
]
```

If you see this, your LSP server is working! 🎉

## 🧪 Testing Methods

### 1. Automated Testing (Recommended)

```bash
npm test
```

Tests all LSP features programmatically:
- ✅ Server initialization
- ✅ Document handling
- ✅ Auto-completion
- ✅ Hover information
- ✅ Go-to-definition
- ✅ Project scanning

### 2. Interactive Testing (Best for Development)

```bash
npm run test:interactive
```

Provides an interactive menu to test specific features:

```
📋 LSP Test Menu:
1. Test completion
2. Test hover  
3. Test go-to-definition
4. Open document
5. Create sample document
6. Show server status
7. Exit
```

**Recommended workflow:**
1. Choose option 5 (Create sample document)
2. Select sample 1 (Basic component)  
3. Test completion at position 0:1 (should show $mol_ components)
4. Test hover on component names
5. Test go-to-definition

### 3. Manual Testing with Real Editors

#### VS Code (Easiest)

1. Install a generic LSP extension like "LSP Client"
2. Configure it to use our server:

```json
{
  "lsp-client.servers": {
    "view-tree": {
      "command": ["node", "/path/to/lsp-view.tree/lib/server.js", "--stdio"],
      "filetypes": ["tree"],
      "rootPatterns": [".git", "package.json"]
    }
  }
}
```

3. Create a test file `test.view.tree`:
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

4. Test features:
   - Type `$` → Should show completions
   - Hover over `$my_app` → Should show component info
   - Ctrl+click on components → Should navigate

#### Vim/Neovim with coc.nvim

Add to `~/.config/nvim/coc-settings.json`:

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

#### Command Line Testing

For manual LSP protocol testing:

```bash
# Start server
node lib/server.js --stdio

# Send raw LSP messages (copy each block separately):

# 1. Initialize
Content-Length: 300

{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"processId":null,"capabilities":{"textDocument":{"completion":{},"hover":{},"definition":{}}},"workspaceFolders":[{"uri":"file:///tmp","name":"test"}]}}

# 2. Initialized notification  
Content-Length: 52

{"jsonrpc":"2.0","method":"initialized","params":{}}

# 3. Open document
Content-Length: 250

{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///test.view.tree","languageId":"tree","version":1,"text":"$my_app $mol_page\n\ttitle \\Hello\n\tbody /\n\t\t<= content"}}}

# 4. Request completion
Content-Length: 157

{"jsonrpc":"2.0","id":2,"method":"textDocument/completion","params":{"textDocument":{"uri":"file:///test.view.tree"},"position":{"line":0,"character":1}}}
```

## 🎯 What to Test

### Auto-completion

**Test positions:**
- `$|` (start of component) → Should show all components
- `	|` (indented property) → Should show properties  
- `	<= |` (binding) → Should show property names
- `	attr *\n		|` → Should show attribute names

**Expected results:**
- Components: `$mol_page`, `$mol_view`, `$mol_text`, etc.
- Properties: `title`, `body`, `sub`, `attr`, `dom_name`, etc.
- Values: `null`, `true`, `false`, `\`, `@\`, `/`, `*`
- Operators: `<=`, `<=>`, `^`

### Hover Information

**Test on:**
- Component names (`$my_app`) → Component documentation
- Property names (`title`) → Property type and description  
- CSS classes → CSS rules if `.css.ts` exists

**Expected format:**
```markdown
**Component**: `$my_app`

**File**: `path/to/file.view.tree`

**Properties**:
- `title`
- `body`
- `content`

**Usage**:
```tree
$my_app
	property <= value
```

### Go-to-Definition

**Test on:**
- Root component → Should go to `.ts` file
- Component references → Should go to `.view.tree` file
- CSS classes → Should go to `.css.ts` file
- Properties → Should go to TypeScript property

### Diagnostics

**Create files with errors:**

```tree
# Syntax errors
$invalid-name $mol_page  # Invalid component name
	title               # Missing value
	<= orphan_binding   # Binding without property
		mixed	tabs and spaces  # Indentation error

# Duplicate properties  
$my_app $mol_page
	title \First
	title \Duplicate  # Should show warning
```

**Expected diagnostics:**
- ❌ Syntax errors (red squiggles)
- ⚠️ Warnings (yellow squiggles)  
- ℹ️ Info messages (blue squiggles)

## 🐛 Troubleshooting

### Server Won't Start

```bash
# Check Node.js version
node --version  # Should be 18.0.0+

# Check build
npm run build  # Should complete without errors

# Check dependencies
npm install    # Should install without issues

# Manual start
node lib/server.js --stdio  # Should wait for input
```

### No Completions/Features

1. **Check file extension:** Must be `.view.tree`
2. **Check language mode:** Should be set to `tree`
3. **Check workspace:** LSP needs workspace folder
4. **Check logs:** Look for error messages

### Debug Mode

Enable detailed logging:

```bash
NODE_ENV=development node lib/server.js --stdio
```

### Performance Issues

For large projects:

1. **Limit file scanning:** Modify `maxTsFiles` in configuration
2. **Check file count:** 
   ```bash
   find . -name "*.view.tree" | wc -l
   find . -name "*.ts" | wc -l
   ```
3. **Profile startup:** Time the initialization

## 📊 Test Results Interpretation

### ✅ Success Indicators

- Server starts without errors
- Initialization completes with all capabilities
- Completions return relevant items
- Hover shows meaningful information
- Go-to-definition finds targets
- Diagnostics detect real issues

### ❌ Common Issues

**Empty completions:** 
- Check if workspace contains `.view.tree` files
- Verify document is opened in LSP client

**No hover information:**
- Ensure corresponding `.ts` files exist
- Check component names match file structure

**Go-to-definition fails:**
- Verify file paths and naming conventions
- Check workspace root is set correctly

## 🚀 Integration Testing

### With Real $mol Project

1. Clone a $mol project with `.view.tree` files
2. Start LSP server in project root
3. Test on real components and properties
4. Verify cross-references work

### Performance Benchmarks

- **Startup time:** < 2 seconds for typical project
- **Completion response:** < 200ms
- **File scanning:** < 5 seconds for 100+ files
- **Memory usage:** < 100MB for typical project

## 📝 Reporting Issues

When reporting problems, include:

1. **Server logs:** Run with `NODE_ENV=development`
2. **Sample files:** Minimal `.view.tree` that reproduces issue
3. **LSP client:** Which editor/extension you're using  
4. **System info:** Node.js version, OS, etc.
5. **Expected vs actual:** What should happen vs what happens

## 🎓 Next Steps

Once basic testing works:

1. **Integrate with your preferred editor**
2. **Test with real $mol projects** 
3. **Customize configuration** (`.view-tree-lsp.json`)
4. **Set up continuous integration**
5. **Contribute improvements** to the LSP server

---

**💡 Pro Tip:** Start with `npm run test:interactive` and option 5 to create a sample document, then test all features step by step. This gives you a good feel for what the LSP server can do!