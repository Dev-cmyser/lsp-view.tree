import { RemoteConsole } from 'vscode-languageserver/node';
export interface ProjectData {
    components: Set<string>;
    componentProperties: Map<string, Set<string>>;
    componentFiles: Map<string, string>;
    fileComponents: Map<string, Set<string>>;
}
export declare class ProjectScanner {
    private workspaceRoot;
    private console;
    private projectData;
    private watchers;
    constructor(workspaceRoot: string, console: RemoteConsole);
    scanProject(): Promise<ProjectData>;
    private scanViewTreeFiles;
    private scanTsFiles;
    private findFiles;
    private parseViewTreeFile;
    private parseTsFile;
    private setupFileWatchers;
    private handleFileChange;
    updateSingleFile(filePath: string, content: string): void;
    private removeSingleFile;
    getProjectData(): ProjectData;
    getComponentsStartingWith(prefix: string): string[];
    getPropertiesForComponent(component: string): string[];
    getAllProperties(): string[];
    getComponentFile(component: string): string | undefined;
    dispose(): void;
}
//# sourceMappingURL=project-scanner.d.ts.map