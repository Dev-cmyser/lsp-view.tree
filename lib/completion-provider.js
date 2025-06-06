"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.CompletionProvider = void 0;
const node_1 = require("vscode-languageserver/node");
const view_tree_parser_1 = require("./view-tree-parser");
class CompletionProvider {
    constructor(projectScanner) {
        this.projectScanner = projectScanner;
        this.parser = new view_tree_parser_1.ViewTreeParser();
    }
    async provideCompletionItems(document, position) {
        console.log(`[completion] Request at ${position.line}:${position.character}`);
        const content = document.getText();
        const line = content.split('\n')[position.line] || '';
        const beforeCursor = line.substring(0, position.character);
        console.log(`[completion] Line: "${line}", Before cursor: "${beforeCursor}"`);
        const items = [];
        const completionContext = this.getCompletionContext(content, position, beforeCursor);
        console.log(`[completion] Context: ${completionContext.type}, indent: ${completionContext.indentLevel}`);
        switch (completionContext.type) {
            case 'component_name':
                console.log(`[completion] Adding component completions`);
                await this.addComponentCompletions(items);
                break;
            case 'component_extends':
                console.log(`[completion] Adding component extends completions`);
                await this.addComponentCompletions(items);
                break;
            case 'property_name':
                console.log(`[completion] Adding property completions for component: ${completionContext.currentComponent}`);
                this.addPropertyCompletions(items, completionContext.currentComponent);
                break;
            case 'property_binding':
                console.log(`[completion] Adding binding completions`);
                this.addBindingCompletions(items);
                break;
            case 'value':
                console.log(`[completion] Adding value completions`);
                this.addValueCompletions(items);
                await this.addComponentCompletions(items);
                break;
        }
        console.log(`[completion] Returning ${items.length} items`);
        return items;
    }
    getCompletionContext(content, position, beforeCursor) {
        const trimmed = beforeCursor.trim();
        const indentLevel = beforeCursor.length - beforeCursor.trimStart().length;
        // If starts with $ anywhere - it's a component
        if (trimmed.startsWith('$')) {
            return { type: 'component_name', indentLevel, currentComponent: null };
        }
        // If at root level and no space - it's a component
        if (indentLevel === 0 && !trimmed.includes(' ')) {
            return { type: 'component_name', indentLevel, currentComponent: null };
        }
        // If at root level and has space - it's inheritance
        if (indentLevel === 0 && trimmed.includes(' ')) {
            return { type: 'component_extends', indentLevel, currentComponent: null };
        }
        // If has binding operators
        if (trimmed.includes('<=')) {
            return { type: 'property_binding', indentLevel, currentComponent: null };
        }
        // If indented - it's a property
        if (indentLevel > 0) {
            const currentComponent = this.getCurrentComponent(content, position);
            return { type: 'property_name', indentLevel, currentComponent };
        }
        return { type: 'value', indentLevel, currentComponent: null };
    }
    getCurrentComponent(content, position) {
        return this.parser.getCurrentComponent(content, position);
    }
    async addComponentCompletions(items) {
        const projectData = this.projectScanner.getProjectData();
        console.log(`[completion] Project has ${projectData.components.size} components`);
        // Add components from project
        for (const component of projectData.components) {
            const item = {
                label: component,
                kind: node_1.CompletionItemKind.Class,
                insertText: component,
                sortText: '1' + component,
                detail: 'Component',
                documentation: `Component: ${component}`
            };
            items.push(item);
        }
        console.log(`[completion] Added ${projectData.components.size} component completions`);
    }
    addPropertyCompletions(items, currentComponent) {
        const projectData = this.projectScanner.getProjectData();
        // Add properties for current component
        if (currentComponent && projectData.componentProperties.has(currentComponent)) {
            const properties = projectData.componentProperties.get(currentComponent);
            for (const property of properties) {
                const item = {
                    label: property,
                    kind: node_1.CompletionItemKind.Property,
                    insertText: property,
                    sortText: '1' + property,
                    detail: `Property of ${currentComponent}`,
                    documentation: `Property from component ${currentComponent}`
                };
                items.push(item);
            }
        }
        // Add common properties if component not found
        if (!currentComponent) {
            const allProperties = this.projectScanner.getAllProperties();
            for (const property of allProperties) {
                const item = {
                    label: property,
                    kind: node_1.CompletionItemKind.Property,
                    insertText: property,
                    sortText: '2' + property,
                    detail: 'Property',
                    documentation: 'Property from project'
                };
                items.push(item);
            }
        }
        // Add list marker
        const listItem = {
            label: '/',
            kind: node_1.CompletionItemKind.Operator,
            insertText: '/',
            sortText: '0/',
            detail: 'Empty list',
            documentation: 'Creates an empty list'
        };
        items.push(listItem);
        // Add common properties
        this.addCommonProperties(items);
    }
    addCommonProperties(items) {
        const commonProperties = [
            { name: 'dom_name', detail: 'DOM element name' },
            { name: 'dom_name_space', detail: 'DOM namespace' },
            { name: 'attr', detail: 'DOM attributes' },
            { name: 'field', detail: 'Form field' },
            { name: 'value', detail: 'Element value' },
            { name: 'enabled', detail: 'Element enabled state' },
            { name: 'visible', detail: 'Element visibility' },
            { name: 'title', detail: 'Element title' },
            { name: 'hint', detail: 'Element hint' },
            { name: 'sub', detail: 'Sub-elements' },
            { name: 'event', detail: 'Event handlers' },
            { name: 'plugins', detail: 'Plugins' }
        ];
        for (const prop of commonProperties) {
            const item = {
                label: prop.name,
                kind: node_1.CompletionItemKind.Property,
                insertText: prop.name,
                sortText: '3' + prop.name,
                detail: prop.detail,
                documentation: prop.detail
            };
            items.push(item);
        }
    }
    addBindingCompletions(items) {
        const operators = [
            {
                text: '<=',
                detail: 'One-way binding',
                documentation: 'Binds property value from parent to child (one direction)'
            },
            {
                text: '<=>',
                detail: 'Two-way binding',
                documentation: 'Binds property value between parent and child (both directions)'
            },
            {
                text: '^',
                detail: 'Override',
                documentation: 'Overrides property in parent class'
            },
            {
                text: '*',
                detail: 'Multi-property marker',
                documentation: 'Marks property as accepting multiple values'
            }
        ];
        for (const op of operators) {
            const item = {
                label: op.text,
                kind: node_1.CompletionItemKind.Operator,
                insertText: op.text,
                sortText: '0' + op.text,
                detail: op.detail,
                documentation: op.documentation
            };
            items.push(item);
        }
    }
    addValueCompletions(items) {
        const specialValues = [
            {
                text: 'null',
                detail: 'Null value',
                documentation: 'Represents empty/null value'
            },
            {
                text: 'true',
                detail: 'Boolean true',
                documentation: 'Boolean true value'
            },
            {
                text: 'false',
                detail: 'Boolean false',
                documentation: 'Boolean false value'
            },
            {
                text: '\\',
                detail: 'String literal',
                insertText: '\\\n\t\\',
                documentation: 'Multi-line string literal'
            },
            {
                text: '@\\',
                detail: 'Localized string',
                insertText: '@\\\n\t\\',
                documentation: 'Localized multi-line string'
            },
            {
                text: '*',
                detail: 'Dictionary marker',
                documentation: 'Marks property as dictionary'
            }
        ];
        for (const value of specialValues) {
            const item = {
                label: value.text,
                kind: node_1.CompletionItemKind.Value,
                insertText: value.insertText || value.text,
                sortText: '0' + value.text,
                detail: value.detail,
                documentation: value.documentation
            };
            if (value.insertText && value.insertText.includes('\n')) {
                item.insertTextFormat = node_1.InsertTextFormat.Snippet;
            }
            items.push(item);
        }
        // Add CSS classes completion
        this.addCssClassCompletions(items);
        // Add event handler completions
        this.addEventHandlerCompletions(items);
    }
    addCssClassCompletions(items) {
        const cssClasses = [
            'mol_theme_auto',
            'mol_theme_dark',
            'mol_theme_light',
            'mol_skin_auto',
            'mol_skin_dark',
            'mol_skin_light'
        ];
        for (const cssClass of cssClasses) {
            const item = {
                label: cssClass,
                kind: node_1.CompletionItemKind.EnumMember,
                insertText: cssClass,
                sortText: '4' + cssClass,
                detail: 'CSS class',
                documentation: `CSS class: ${cssClass}`
            };
            items.push(item);
        }
    }
    addEventHandlerCompletions(items) {
        const events = [
            'event_click',
            'event_focus',
            'event_blur',
            'event_change',
            'event_input',
            'event_keydown',
            'event_keyup',
            'event_mousedown',
            'event_mouseup',
            'event_mouseover',
            'event_mouseout'
        ];
        for (const event of events) {
            const item = {
                label: event,
                kind: node_1.CompletionItemKind.Event,
                insertText: event,
                sortText: '5' + event,
                detail: 'Event handler',
                documentation: `Event handler: ${event}`
            };
            items.push(item);
        }
    }
}
exports.CompletionProvider = CompletionProvider;
//# sourceMappingURL=completion-provider.js.map