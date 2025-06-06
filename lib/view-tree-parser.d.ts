import { Position, Range } from "vscode-languageserver/node";
export interface ParsedComponent {
    name: string;
    range: Range;
    properties: ParsedProperty[];
    startLine: number;
    endLine: number;
}
export interface ParsedProperty {
    name: string;
    range: Range;
    line: number;
    indentLevel: number;
    isBinding: boolean;
    bindingType?: "one-way" | "two-way" | "override" | undefined;
    value?: string;
}
export interface ParsedNode {
    type: "root_class" | "class" | "comp" | "prop" | "sub_prop";
    name: string;
    range: Range;
    line: number;
    indentLevel: number;
}
export interface ParseResult {
    components: ParsedComponent[];
    nodes: ParsedNode[];
    errors: ParseError[];
}
export interface ParseError {
    message: string;
    range: Range;
    severity: "error" | "warning" | "info";
}
export declare class ViewTreeParser {
    private lines;
    parse(content: string): ParseResult;
    getNodeAtPosition(content: string, position: Position): ParsedNode | null;
    getWordRangeAtPosition(content: string, position: Position): Range | null;
    getCurrentComponent(content: string, position: Position): string | null;
    private getIndentLevel;
    private getWordRange;
    private isPositionInRange;
    private isWordCharacter;
    validateSyntax(content: string): ParseError[];
    private isValidPropertyName;
}
//# sourceMappingURL=view-tree-parser.d.ts.map