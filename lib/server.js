"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const node_1 = require("vscode-languageserver/node");
const vscode_languageserver_textdocument_1 = require("vscode-languageserver-textdocument");
const vscode_uri_1 = require("vscode-uri");
const project_scanner_1 = require("./project-scanner");
const definition_provider_1 = require("./definition-provider");
const completion_provider_1 = require("./completion-provider");
const hover_provider_1 = require("./hover-provider");
const diagnostic_provider_1 = require("./diagnostic-provider");
// Create a connection for the server, using Node's IPC as a transport.
const connection = (0, node_1.createConnection)(node_1.ProposedFeatures.all);
// Create a simple text document manager.
const documents = new node_1.TextDocuments(vscode_languageserver_textdocument_1.TextDocument);
let hasConfigurationCapability = false;
let hasWorkspaceFolderCapability = false;
// Project scanner instance
let projectScanner;
let definitionProvider;
let completionProvider;
let hoverProvider;
let diagnosticProvider;
connection.onInitialize((params) => {
    const capabilities = params.capabilities;
    // Does the client support the `workspace/configuration` request?
    hasConfigurationCapability = !!(capabilities.workspace && !!capabilities.workspace.configuration);
    hasWorkspaceFolderCapability = !!(capabilities.workspace && !!capabilities.workspace.workspaceFolders);
    const result = {
        capabilities: {
            textDocumentSync: node_1.TextDocumentSyncKind.Incremental,
            // Tell the client that this server supports code completion.
            completionProvider: {
                resolveProvider: true,
                triggerCharacters: ['$', '_', ' ', '\t']
            },
            // Tell the client that this server supports go to definition.
            definitionProvider: true,
            // Tell the client that this server supports hover.
            hoverProvider: true,
            // Tell the client that this server supports diagnostics.
            diagnosticProvider: {
                interFileDependencies: true,
                workspaceDiagnostics: true
            }
        }
    };
    if (hasWorkspaceFolderCapability) {
        result.capabilities.workspace = {
            workspaceFolders: {
                supported: true
            }
        };
    }
    return result;
});
connection.onInitialized(() => {
    if (hasConfigurationCapability) {
        // Register for all configuration changes.
        connection.client.register(node_1.DidChangeConfigurationNotification.type, undefined);
    }
    if (hasWorkspaceFolderCapability) {
        connection.workspace.onDidChangeWorkspaceFolders(_event => {
            connection.console.log('Workspace folder change event received.');
        });
    }
    // Initialize project scanner and providers
    initializeProviders();
});
async function initializeProviders() {
    try {
        const workspaceFolders = await connection.workspace.getWorkspaceFolders();
        if (!workspaceFolders || workspaceFolders.length === 0) {
            connection.console.log('[view.tree] No workspace folders found');
            return;
        }
        const firstFolder = workspaceFolders[0];
        if (!firstFolder) {
            connection.console.log('[view.tree] No workspace folders found');
            return;
        }
        const workspaceRoot = vscode_uri_1.URI.parse(firstFolder.uri).fsPath;
        connection.console.log(`[view.tree] Initializing with workspace: ${workspaceRoot}`);
        // Initialize project scanner
        projectScanner = new project_scanner_1.ProjectScanner(workspaceRoot, connection.console);
        // Initialize providers
        definitionProvider = new definition_provider_1.DefinitionProvider(projectScanner);
        completionProvider = new completion_provider_1.CompletionProvider(projectScanner);
        hoverProvider = new hover_provider_1.HoverProvider(projectScanner);
        diagnosticProvider = new diagnostic_provider_1.DiagnosticProvider(projectScanner);
        // Start initial project scan
        await projectScanner.scanProject();
        connection.console.log('[view.tree] LSP server initialized successfully');
    }
    catch (error) {
        connection.console.error(`[view.tree] Failed to initialize: ${error}`);
    }
}
// The global settings, used when the `workspace/configuration` request is not supported by the client.
const defaultSettings = { maxNumberOfProblems: 1000 };
let globalSettings = defaultSettings;
// Cache the settings of all open documents
const documentSettings = new Map();
connection.onDidChangeConfiguration(change => {
    if (hasConfigurationCapability) {
        // Reset all cached document settings
        documentSettings.clear();
    }
    else {
        globalSettings = ((change.settings.languageServerExample || defaultSettings));
    }
    // Revalidate all open text documents
    documents.all().forEach(validateTextDocument);
});
function getDocumentSettings(resource) {
    if (!hasConfigurationCapability) {
        return Promise.resolve(globalSettings);
    }
    let result = documentSettings.get(resource);
    if (!result) {
        result = connection.workspace.getConfiguration({
            scopeUri: resource,
            section: 'languageServerExample'
        });
        documentSettings.set(resource, result);
    }
    return result;
}
// Use the function to avoid "unused" error
void getDocumentSettings;
// Only keep settings for open documents
documents.onDidClose(e => {
    documentSettings.delete(e.document.uri);
});
// The content of a text document has changed. This event is emitted
// when the text document first opened or when its content has changed.
documents.onDidChangeContent(change => {
    validateTextDocument(change.document);
    // Update project data incrementally
    if (projectScanner) {
        const uri = vscode_uri_1.URI.parse(change.document.uri);
        if (uri.fsPath.endsWith('.view.tree') || uri.fsPath.endsWith('.ts')) {
            projectScanner.updateSingleFile(uri.fsPath, change.document.getText());
        }
    }
});
async function validateTextDocument(textDocument) {
    if (!diagnosticProvider) {
        return;
    }
    try {
        const diagnostics = await diagnosticProvider.provideDiagnostics(textDocument);
        connection.sendDiagnostics({ uri: textDocument.uri, diagnostics });
    }
    catch (error) {
        connection.console.error(`[view.tree] Error validating document: ${error}`);
    }
}
// This handler provides the initial list of the completion items.
connection.onCompletion(async (textDocumentPosition) => {
    if (!completionProvider) {
        return [];
    }
    try {
        const document = documents.get(textDocumentPosition.textDocument.uri);
        if (!document) {
            return [];
        }
        return await completionProvider.provideCompletionItems(document, textDocumentPosition.position);
    }
    catch (error) {
        connection.console.error(`[view.tree] Error providing completion: ${error}`);
        return [];
    }
});
// This handler resolves additional information for the item selected in
// the completion list.
connection.onCompletionResolve((item) => {
    // Add additional information if needed
    return item;
});
// Handle go to definition requests
connection.onDefinition(async (params) => {
    if (!definitionProvider) {
        return [];
    }
    try {
        const document = documents.get(params.textDocument.uri);
        if (!document) {
            return [];
        }
        return await definitionProvider.provideDefinition(document, params.position);
    }
    catch (error) {
        connection.console.error(`[view.tree] Error providing definition: ${error}`);
        return [];
    }
});
// Handle hover requests
connection.onHover(async (params) => {
    if (!hoverProvider) {
        return null;
    }
    try {
        const document = documents.get(params.textDocument.uri);
        if (!document) {
            return null;
        }
        return await hoverProvider.provideHover(document, params.position);
    }
    catch (error) {
        connection.console.error(`[view.tree] Error providing hover: ${error}`);
        return null;
    }
});
// Make the text document manager listen on the connection
// for open, change and close text document events
documents.listen(connection);
// Listen on the connection
connection.listen();
//# sourceMappingURL=server.js.map