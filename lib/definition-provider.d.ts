import { Location, Position, TextDocument } from 'vscode-languageserver/node';
import { ProjectScanner } from './project-scanner';
export declare class DefinitionProvider {
    private projectScanner;
    private parser;
    constructor(projectScanner: ProjectScanner);
    provideDefinition(document: TextDocument, position: Position): Promise<Location[]>;
    private getNodeType;
    private findRootClassDefinition;
    private findClassDefinition;
    private findCompDefinition;
    private findPropDefinition;
    private findSubPropDefinition;
    private findClassSymbolInFile;
    private findPropertyInFile;
    private getDocumentContent;
    private getCurrentComponentFromContent;
    private getOffset;
}
//# sourceMappingURL=definition-provider.d.ts.map