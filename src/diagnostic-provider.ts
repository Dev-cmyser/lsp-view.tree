import {
	Diagnostic,
	DiagnosticSeverity,
	Position,
	Range,
	TextDocument
} from 'vscode-languageserver/node';
import { URI } from 'vscode-uri';
import * as fs from 'fs';
import * as path from 'path';

import { ProjectScanner } from './project-scanner';
import { ViewTreeParser } from './view-tree-parser';

export class DiagnosticProvider {
	private parser: ViewTreeParser;

	constructor(private projectScanner: ProjectScanner) {
		this.parser = new ViewTreeParser();
	}

	async provideDiagnostics(document: TextDocument): Promise<Diagnostic[]> {
		const content = document.getText();
		const diagnostics: Diagnostic[] = [];

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

		} catch (error) {
			// If parsing completely fails, add a general error
			diagnostics.push({
				severity: DiagnosticSeverity.Error,
				range: Range.create(Position.create(0, 0), Position.create(0, 0)),
				message: `Failed to parse view.tree file: ${error}`,
				source: 'view.tree'
			});
		}

		return diagnostics;
	}

	private async validateSyntax(content: string, _documentUri: string): Promise<Diagnostic[]> {
		const diagnostics: Diagnostic[] = [];
		const lines = content.split('\n');

		for (let lineIndex = 0; lineIndex < lines.length; lineIndex++) {
			const line = lines[lineIndex];
			if (!line) continue;
			const trimmed = line.trim();

			// Skip empty lines and comments
			if (!trimmed || trimmed.startsWith('//')) {
				continue;
			}

			// Check for invalid characters in component names
			if (trimmed.startsWith('$')) {
				const componentName = trimmed.split(/\s+/)[0];
				if (componentName && !/^\$[a-zA-Z_][a-zA-Z0-9_]*$/.test(componentName)) {
					const range = Range.create(
						Position.create(lineIndex, line.indexOf(componentName)),
						Position.create(lineIndex, line.indexOf(componentName) + componentName.length)
					);
					diagnostics.push({
						severity: DiagnosticSeverity.Error,
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
					const range = Range.create(
						Position.create(lineIndex, 0),
						Position.create(lineIndex, leadingWhitespace.length)
					);
					diagnostics.push({
						severity: DiagnosticSeverity.Warning,
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
					const range = Range.create(
						Position.create(lineIndex, operatorIndex),
						Position.create(lineIndex, operatorIndex + bindingMatch[1].length)
					);
					diagnostics.push({
						severity: DiagnosticSeverity.Error,
						range,
						message: 'Binding operator must be followed by a property name.',
						source: 'view.tree'
					});
				}
			}
		}

		return diagnostics;
	}

	private async validateComponents(components: any[], documentUri: string): Promise<Diagnostic[]> {
		const diagnostics: Diagnostic[] = [];
		const projectData = this.projectScanner.getProjectData();

		for (const component of components) {
			const componentName = component.name;

			// Check if component exists in project
			if (!projectData.components.has(componentName) && !componentName.startsWith('$mol_')) {
				// Skip built-in $mol_ components for now
				diagnostics.push({
					severity: DiagnosticSeverity.Warning,
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
						severity: DiagnosticSeverity.Error,
						range: duplicates[i].range,
						message: `Duplicate component definition: ${componentName}`,
						source: 'view.tree'
					});
				}
			}

			// Validate corresponding TypeScript file
			const tsPath = documentUri.replace(/\.view\.tree$/, '.ts');
			const tsUri = URI.parse(tsPath);
			
			try {
				await fs.promises.access(tsUri.fsPath);
			} catch {
				if (component.startLine === 0) { // Only for root component
					diagnostics.push({
						severity: DiagnosticSeverity.Information,
						range: component.range,
						message: `TypeScript file not found: ${path.basename(tsUri.fsPath)}. Consider creating it for component implementation.`,
						source: 'view.tree'
					});
				}
			}
		}

		return diagnostics;
	}

	private async validateProperties(components: any[], content: string): Promise<Diagnostic[]> {
		const diagnostics: Diagnostic[] = [];

		for (const component of components) {
			for (const property of component.properties) {
				const propertyName = property.name;

				// Check for invalid property names
				if (!/^[a-zA-Z_$][a-zA-Z0-9_?*]*$/.test(propertyName)) {
					diagnostics.push({
						severity: DiagnosticSeverity.Error,
						range: property.range,
						message: `Invalid property name: ${propertyName}. Property names must start with a letter, $, or underscore.`,
						source: 'view.tree'
					});
				}

				// Check for reserved property names
				const reservedNames = ['constructor', 'prototype', '__proto__'];
				if (reservedNames.includes(propertyName)) {
					diagnostics.push({
						severity: DiagnosticSeverity.Error,
						range: property.range,
						message: `Reserved property name: ${propertyName}. Choose a different name.`,
						source: 'view.tree'
					});
				}

				// Check for duplicate properties within component
				const duplicateProps = component.properties.filter((p: any) => p.name === propertyName);
				if (duplicateProps.length > 1) {
					for (let i = 1; i < duplicateProps.length; i++) {
						diagnostics.push({
							severity: DiagnosticSeverity.Warning,
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
								const range = Range.create(
									Position.create(property.line, bindingIndex),
									Position.create(property.line, bindingIndex + bindingTarget.length)
								);
								diagnostics.push({
									severity: DiagnosticSeverity.Error,
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

	private validateIndentation(content: string): Diagnostic[] {
		const diagnostics: Diagnostic[] = [];
		const lines = content.split('\n');
		let lastNonEmptyIndent = 0;

		for (let lineIndex = 0; lineIndex < lines.length; lineIndex++) {
			const line = lines[lineIndex];
			if (!line) continue;
			const trimmed = line.trim();

			// Skip empty lines and comments
			if (!trimmed || trimmed.startsWith('//')) {
				continue;
			}

			const currentIndent = this.getIndentLevel(line);

			// Root level components should have no indentation
			if (trimmed.startsWith('$') && currentIndent > 0) {
				const range = Range.create(
					Position.create(lineIndex, 0),
					Position.create(lineIndex, currentIndent)
				);
				diagnostics.push({
					severity: DiagnosticSeverity.Error,
					range,
					message: 'Component definitions should not be indented.',
					source: 'view.tree'
				});
			}

			// Properties should be indented
			if (!trimmed.startsWith('$') && currentIndent === 0) {
				const range = Range.create(
					Position.create(lineIndex, 0),
					Position.create(lineIndex, 1)
				);
				diagnostics.push({
					severity: DiagnosticSeverity.Error,
					range,
					message: 'Properties must be indented under their component.',
					source: 'view.tree'
				});
			}

			// Check for excessive indentation jumps
			if (currentIndent > lastNonEmptyIndent + 1) {
				const range = Range.create(
					Position.create(lineIndex, 0),
					Position.create(lineIndex, currentIndent)
				);
				diagnostics.push({
					severity: DiagnosticSeverity.Warning,
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

	private validateBindings(content: string): Diagnostic[] {
		const diagnostics: Diagnostic[] = [];
		const lines = content.split('\n');

		for (let lineIndex = 0; lineIndex < lines.length; lineIndex++) {
			const line = lines[lineIndex];
			if (!line) continue;
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
					const range = Range.create(
						Position.create(lineIndex, matchIndex),
						Position.create(lineIndex, matchIndex + match[0].length)
					);
					diagnostics.push({
						severity: DiagnosticSeverity.Error,
						range,
						message: check.message,
						source: 'view.tree'
					});
				}
			}

			// Check for conflicting bindings
			if (trimmed.includes('<=') && trimmed.includes('<=>')) {
				const range = Range.create(
					Position.create(lineIndex, 0),
					Position.create(lineIndex, line.length)
				);
				diagnostics.push({
					severity: DiagnosticSeverity.Error,
					range,
					message: 'Cannot use both <= and <=> operators in the same line.',
					source: 'view.tree'
				});
			}
		}

		return diagnostics;
	}

	private getIndentLevel(line: string): number {
		let indent = 0;
		for (const char of line) {
			if (char === '\t') {
				indent++;
			} else if (char === ' ') {
				indent++; // Could be adjusted for different space-to-tab ratios
			} else {
				break;
			}
		}
		return indent;
	}

	private mapSeverity(severity: 'error' | 'warning' | 'info'): DiagnosticSeverity {
		switch (severity) {
			case 'error':
				return DiagnosticSeverity.Error;
			case 'warning':
				return DiagnosticSeverity.Warning;
			case 'info':
				return DiagnosticSeverity.Information;
			default:
				return DiagnosticSeverity.Information;
		}
	}
}