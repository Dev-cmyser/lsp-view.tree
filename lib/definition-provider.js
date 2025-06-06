"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || (function () {
    var ownKeys = function(o) {
        ownKeys = Object.getOwnPropertyNames || function (o) {
            var ar = [];
            for (var k in o) if (Object.prototype.hasOwnProperty.call(o, k)) ar[ar.length] = k;
            return ar;
        };
        return ownKeys(o);
    };
    return function (mod) {
        if (mod && mod.__esModule) return mod;
        var result = {};
        if (mod != null) for (var k = ownKeys(mod), i = 0; i < k.length; i++) if (k[i] !== "default") __createBinding(result, mod, k[i]);
        __setModuleDefault(result, mod);
        return result;
    };
})();
Object.defineProperty(exports, "__esModule", { value: true });
exports.DefinitionProvider = void 0;
const node_1 = require("vscode-languageserver/node");
const vscode_uri_1 = require("vscode-uri");
const fs = __importStar(require("fs"));
const path = __importStar(require("path"));
const view_tree_parser_1 = require("./view-tree-parser");
class DefinitionProvider {
    constructor(projectScanner) {
        this.projectScanner = projectScanner;
        this.parser = new view_tree_parser_1.ViewTreeParser();
    }
    async provideDefinition(document, position) {
        const content = document.getText();
        const wordRange = this.parser.getWordRangeAtPosition(content, position);
        if (!wordRange) {
            return [];
        }
        const nodeName = content.substring(this.getOffset(content, wordRange.start), this.getOffset(content, wordRange.end));
        if (!nodeName) {
            return [];
        }
        const nodeType = this.getNodeType(content, position, wordRange);
        const documentUri = vscode_uri_1.URI.parse(document.uri);
        switch (nodeType) {
            case 'root_class':
                return await this.findRootClassDefinition(documentUri, nodeName);
            case 'class':
                return await this.findClassDefinition(nodeName);
            case 'comp':
                return await this.findCompDefinition(documentUri, nodeName);
            case 'prop':
                return await this.findPropDefinition(documentUri, nodeName);
            case 'sub_prop':
                return await this.findSubPropDefinition(documentUri, position, nodeName);
            default:
                return [];
        }
    }
    getNodeType(content, position, wordRange) {
        // Root class - first line, first character after $
        if (wordRange.start.character === 1 && wordRange.start.line === 0) {
            return 'root_class';
        }
        const lines = content.split('\n');
        const line = lines[position.line];
        if (!line) {
            return 'sub_prop';
        }
        // Check if preceded by $
        const firstChar = wordRange.start.character > 0 ?
            line[wordRange.start.character - 1] : '';
        if (firstChar === '$') {
            return 'class';
        }
        // Property at root level (character 1)
        if (wordRange.start.character === 1) {
            return 'prop';
        }
        // Check for binding operators before the word
        const beforeWord = line.substring(0, wordRange.start.character);
        if (/[>=^]\s*$/.test(beforeWord)) {
            return 'prop';
        }
        // Default to sub_prop for deeper nested items
        return 'sub_prop';
    }
    async findRootClassDefinition(documentUri, nodeName) {
        // Find corresponding .ts file
        const tsPath = documentUri.fsPath.replace(/\.view\.tree$/, '.ts');
        const tsUri = vscode_uri_1.URI.file(tsPath);
        try {
            // Check if .ts file exists
            await fs.promises.access(tsPath);
            // Try to find class symbol in .ts file
            const classLocation = await this.findClassSymbolInFile(tsUri, '$' + nodeName);
            if (classLocation) {
                return [classLocation];
            }
            // If no specific class found, return beginning of file
            const locationRange = node_1.Range.create(node_1.Position.create(0, 0), node_1.Position.create(0, 0));
            return [node_1.Location.create(tsUri.toString(), locationRange)];
        }
        catch {
            // If .ts file doesn't exist, return empty
            return [];
        }
    }
    async findClassDefinition(nodeName) {
        const parts = nodeName.split('_');
        const workspaceRoot = this.projectScanner['workspaceRoot'];
        // Try to find .view.tree file
        const lastPart = parts[parts.length - 1];
        if (!lastPart) {
            return [];
        }
        const possiblePaths = [
            path.join(workspaceRoot, ...parts, lastPart + '.view.tree'),
            path.join(workspaceRoot, ...parts, lastPart, lastPart + '.view.tree')
        ];
        for (const viewTreePath of possiblePaths) {
            try {
                await fs.promises.access(viewTreePath);
                const uri = vscode_uri_1.URI.file(viewTreePath);
                const firstCharRange = node_1.Range.create(node_1.Position.create(0, 0), node_1.Position.create(0, 0));
                return [node_1.Location.create(uri.toString(), firstCharRange)];
            }
            catch {
                // Continue to next path
            }
        }
        // Try to find in project data
        const componentFile = this.projectScanner.getComponentFile(nodeName);
        if (componentFile) {
            const uri = vscode_uri_1.URI.file(componentFile);
            const firstCharRange = node_1.Range.create(node_1.Position.create(0, 0), node_1.Position.create(0, 0));
            return [node_1.Location.create(uri.toString(), firstCharRange)];
        }
        return [];
    }
    async findCompDefinition(documentUri, nodeName) {
        // Find corresponding .css.ts file
        const cssPath = documentUri.fsPath.replace(/\.view\.tree$/, '.css.ts');
        const cssUri = vscode_uri_1.URI.file(cssPath);
        try {
            await fs.promises.access(cssPath);
            // Try to find the CSS class definition
            const content = await fs.promises.readFile(cssPath, 'utf8');
            const classMatch = content.match(new RegExp(`${nodeName}\\s*:\\s*`, 'm'));
            if (classMatch) {
                const lines = content.split('\n');
                for (let i = 0; i < lines.length; i++) {
                    const currentLine = lines[i];
                    if (currentLine && currentLine.includes(classMatch[0])) {
                        const character = currentLine.indexOf(nodeName);
                        const range = node_1.Range.create(node_1.Position.create(i, character), node_1.Position.create(i, character + nodeName.length));
                        return [node_1.Location.create(cssUri.toString(), range)];
                    }
                }
            }
            // If no specific match, return beginning of file
            const locationRange = node_1.Range.create(node_1.Position.create(0, 0), node_1.Position.create(0, 0));
            return [node_1.Location.create(cssUri.toString(), locationRange)];
        }
        catch {
            return [];
        }
    }
    async findPropDefinition(documentUri, nodeName) {
        // Get the current component name
        const content = await this.getDocumentContent(documentUri);
        const currentComponent = this.getCurrentComponentFromContent(content);
        if (!currentComponent) {
            return [];
        }
        // Find corresponding .ts file
        const tsPath = documentUri.fsPath.replace(/\.view\.tree$/, '.ts');
        const tsUri = vscode_uri_1.URI.file(tsPath);
        try {
            await fs.promises.access(tsPath);
            // Find property in .ts file
            const propLocation = await this.findPropertyInFile(tsUri, currentComponent, nodeName);
            if (propLocation) {
                return [propLocation];
            }
            // Fallback to comp definition
            return await this.findCompDefinition(documentUri, nodeName);
        }
        catch {
            return [];
        }
    }
    async findSubPropDefinition(documentUri, _position, nodeName) {
        // This is a simplified version - in the original code this uses source maps
        // For now, we'll try to find it as a regular property
        return await this.findPropDefinition(documentUri, nodeName);
    }
    async findClassSymbolInFile(fileUri, className) {
        try {
            const content = await fs.promises.readFile(fileUri.fsPath, 'utf8');
            // Look for class definition
            const classRegex = new RegExp(`class\\s+${className.replace('$', '\\$')}\\b`, 'g');
            const match = classRegex.exec(content);
            if (match) {
                const lines = content.substring(0, match.index).split('\n');
                const line = lines.length - 1;
                const lastLine = lines[line];
                const character = lastLine ? lastLine.length : 0;
                const range = node_1.Range.create(node_1.Position.create(line, character), node_1.Position.create(line, character + className.length));
                return node_1.Location.create(fileUri.toString(), range);
            }
        }
        catch (error) {
            // File doesn't exist or can't be read
        }
        return null;
    }
    async findPropertyInFile(fileUri, className, propertyName) {
        try {
            const content = await fs.promises.readFile(fileUri.fsPath, 'utf8');
            // Look for property definition within class
            const classRegex = new RegExp(`class\\s+${className.replace('$', '\\$')}[^{]*{([^}]*(?:{[^}]*}[^}]*)*)}`, 'gs');
            const classMatch = classRegex.exec(content);
            if (classMatch) {
                const classContent = classMatch[1];
                if (classContent) {
                    const propRegex = new RegExp(`\\b${propertyName}\\s*[(:=]`, 'g');
                    const propMatch = propRegex.exec(classContent);
                    if (propMatch) {
                        const beforeMatch = content.substring(0, classMatch.index + classMatch[0].indexOf(classContent) + propMatch.index);
                        const lines = beforeMatch.split('\n');
                        const line = lines.length - 1;
                        const lastLine = lines[line];
                        const character = lastLine ? lastLine.length : 0;
                        const range = node_1.Range.create(node_1.Position.create(line, character), node_1.Position.create(line, character + propertyName.length));
                        return node_1.Location.create(fileUri.toString(), range);
                    }
                }
            }
        }
        catch (error) {
            // File doesn't exist or can't be read
        }
        return null;
    }
    async getDocumentContent(uri) {
        try {
            return await fs.promises.readFile(uri.fsPath, 'utf8');
        }
        catch {
            return '';
        }
    }
    getCurrentComponentFromContent(content) {
        const lines = content.split('\n');
        for (const line of lines) {
            const trimmed = line.trim();
            if (!line.startsWith('\t') && !line.startsWith(' ') && trimmed.startsWith('$')) {
                const firstWord = trimmed.split(/\s+/)[0];
                if (firstWord && firstWord.startsWith('$')) {
                    return firstWord;
                }
            }
        }
        return null;
    }
    getOffset(content, position) {
        const lines = content.split('\n');
        let offset = 0;
        for (let i = 0; i < position.line && i < lines.length; i++) {
            const currentLine = lines[i];
            if (currentLine) {
                offset += currentLine.length + 1; // +1 for newline
            }
        }
        return offset + position.character;
    }
}
exports.DefinitionProvider = DefinitionProvider;
//# sourceMappingURL=definition-provider.js.map