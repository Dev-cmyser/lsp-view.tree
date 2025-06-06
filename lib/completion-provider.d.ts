import { CompletionItem, Position, TextDocument } from 'vscode-languageserver/node';
import { ProjectScanner } from './project-scanner';
export declare class CompletionProvider {
    private projectScanner;
    private parser;
    constructor(projectScanner: ProjectScanner);
    provideCompletionItems(document: TextDocument, position: Position): Promise<CompletionItem[]>;
    private getCompletionContext;
    private getCurrentComponent;
    private addComponentCompletions;
    private addPropertyCompletions;
    private addCommonProperties;
    private addBindingCompletions;
    private addValueCompletions;
    private addCssClassCompletions;
    private addEventHandlerCompletions;
}
//# sourceMappingURL=completion-provider.d.ts.map