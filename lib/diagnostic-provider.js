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
exports.DiagnosticProvider = void 0;
const node_1 = require("vscode-languageserver/node");
const vscode_uri_1 = require("vscode-uri");
const fs = __importStar(require("fs"));
const path = __importStar(require("path"));
const view_tree_parser_1 = require("./view-tree-parser");
class DiagnosticProvider {
    constructor(projectScanner) {
        this.projectScanner = projectScanner;
        this.parser = new view_tree_parser_1.ViewTreeParser();
    }
    async provideDiagnostics(document) {
        const content = document.getText();
        const diagnostics = [];
        // Only process .view.tree files
        if (!document.uri.endsWith('.view.tree')) {
            return diagnostics;
        }
        try {
            // Parse the document
            const parseResult = this.parser.parse(content);
            // Add parse errors
            for (const error of parseResult.errors) {
                diagnostics.push({
                    severity: this.mapSeverity(error.severity),
                    range: error.range,
                    message: error.message,
                    source: 'view.tree'
                });
            }
            // Validate syntax
            const syntaxDiagnostics = await this.validateSyntax(content, document.uri);
            diagnostics.push(...syntaxDiagnostics);
            // Validate components
            const componentDiagnostics = await this.validateComponents(parseResult.components, document.uri);
            diagnostics.push(...componentDiagnostics);
            // Validate properties
            const propertyDiagnostics = await this.validateProperties(parseResult.components, content);
            diagnostics.push(...propertyDiagnostics);
            // Validate indentation
            const indentationDiagnostics = this.validateIndentation(content);
            diagnostics.push(...indentationDiagnostics);
            // Validate bindings
            const bindingDiagnostics = this.validateBindings(content);
            diagnostics.push(...bindingDiagnostics);
        }
        catch (error) {
            // If parsing completely fails, add a general error
            diagnostics.push({
                severity: node_1.DiagnosticSeverity.Error,
                range: node_1.Range.create(node_1.Position.create(0, 0), node_1.Position.create(0, 0)),
                message: `Failed to parse view.tree file: ${error}`,
                source: 'view.tree'
            });
        }
        return diagnostics;
    }
    async validateSyntax(content, _documentUri) {
        const diagnostics = [];
        const lines = content.split('\n');
        for (let lineIndex = 0; lineIndex < lines.length; lineIndex++) {
            const line = lines[lineIndex];
            if (!line)
                continue;
            const trimmed = line.trim();
            // Skip empty lines and comments
            if (!trimmed || trimmed.startsWith('//')) {
                continue;
            }
            // Check for invalid characters in component names
            if (trimmed.startsWith('$')) {
                const componentName = trimmed.split(/\s+/)[0];
                if (componentName && !/^\$[a-zA-Z_][a-zA-Z0-9_]*$/.test(componentName)) {
                    const range = node_1.Range.create(node_1.Position.create(lineIndex, line.indexOf(componentName)), node_1.Position.create(lineIndex, line.indexOf(componentName) + componentName.length));
                    diagnostics.push({
                        severity: node_1.DiagnosticSeverity.Error,
                        range,
                        message: `Invalid component name: ${componentName}. Component names must start with $ followed by letters, numbers, or underscores.`,
                        source: 'view.tree'
                    });
                }
            }
            // Check for mixing tabs and spaces
            if (line && line.length > 0 && !trimmed.startsWith('//')) {
                const leadingWhitespace = line.match(/^(\s*)/)?.[1] || '';
                const hasTab = leadingWhitespace.includes('\t');
                const hasSpace = leadingWhitespace.includes(' ');
                if (hasTab && hasSpace) {
                    const range = node_1.Range.create(node_1.Position.create(lineIndex, 0), node_1.Position.create(lineIndex, leadingWhitespace.length));
                    diagnostics.push({
                        severity: node_1.DiagnosticSeverity.Warning,
                        range,
                        message: 'Mixed tabs and spaces in indentation. Use either tabs or spaces consistently.',
                        source: 'view.tree'
                    });
                }
            }
            // Check for invalid binding syntax
            if (trimmed.includes('<=') || trimmed.includes('<=>')) {
                const bindingMatch = trimmed.match(/(<=?>?)\s*([a-zA-Z_$][a-zA-Z0-9_]*)?/);
                if (bindingMatch && !bindingMatch[2] && bindingMatch[1]) {
                    const operatorIndex = line.indexOf(bindingMatch[1]);
                    const range = node_1.Range.create(node_1.Position.create(lineIndex, operatorIndex), node_1.Position.create(lineIndex, operatorIndex + bindingMatch[1].length));
                    diagnostics.push({
                        severity: node_1.DiagnosticSeverity.Error,
                        range,
                        message: 'Binding operator must be followed by a property name.',
                        source: 'view.tree'
                    });
                }
            }
        }
        return diagnostics;
    }
    async validateComponents(components, documentUri) {
        const diagnostics = [];
        const projectData = this.projectScanner.getProjectData();
        for (const component of components) {
            const componentName = component.name;
            // Check if component exists in project
            if (!projectData.components.has(componentName) && !componentName.startsWith('$mol_')) {
                // Skip built-in $mol_ components for now
                diagnostics.push({
                    severity: node_1.DiagnosticSeverity.Warning,
                    range: component.range,
                    message: `Component '${componentName}' not found in project. Consider defining it or check the spelling.`,
                    source: 'view.tree'
                });
            }
            // Check for duplicate component definitions in same file
            const duplicates = components.filter(c => c.name === componentName);
            if (duplicates.length > 1) {
                for (let i = 1; i < duplicates.length; i++) {
                    diagnostics.push({
                        severity: node_1.DiagnosticSeverity.Error,
                        range: duplicates[i].range,
                        message: `Duplicate component definition: ${componentName}`,
                        source: 'view.tree'
                    });
                }
            }
            // Validate corresponding TypeScript file
            const tsPath = documentUri.replace(/\.view\.tree$/, '.ts');
            const tsUri = vscode_uri_1.URI.parse(tsPath);
            try {
                await fs.promises.access(tsUri.fsPath);
            }
            catch {
                if (component.startLine === 0) { // Only for root component
                    diagnostics.push({
                        severity: node_1.DiagnosticSeverity.Information,
                        range: component.range,
                        message: `TypeScript file not found: ${path.basename(tsUri.fsPath)}. Consider creating it for component implementation.`,
                        source: 'view.tree'
                    });
                }
            }
        }
        return diagnostics;
    }
    async validateProperties(components, content) {
        const diagnostics = [];
        for (const component of components) {
            for (const property of component.properties) {
                const propertyName = property.name;
                // Check for invalid property names
                if (!/^[a-zA-Z_$][a-zA-Z0-9_?*]*$/.test(propertyName)) {
                    diagnostics.push({
                        severity: node_1.DiagnosticSeverity.Error,
                        range: property.range,
                        message: `Invalid property name: ${propertyName}. Property names must start with a letter, $, or underscore.`,
                        source: 'view.tree'
                    });
                }
                // Check for reserved property names
                const reservedNames = ['constructor', 'prototype', '__proto__'];
                if (reservedNames.includes(propertyName)) {
                    diagnostics.push({
                        severity: node_1.DiagnosticSeverity.Error,
                        range: property.range,
                        message: `Reserved property name: ${propertyName}. Choose a different name.`,
                        source: 'view.tree'
                    });
                }
                // Check for duplicate properties within component
                const duplicateProps = component.properties.filter((p) => p.name === propertyName);
                if (duplicateProps.length > 1) {
                    for (let i = 1; i < duplicateProps.length; i++) {
                        diagnostics.push({
                            severity: node_1.DiagnosticSeverity.Warning,
                            range: duplicateProps[i].range,
                            message: `Duplicate property: ${propertyName}`,
                            source: 'view.tree'
                        });
                    }
                }
                // Validate binding targets
                if (property.isBinding && property.value) {
                    const bindingTarget = property.value;
                    if (!/^[a-zA-Z_$][a-zA-Z0-9_?*]*$/.test(bindingTarget)) {
                        const lines = content.split('\n');
                        const line = lines[property.line];
                        if (line) {
                            const bindingIndex = line.indexOf(bindingTarget);
                            if (bindingIndex >= 0) {
                                const range = node_1.Range.create(node_1.Position.create(property.line, bindingIndex), node_1.Position.create(property.line, bindingIndex + bindingTarget.length));
                                diagnostics.push({
                                    severity: node_1.DiagnosticSeverity.Error,
                                    range,
                                    message: `Invalid binding target: ${bindingTarget}`,
                                    source: 'view.tree'
                                });
                            }
                        }
                    }
                }
            }
        }
        return diagnostics;
    }
    validateIndentation(content) {
        const diagnostics = [];
        const lines = content.split('\n');
        let lastNonEmptyIndent = 0;
        for (let lineIndex = 0; lineIndex < lines.length; lineIndex++) {
            const line = lines[lineIndex];
            if (!line)
                continue;
            const trimmed = line.trim();
            // Skip empty lines and comments
            if (!trimmed || trimmed.startsWith('//')) {
                continue;
            }
            const currentIndent = this.getIndentLevel(line);
            // Root level components should have no indentation
            if (trimmed.startsWith('$') && currentIndent > 0) {
                const range = node_1.Range.create(node_1.Position.create(lineIndex, 0), node_1.Position.create(lineIndex, currentIndent));
                diagnostics.push({
                    severity: node_1.DiagnosticSeverity.Error,
                    range,
                    message: 'Component definitions should not be indented.',
                    source: 'view.tree'
                });
            }
            // Properties should be indented
            if (!trimmed.startsWith('$') && currentIndent === 0) {
                const range = node_1.Range.create(node_1.Position.create(lineIndex, 0), node_1.Position.create(lineIndex, 1));
                diagnostics.push({
                    severity: node_1.DiagnosticSeverity.Error,
                    range,
                    message: 'Properties must be indented under their component.',
                    source: 'view.tree'
                });
            }
            // Check for excessive indentation jumps
            if (currentIndent > lastNonEmptyIndent + 1) {
                const range = node_1.Range.create(node_1.Position.create(lineIndex, 0), node_1.Position.create(lineIndex, currentIndent));
                diagnostics.push({
                    severity: node_1.DiagnosticSeverity.Warning,
                    range,
                    message: 'Indentation increased by more than one level. This might indicate a structural issue.',
                    source: 'view.tree'
                });
            }
            if (trimmed.length > 0) {
                lastNonEmptyIndent = currentIndent;
            }
        }
        return diagnostics;
    }
    validateBindings(content) {
        const diagnostics = [];
        const lines = content.split('\n');
        for (let lineIndex = 0; lineIndex < lines.length; lineIndex++) {
            const line = lines[lineIndex];
            if (!line)
                continue;
            const trimmed = line.trim();
            // Skip empty lines and comments
            if (!trimmed || trimmed.startsWith('//')) {
                continue;
            }
            // Check for malformed binding operators
            const malformedBindings = [
                { pattern: /[^<]=[^>]/, message: 'Use <= or <=> for bindings, not =' },
                { pattern: /<[^=]/, message: 'Incomplete binding operator. Use <= or <=>' },
                { pattern: />[^=]/, message: 'Invalid operator. Use <= or <=>' },
                { pattern: /<=\s*$/, message: 'Binding operator <= must be followed by a property name' },
                { pattern: /<=>\s*$/, message: 'Binding operator <=> must be followed by a property name' }
            ];
            for (const check of malformedBindings) {
                const match = trimmed.match(check.pattern);
                if (match && match[0]) {
                    const matchIndex = line.indexOf(match[0]);
                    const range = node_1.Range.create(node_1.Position.create(lineIndex, matchIndex), node_1.Position.create(lineIndex, matchIndex + match[0].length));
                    diagnostics.push({
                        severity: node_1.DiagnosticSeverity.Error,
                        range,
                        message: check.message,
                        source: 'view.tree'
                    });
                }
            }
            // Check for conflicting bindings
            if (trimmed.includes('<=') && trimmed.includes('<=>')) {
                const range = node_1.Range.create(node_1.Position.create(lineIndex, 0), node_1.Position.create(lineIndex, line.length));
                diagnostics.push({
                    severity: node_1.DiagnosticSeverity.Error,
                    range,
                    message: 'Cannot use both <= and <=> operators in the same line.',
                    source: 'view.tree'
                });
            }
        }
        return diagnostics;
    }
    getIndentLevel(line) {
        let indent = 0;
        for (const char of line) {
            if (char === '\t') {
                indent++;
            }
            else if (char === ' ') {
                indent++; // Could be adjusted for different space-to-tab ratios
            }
            else {
                break;
            }
        }
        return indent;
    }
    mapSeverity(severity) {
        switch (severity) {
            case 'error':
                return node_1.DiagnosticSeverity.Error;
            case 'warning':
                return node_1.DiagnosticSeverity.Warning;
            case 'info':
                return node_1.DiagnosticSeverity.Information;
            default:
                return node_1.DiagnosticSeverity.Information;
        }
    }
}
exports.DiagnosticProvider = DiagnosticProvider;
//# sourceMappingURL=diagnostic-provider.js.map