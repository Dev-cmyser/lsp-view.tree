import * as fs from 'fs';
import * as path from 'path';
import { RemoteConsole } from 'vscode-languageserver/node';

export interface ProjectData {
	components: Set<string>;
	componentProperties: Map<string, Set<string>>;
	componentFiles: Map<string, string>; // component -> file path
	fileComponents: Map<string, Set<string>>; // file path -> components
}

export class ProjectScanner {
	private projectData: ProjectData = {
		components: new Set(),
		componentProperties: new Map(),
		componentFiles: new Map(),
		fileComponents: new Map()
	};

	private watchers: Map<string, fs.FSWatcher> = new Map();

	constructor(
		private workspaceRoot: string,
		private console: RemoteConsole
	) {}

	async scanProject(): Promise<ProjectData> {
		this.console.log('[view.tree] Starting project scan...');
		
		// Reset project data
		this.projectData = {
			components: new Set(),
			componentProperties: new Map(),
			componentFiles: new Map(),
			fileComponents: new Map()
		};

		try {
			// Scan .view.tree files
			await this.scanViewTreeFiles();
			
			// Scan .ts files
			await this.scanTsFiles();

			// Setup file watchers
			this.setupFileWatchers();

			this.console.log(`[view.tree] Scan complete: ${this.projectData.components.size} components, ${this.projectData.componentProperties.size} components with properties`);
			this.console.log(`[view.tree] Components found: ${Array.from(this.projectData.components).join(', ')}`);

		} catch (error) {
			this.console.error(`[view.tree] Error during project scan: ${error}`);
		}

		return this.projectData;
	}

	private async scanViewTreeFiles(): Promise<void> {
		const viewTreeFiles = await this.findFiles('**/*.view.tree');
		this.console.log(`[view.tree] Found ${viewTreeFiles.length} .view.tree files`);

		for (const filePath of viewTreeFiles) {
			try {
				const content = await fs.promises.readFile(filePath, 'utf8');
				this.parseViewTreeFile(content, this.projectData, filePath);
			} catch (error) {
				this.console.log(`[view.tree] Error reading ${filePath}: ${error}`);
			}
		}
	}

	private async scanTsFiles(): Promise<void> {
		const tsFiles = await this.findFiles('**/*.ts');
		this.console.log(`[view.tree] Found ${tsFiles.length} .ts files`);

		// Limit to first 100 files for performance
		const limitedTsFiles = tsFiles.slice(0, 100);
		
		for (const file of limitedTsFiles) {
			try {
				const content = await fs.promises.readFile(file, 'utf8');
				this.parseTsFile(content, this.projectData, file);
			} catch (error) {
				this.console.log(`[view.tree] Error reading ${file}: ${error}`);
			}
		}
	}

	private async findFiles(pattern: string): Promise<string[]> {
		const files: string[] = [];
		
		const scanDirectory = async (dir: string): Promise<void> => {
			try {
				const entries = await fs.promises.readdir(dir, { withFileTypes: true });
				
				for (const entry of entries) {
					const fullPath = path.join(dir, entry.name);
					
					if (entry.isDirectory()) {
						// Skip node_modules and other common directories to ignore
						if (!entry.name.startsWith('.') && entry.name !== 'node_modules') {
							await scanDirectory(fullPath);
						}
					} else if (entry.isFile()) {
						if (pattern.includes('*.view.tree') && fullPath.endsWith('.view.tree')) {
							files.push(fullPath);
						} else if (pattern.includes('*.ts') && fullPath.endsWith('.ts') && !fullPath.endsWith('.d.ts')) {
							files.push(fullPath);
						}
					}
				}
			} catch (error) {
				// Ignore permission errors and continue
			}
		};

		await scanDirectory(this.workspaceRoot);
		return files;
	}

	private parseViewTreeFile(content: string, data: ProjectData, filePath: string): void {
		const lines = content.split('\n');
		let currentComponent: string | null = null;

		// Clear previous components for this file
		if (data.fileComponents.has(filePath)) {
			const previousComponents = data.fileComponents.get(filePath)!;
			for (const comp of previousComponents) {
				data.componentFiles.delete(comp);
			}
		}
		data.fileComponents.set(filePath, new Set());

		for (const line of lines) {
			const trimmed = line.trim();

			// Take only the first word from lines without indentation
			if (!line.startsWith('\t') && !line.startsWith(' ') && trimmed.startsWith('$')) {
				const firstWord = trimmed.split(/\s+/)[0];
				if (firstWord && firstWord.startsWith('$')) {
					currentComponent = firstWord;
					data.components.add(firstWord);
					data.componentFiles.set(firstWord, filePath);
					data.fileComponents.get(filePath)!.add(firstWord);
				
					if (!data.componentProperties.has(firstWord)) {
						data.componentProperties.set(firstWord, new Set());
					}
				}
			}

			// Look for properties (indented lines without <= and <=>)
			if (currentComponent) {
				const indentMatch = line.match(/^(\s+)([a-zA-Z_][a-zA-Z0-9_?*]*)\s*/);
				if (indentMatch && indentMatch[1] && indentMatch[1].length > 0 && !trimmed.includes('<=') && !trimmed.includes('<=>')) {
					const property = indentMatch[2];
					if (property && !property.startsWith('$') && property !== 'null' && property !== 'true' && property !== 'false') {
						data.componentProperties.get(currentComponent)!.add(property);
					}
				}

				// Look for properties in bindings: <= PropertyName
				const bindingMatch = trimmed.match(/<=\s+([a-zA-Z_][a-zA-Z0-9_?*]*)/);
				if (bindingMatch) {
					const property = bindingMatch[1];
					if (property && !property.startsWith('$')) {
						data.componentProperties.get(currentComponent)!.add(property);
					}
				}
			}
		}
	}

	private parseTsFile(content: string, data: ProjectData, filePath: string): void {
		// Look for all $ components in TypeScript files
		const componentMatches = content.match(/\$\w+/g);
		if (componentMatches) {
			// Clear previous components for this file
			if (data.fileComponents.has(filePath)) {
				const previousComponents = data.fileComponents.get(filePath)!;
				for (const comp of previousComponents) {
					if (data.componentFiles.get(comp) === filePath) {
						data.componentFiles.delete(comp);
					}
				}
			}
			data.fileComponents.set(filePath, new Set());

			for (const match of componentMatches) {
				data.components.add(match);
				// Only set file mapping if not already set by .view.tree file
				if (!data.componentFiles.has(match)) {
					data.componentFiles.set(match, filePath);
				}
				data.fileComponents.get(filePath)!.add(match);
			}
		}
	}

	private setupFileWatchers(): void {
		// Clean up existing watchers
		for (const watcher of this.watchers.values()) {
			watcher.close();
		}
		this.watchers.clear();

		try {
			// Watch for .view.tree files
			const viewTreeWatcher = fs.watch(this.workspaceRoot, { recursive: true }, (eventType, filename) => {
				if (filename && (filename.endsWith('.view.tree') || filename.endsWith('.ts'))) {
					const fullPath = path.join(this.workspaceRoot, filename);
					this.handleFileChange(eventType, fullPath);
				}
			});
			
			this.watchers.set('main', viewTreeWatcher);
		} catch (error) {
			this.console.error(`[view.tree] Error setting up file watchers: ${error}`);
		}
	}

	private async handleFileChange(eventType: string, filePath: string): Promise<void> {
		this.console.log(`[view.tree] File ${eventType}: ${filePath}`);
		
		try {
			if (eventType === 'change') {
				// File modified
				if (fs.existsSync(filePath)) {
					const content = await fs.promises.readFile(filePath, 'utf8');
					this.updateSingleFile(filePath, content);
				}
			} else {
				// File deleted or renamed
				this.removeSingleFile(filePath);
			}
		} catch (error) {
			this.console.error(`[view.tree] Error handling file change: ${error}`);
		}
	}

	public updateSingleFile(filePath: string, content: string): void {
		this.console.log(`[view.tree] Updating single file: ${filePath}`);
		
		try {
			if (filePath.endsWith('.view.tree')) {
				this.parseViewTreeFile(content, this.projectData, filePath);
			} else if (filePath.endsWith('.ts')) {
				this.parseTsFile(content, this.projectData, filePath);
			}
		} catch (error) {
			this.console.error(`[view.tree] Error updating file ${filePath}: ${error}`);
		}
	}

	private removeSingleFile(filePath: string): void {
		this.console.log(`[view.tree] File deleted: ${filePath}`);
		
		// Remove components that were defined in this file
		if (this.projectData.fileComponents.has(filePath)) {
			const components = this.projectData.fileComponents.get(filePath)!;
			for (const component of components) {
				if (this.projectData.componentFiles.get(component) === filePath) {
					this.projectData.components.delete(component);
					this.projectData.componentFiles.delete(component);
					this.projectData.componentProperties.delete(component);
				}
			}
			this.projectData.fileComponents.delete(filePath);
		}
	}

	public getProjectData(): ProjectData {
		return this.projectData;
	}

	public getComponentsStartingWith(prefix: string): string[] {
		return Array.from(this.projectData.components)
			.filter(component => component.startsWith(prefix))
			.sort();
	}

	public getPropertiesForComponent(component: string): string[] {
		const properties = this.projectData.componentProperties.get(component);
		return properties ? Array.from(properties).sort() : [];
	}

	public getAllProperties(): string[] {
		const allProperties = new Set<string>();
		for (const properties of this.projectData.componentProperties.values()) {
			for (const property of properties) {
				allProperties.add(property);
			}
		}
		return Array.from(allProperties).sort();
	}

	public getComponentFile(component: string): string | undefined {
		return this.projectData.componentFiles.get(component);
	}

	public dispose(): void {
		// Clean up file watchers
		for (const watcher of this.watchers.values()) {
			watcher.close();
		}
		this.watchers.clear();
	}
}