import { Diagnostic, TextDocument } from 'vscode-languageserver/node';
import { ProjectScanner } from './project-scanner';
export declare class DiagnosticProvider {
    private projectScanner;
    private parser;
    constructor(projectScanner: ProjectScanner);
    provideDiagnostics(document: TextDocument): Promise<Diagnostic[]>;
    private validateSyntax;
    private validateComponents;
    private validateProperties;
    private validateIndentation;
    private validateBindings;
    private getIndentLevel;
    private mapSeverity;
}
//# sourceMappingURL=diagnostic-provider.d.ts.map