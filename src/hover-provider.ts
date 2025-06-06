import {
	Hover,
	MarkupContent,
	MarkupKind,
	Position,
	Range,
	TextDocument
} from 'vscode-languageserver/node';
import { URI } from 'vscode-uri';
import * as fs from 'fs';


import { ProjectScanner } from './project-scanner';
import { ViewTreeParser } from './view-tree-parser';

export class HoverProvider {
	private parser: ViewTreeParser;

	constructor(private projectScanner: ProjectScanner) {
		this.parser = new ViewTreeParser();
	}

	async provideHover(document: TextDocument, position: Position): Promise<Hover | null> {
		const content = document.getText();
		const wordRange = this.parser.getWordRangeAtPosition(content, position);
		
		if (!wordRange) {
			return null;
		}

		const nodeName = content.substring(
			this.getOffset(content, wordRange.start),
			this.getOffset(content, wordRange.end)
		);

		if (!nodeName) {
			return null;
		}

		const nodeType = this.getNodeType(content, position, wordRange);
		const documentUri = URI.parse(document.uri);

		let hoverContent: MarkupContent | null = null;

		switch (nodeType) {
			case 'root_class':
				hoverContent = await this.getComponentHover(nodeName, documentUri);
				break;
			case 'class':
				hoverContent = await this.getComponentHover(nodeName);
				break;
			case 'comp':
				hoverContent = await this.getCssClassHover(nodeName, documentUri);
				break;
			case 'prop':
				hoverContent = await this.getPropertyHover(nodeName, content);
				break;
			case 'sub_prop':
				hoverContent = await this.getSubPropertyHover(nodeName, content);
				break;
			default:
				hoverContent = this.getGenericHover(nodeName);
		}

		if (!hoverContent) {
			return null;
		}

		return {
			contents: hoverContent,
			range: wordRange
		};
	}

	private getNodeType(content: string, position: Position, wordRange: Range): string {
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
		const beforeWord = line ? line.substring(0, wordRange.start.character) : '';
		if (/[>=^]\s*$/.test(beforeWord)) {
			return 'prop';
		}

		// Default to sub_prop for deeper nested items
		return 'sub_prop';
	}

	private async getComponentHover(componentName: string, documentUri?: URI): Promise<MarkupContent | null> {
		const projectData = this.projectScanner.getProjectData();
		
		if (!projectData.components.has(componentName)) {
			return null;
		}

		const markdownContent: string[] = [];
		
		// Component header
		markdownContent.push(`**Component**: \`${componentName}\``);
		markdownContent.push('');

		// Component file location
		const componentFile = this.projectScanner.getComponentFile(componentName);
		if (componentFile) {
			const relativePath = this.getRelativePath(componentFile);
			markdownContent.push(`**File**: \`${relativePath}\``);
			markdownContent.push('');
		}

		// Component properties
		const properties = this.projectScanner.getPropertiesForComponent(componentName);
		if (properties.length > 0) {
			markdownContent.push('**Properties**:');
			for (const prop of properties.slice(0, 10)) { // Limit to first 10
				markdownContent.push(`- \`${prop}\``);
			}
			if (properties.length > 10) {
				markdownContent.push(`- ... and ${properties.length - 10} more`);
			}
			markdownContent.push('');
		}

		// Component documentation from TypeScript file
		if (documentUri) {
			const tsDoc = await this.getTypeScriptDocumentation(componentName, documentUri);
			if (tsDoc) {
				markdownContent.push('**Documentation**:');
				markdownContent.push(tsDoc);
				markdownContent.push('');
			}
		}

		// Usage information
		markdownContent.push('**Usage**:');
		markdownContent.push('```tree');
		markdownContent.push(`${componentName}`);
		if (properties.length > 0) {
			markdownContent.push(`\tproperty <= value`);
		}
		markdownContent.push('```');

		return {
			kind: MarkupKind.Markdown,
			value: markdownContent.join('\n')
		};
	}

	private async getCssClassHover(className: string, documentUri: URI): Promise<MarkupContent | null> {
		const markdownContent: string[] = [];
		
		markdownContent.push(`**CSS Class**: \`${className}\``);
		markdownContent.push('');

		// Try to find CSS definition
		const cssPath = documentUri.fsPath.replace(/\.view\.tree$/, '.css.ts');
		
		try {
			await fs.promises.access(cssPath);
			const relativePath = this.getRelativePath(cssPath);
			markdownContent.push(`**Defined in**: \`${relativePath}\``);
			markdownContent.push('');

			// Try to extract CSS rules
			const cssContent = await fs.promises.readFile(cssPath, 'utf8');
			const cssRule = this.extractCssRule(cssContent, className);
			if (cssRule) {
				markdownContent.push('**CSS Rules**:');
				markdownContent.push('```css');
				markdownContent.push(cssRule);
				markdownContent.push('```');
			}
		} catch {
			markdownContent.push('*CSS file not found*');
		}

		return {
			kind: MarkupKind.Markdown,
			value: markdownContent.join('\n')
		};
	}

	private async getPropertyHover(propertyName: string, content: string): Promise<MarkupContent | null> {
		const currentComponent = this.parser.getCurrentComponent(content, { line: 0, character: 0 });
		const markdownContent: string[] = [];
		
		markdownContent.push(`**Property**: \`${propertyName}\``);
		markdownContent.push('');

		if (currentComponent) {
			markdownContent.push(`**Component**: \`${currentComponent}\``);
			markdownContent.push('');
		}

		// Property type information
		const propertyInfo = this.getPropertyTypeInfo(propertyName);
		if (propertyInfo) {
			markdownContent.push(`**Type**: ${propertyInfo.type}`);
			markdownContent.push('');
			markdownContent.push(`**Description**: ${propertyInfo.description}`);
			markdownContent.push('');
		}

		// Usage examples
		const usageExamples = this.getPropertyUsageExamples(propertyName);
		if (usageExamples.length > 0) {
			markdownContent.push('**Usage**:');
			markdownContent.push('```tree');
			for (const example of usageExamples) {
				markdownContent.push(example);
			}
			markdownContent.push('```');
		}

		return {
			kind: MarkupKind.Markdown,
			value: markdownContent.join('\n')
		};
	}

	private async getSubPropertyHover(propertyName: string, content: string): Promise<MarkupContent | null> {
		// For sub-properties, provide similar information as regular properties
		return this.getPropertyHover(propertyName, content);
	}

	private getGenericHover(nodeName: string): MarkupContent | null {
		const markdownContent: string[] = [];
		
		markdownContent.push(`**Element**: \`${nodeName}\``);
		markdownContent.push('');

		// Check if it's a special value
		const specialValueInfo = this.getSpecialValueInfo(nodeName);
		if (specialValueInfo) {
			markdownContent.push(`**Type**: ${specialValueInfo.type}`);
			markdownContent.push('');
			markdownContent.push(`**Description**: ${specialValueInfo.description}`);
			markdownContent.push('');
		}

		if (markdownContent.length <= 2) {
			return null; // No useful information to show
		}

		return {
			kind: MarkupKind.Markdown,
			value: markdownContent.join('\n')
		};
	}

	private getPropertyTypeInfo(propertyName: string): { type: string; description: string } | null {
		const propertyTypes: Record<string, { type: string; description: string }> = {
			'dom_name': {
				type: 'string',
				description: 'HTML tag name for the DOM element'
			},
			'dom_name_space': {
				type: 'string',
				description: 'XML namespace for the DOM element'
			},
			'attr': {
				type: 'Dictionary<string>',
				description: 'HTML attributes for the DOM element'
			},
			'field': {
				type: 'any',
				description: 'Form field value binding'
			},
			'value': {
				type: 'any',
				description: 'Element value or content'
			},
			'enabled': {
				type: 'boolean',
				description: 'Whether the element is enabled'
			},
			'visible': {
				type: 'boolean',
				description: 'Whether the element is visible'
			},
			'title': {
				type: 'string',
				description: 'Element title or tooltip text'
			},
			'hint': {
				type: 'string',
				description: 'Hint text for the element'
			},
			'sub': {
				type: 'Array<$mol_view>',
				description: 'Child elements or components'
			},
			'event': {
				type: 'Dictionary<Function>',
				description: 'Event handlers'
			},
			'plugins': {
				type: 'Array<$mol_plugin>',
				description: 'Plugins to apply to the element'
			}
		};

		return propertyTypes[propertyName] || null;
	}

	private getPropertyUsageExamples(propertyName: string): string[] {
		const examples: Record<string, string[]> = {
			'dom_name': [
				'\tdom_name \\div',
				'\tdom_name \\span'
			],
			'attr': [
				'\tattr *',
				'\t\tclass \\my-class',
				'\t\tid \\my-id'
			],
			'field': [
				'\tfield <= value',
				'\tfield <=> current_value'
			],
			'value': [
				'\tvalue \\Hello World',
				'\tvalue <= text'
			],
			'enabled': [
				'\tenabled <= is_active',
				'\tenabled true'
			],
			'visible': [
				'\tvisible <= show_element',
				'\tvisible false'
			],
			'sub': [
				'\tsub /',
				'\t\t<= items',
				'\t\t$my_component'
			],
			'event': [
				'\tevent *',
				'\t\tclick <= handle_click'
			]
		};

		return examples[propertyName] || [];
	}

	private getSpecialValueInfo(value: string): { type: string; description: string } | null {
		const specialValues: Record<string, { type: string; description: string }> = {
			'null': {
				type: 'null',
				description: 'Represents an empty or undefined value'
			},
			'true': {
				type: 'boolean',
				description: 'Boolean true value'
			},
			'false': {
				type: 'boolean',
				description: 'Boolean false value'
			},
			'/': {
				type: 'list',
				description: 'Empty list marker'
			},
			'*': {
				type: 'dictionary',
				description: 'Dictionary marker for key-value pairs'
			},
			'\\': {
				type: 'string',
				description: 'String literal marker'
			},
			'@\\': {
				type: 'localized string',
				description: 'Localized string literal marker'
			}
		};

		return specialValues[value] || null;
	}

	private async getTypeScriptDocumentation(componentName: string, documentUri: URI): Promise<string | null> {
		try {
			const tsPath = documentUri.fsPath.replace(/\.view\.tree$/, '.ts');
			const content = await fs.promises.readFile(tsPath, 'utf8');
			
			// Look for JSDoc comments before class definition
			const classRegex = new RegExp(`/\\*\\*([^*]|\\*(?!/))*\\*/\\s*export\\s+class\\s+${componentName.replace('$', '\\$')}`, 'gs');
			const match = classRegex.exec(content);
			
			if (match) {
				const docComment = match[0].match(/\/\*\*([\s\S]*?)\*\//);
				if (docComment && docComment[1]) {
					return docComment[1]
						.split('\n')
						.map(line => line.replace(/^\s*\*\s?/, '').trim())
						.filter(line => line.length > 0)
						.join('\n');
				}
			}
		} catch {
			// File doesn't exist or can't be read
		}
		
		return null;
	}

	private extractCssRule(cssContent: string, className: string): string | null {
		try {
			// Look for CSS class definition in TypeScript CSS-in-JS format
			const classRegex = new RegExp(`${className}\\s*:\\s*{([^}]+)}`, 'gs');
			const match = classRegex.exec(cssContent);
			
			if (match && match[1]) {
				return match[1].trim()
					.split('\n')
					.map(line => line.trim())
					.filter(line => line.length > 0)
					.join('\n');
			}
		} catch {
			// Error parsing CSS
		}
		
		return null;
	}

	private getRelativePath(filePath: string): string {
		const workspaceRoot = this.projectScanner['workspaceRoot'];
		if (filePath.startsWith(workspaceRoot)) {
			return filePath.substring(workspaceRoot.length + 1);
		}
		return filePath;
	}

	private getOffset(content: string, position: Position): number {
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