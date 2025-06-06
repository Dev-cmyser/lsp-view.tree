import { Hover, Position, TextDocument } from 'vscode-languageserver/node';
import { ProjectScanner } from './project-scanner';
export declare class HoverProvider {
    private projectScanner;
    private parser;
    constructor(projectScanner: ProjectScanner);
    provideHover(document: TextDocument, position: Position): Promise<Hover | null>;
    private getNodeType;
    private getComponentHover;
    private getCssClassHover;
    private getPropertyHover;
    private getSubPropertyHover;
    private getGenericHover;
    private getPropertyTypeInfo;
    private getPropertyUsageExamples;
    private getSpecialValueInfo;
    private getTypeScriptDocumentation;
    private extractCssRule;
    private getRelativePath;
    private getOffset;
}
//# sourceMappingURL=hover-provider.d.ts.map