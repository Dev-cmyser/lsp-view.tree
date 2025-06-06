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

export class ViewTreeParser {
  private lines: string[] = [];

  parse(content: string): ParseResult {
    this.lines = content.split("\n");

    const result: ParseResult = {
      components: [],
      nodes: [],
      errors: [],
    };

    let currentComponent: ParsedComponent | null = null;

    for (let lineIndex = 0; lineIndex < this.lines.length; lineIndex++) {
      const line = this.lines[lineIndex];
      if (!line) continue;
      const trimmed = line.trim();

      // Skip empty lines and comments
      if (!trimmed || trimmed.startsWith("//")) {
        continue;
      }

      const indentLevel = this.getIndentLevel(line);

      // Root level component definition
      if (indentLevel === 0 && trimmed.startsWith("$")) {
        // Finish previous component
        if (currentComponent) {
          currentComponent.endLine = lineIndex - 1;
          result.components.push(currentComponent);
        }

        // Start new component
        const firstWord = trimmed.split(/\s+/)[0];
        if (!firstWord) continue;
        const wordRange = this.getWordRange(lineIndex, line.indexOf(firstWord), firstWord);

        currentComponent = {
          name: firstWord,
          range: wordRange,
          properties: [],
          startLine: lineIndex,
          endLine: lineIndex,
        };

        // Add node for root class
        const nodeType = lineIndex === 0 && wordRange.start.character === 1 ? "root_class" : "class";
        result.nodes.push({
          type: nodeType,
          name: firstWord,
          range: wordRange,
          line: lineIndex,
          indentLevel: 0,
        });
      }
      // Property or sub-component
      else if (indentLevel > 0 && currentComponent) {
        const wordMatch = line.match(/^(\s+)([a-zA-Z_$][a-zA-Z0-9_?*]*)/);
        if (wordMatch) {
          const propertyName = wordMatch[2];
          if (!propertyName) continue;
          const propertyStart = line.indexOf(propertyName);
          const wordRange = this.getWordRange(lineIndex, propertyStart, propertyName);

          // Determine if it's a binding
          const isBinding = trimmed.includes("<=") || trimmed.includes("<=>");
          let bindingType: "one-way" | "two-way" | "override" | undefined;
          let value: string | undefined;

          if (isBinding) {
            if (trimmed.includes("<=>")) {
              bindingType = "two-way";
            } else if (trimmed.includes("<=")) {
              bindingType = "one-way";
            }

            // Extract bound property name
            const bindingMatch = trimmed.match(/<=>\s*([a-zA-Z_][a-zA-Z0-9_?*]*)|<=\s*([a-zA-Z_][a-zA-Z0-9_?*]*)/);
            if (bindingMatch) {
              value = bindingMatch[1] || bindingMatch[2];
            }
          } else if (trimmed.includes("^")) {
            bindingType = "override";
          } else {
            // Extract value after property name
            const valueMatch = trimmed.match(/^[a-zA-Z_$][a-zA-Z0-9_?*]*\s+(.+)$/);
            if (valueMatch && valueMatch[1]) {
              value = valueMatch[1].trim();
            }
          }

          const property: ParsedProperty = {
            name: propertyName,
            range: wordRange,
            line: lineIndex,
            indentLevel,
            isBinding,
            bindingType,
            value,
          };

          currentComponent.properties.push(property);

          // Determine node type
          let nodeType: "comp" | "prop" | "sub_prop";
          if (propertyName.startsWith("$")) {
            nodeType = "comp";
          } else if (indentLevel === 1) {
            nodeType = "prop";
          } else {
            nodeType = "sub_prop";
          }

          result.nodes.push({
            type: nodeType,
            name: propertyName,
            range: wordRange,
            line: lineIndex,
            indentLevel,
          });
        }
      }
      // Error: indented line without current component
      else if (indentLevel > 0 && !currentComponent) {
        const errorRange = Range.create(Position.create(lineIndex, 0), Position.create(lineIndex, line.length));
        result.errors.push({
          message: "Property defined outside of component",
          range: errorRange,
          severity: "error",
        });
      }
    }

    // Finish last component
    if (currentComponent) {
      currentComponent.endLine = this.lines.length - 1;
      result.components.push(currentComponent);
    }

    return result;
  }

  getNodeAtPosition(content: string, position: Position): ParsedNode | null {
    const parseResult = this.parse(content);

    for (const node of parseResult.nodes) {
      if (this.isPositionInRange(position, node.range)) {
        return node;
      }
    }

    return null;
  }

  getWordRangeAtPosition(content: string, position: Position): Range | null {
    this.lines = content.split("\n");

    if (position.line >= this.lines.length) {
      return null;
    }

    const line = this.lines[position.line];
    if (!line) {
      return null;
    }
    const character = position.character;

    // Find word boundaries
    let start = character;
    let end = character;

    // Move start backwards to find word start
    while (start > 0 && line[start - 1] && this.isWordCharacter(line[start - 1])) {
      start--;
    }

    // Move end forwards to find word end
    while (end < line.length && line[end] && this.isWordCharacter(line[end])) {
      end++;
    }

    if (start === end) {
      return null;
    }

    return Range.create(Position.create(position.line, start), Position.create(position.line, end));
  }

  getCurrentComponent(content: string, position: Position): string | null {
    this.lines = content.split("\n");

    // Look backwards from current position to find the component
    for (let i = position.line; i >= 0; i--) {
      const line = this.lines[i];
      if (!line) continue;
      const trimmed = line.trim();

      // If line has no indentation and starts with $
      if (!line.startsWith("\t") && !line.startsWith(" ") && trimmed.startsWith("$")) {
        const firstWord = trimmed.split(/\s+/)[0];
        if (firstWord && firstWord.startsWith("$")) {
          return firstWord;
        }
      }
    }

    return null;
  }

  private getIndentLevel(line: string): number {
    let indent = 0;
    for (const char of line) {
      if (char === "\t") {
        indent++;
      } else if (char === " ") {
        indent++; // Could be adjusted for different space-to-tab ratios
      } else {
        break;
      }
    }
    return indent;
  }

  private getWordRange(line: number, start: number, word: string): Range {
    return Range.create(Position.create(line, start), Position.create(line, start + word.length));
  }

  private isPositionInRange(position: Position, range: Range): boolean {
    if (position.line < range.start.line || position.line > range.end.line) {
      return false;
    }

    if (position.line === range.start.line && position.character < range.start.character) {
      return false;
    }

    if (position.line === range.end.line && position.character > range.end.character) {
      return false;
    }

    return true;
  }

  private isWordCharacter(char: string): boolean {
    return /[a-zA-Z0-9_$?*]/.test(char);
  }

  // Helper method to validate view.tree syntax
  validateSyntax(content: string): ParseError[] {
    const parseResult = this.parse(content);
    const errors: ParseError[] = [...parseResult.errors];

    // Add additional validation rules
    for (const component of parseResult.components) {
      // Check for duplicate component names
      const duplicates = parseResult.components.filter((c) => c.name === component.name);
      if (duplicates.length > 1) {
        errors.push({
          message: `Duplicate component name: ${component.name}`,
          range: component.range,
          severity: "warning",
        });
      }

      // Check for invalid property names
      for (const property of component.properties) {
        if (!this.isValidPropertyName(property.name)) {
          errors.push({
            message: `Invalid property name: ${property.name}`,
            range: property.range,
            severity: "error",
          });
        }
      }
    }

    return errors;
  }

  private isValidPropertyName(name: string): boolean {
    // Basic validation - starts with letter or underscore, contains only alphanumeric, underscore, ?, *
    return /^[a-zA-Z_$][a-zA-Z0-9_?*]*$/.test(name);
  }
}
