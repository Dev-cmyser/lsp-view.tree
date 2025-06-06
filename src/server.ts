import {
	createConnection,
	TextDocuments,
	ProposedFeatures,
	InitializeParams,
	DidChangeConfigurationNotification,
	CompletionItem,
	TextDocumentPositionParams,
	TextDocumentSyncKind,
	InitializeResult,
	DefinitionParams,
	Location,
	HoverParams,
	Hover
} from 'vscode-languageserver/node';

import { TextDocument } from 'vscode-languageserver-textdocument';
import { URI } from 'vscode-uri';

import { ProjectScanner } from './project-scanner';
import { DefinitionProvider } from './definition-provider';
import { CompletionProvider } from './completion-provider';
import { HoverProvider } from './hover-provider';
import { DiagnosticProvider } from './diagnostic-provider';

// Create a connection for the server, using Node's IPC as a transport.
const connection = createConnection(ProposedFeatures.all);

// Create a simple text document manager.
const documents: TextDocuments<TextDocument> = new TextDocuments(TextDocument);

let hasConfigurationCapability = false;
let hasWorkspaceFolderCapability = false;

// Project scanner instance
let projectScanner: ProjectScanner;
let definitionProvider: DefinitionProvider;
let completionProvider: CompletionProvider;
let hoverProvider: HoverProvider;
let diagnosticProvider: DiagnosticProvider;

connection.onInitialize((params: InitializeParams) => {
	const capabilities = params.capabilities;

	// Does the client support the `workspace/configuration` request?
	hasConfigurationCapability = !!(
		capabilities.workspace && !!capabilities.workspace.configuration
	);
	hasWorkspaceFolderCapability = !!(
		capabilities.workspace && !!capabilities.workspace.workspaceFolders
	);

	const result: InitializeResult = {
		capabilities: {
			textDocumentSync: TextDocumentSyncKind.Incremental,
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
		connection.client.register(DidChangeConfigurationNotification.type, undefined);
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

		const workspaceRoot = URI.parse(firstFolder.uri).fsPath;
		connection.console.log(`[view.tree] Initializing with workspace: ${workspaceRoot}`);

		// Initialize project scanner
		projectScanner = new ProjectScanner(workspaceRoot, connection.console);
		
		// Initialize providers
		definitionProvider = new DefinitionProvider(projectScanner);
		completionProvider = new CompletionProvider(projectScanner);
		hoverProvider = new HoverProvider(projectScanner);
		diagnosticProvider = new DiagnosticProvider(projectScanner);

		// Start initial project scan
		await projectScanner.scanProject();
		
		connection.console.log('[view.tree] LSP server initialized successfully');
	} catch (error) {
		connection.console.error(`[view.tree] Failed to initialize: ${error}`);
	}
}

// The example settings
interface ExampleSettings {
	maxNumberOfProblems: number;
}

// The global settings, used when the `workspace/configuration` request is not supported by the client.
const defaultSettings: ExampleSettings = { maxNumberOfProblems: 1000 };
let globalSettings: ExampleSettings = defaultSettings;

// Cache the settings of all open documents
const documentSettings: Map<string, Thenable<ExampleSettings>> = new Map();

connection.onDidChangeConfiguration(change => {
	if (hasConfigurationCapability) {
		// Reset all cached document settings
		documentSettings.clear();
	} else {
		globalSettings = <ExampleSettings>(
			(change.settings.languageServerExample || defaultSettings)
		);
	}

	// Revalidate all open text documents
	documents.all().forEach(validateTextDocument);
});

function getDocumentSettings(resource: string): Thenable<ExampleSettings> {
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
		const uri = URI.parse(change.document.uri);
		if (uri.fsPath.endsWith('.view.tree') || uri.fsPath.endsWith('.ts')) {
			projectScanner.updateSingleFile(uri.fsPath, change.document.getText());
		}
	}
});

async function validateTextDocument(textDocument: TextDocument): Promise<void> {
	if (!diagnosticProvider) {
		return;
	}

	try {
		const diagnostics = await diagnosticProvider.provideDiagnostics(textDocument);
		connection.sendDiagnostics({ uri: textDocument.uri, diagnostics });
	} catch (error) {
		connection.console.error(`[view.tree] Error validating document: ${error}`);
	}
}

// This handler provides the initial list of the completion items.
connection.onCompletion(
	async (textDocumentPosition: TextDocumentPositionParams): Promise<CompletionItem[]> => {
		if (!completionProvider) {
			return [];
		}

		try {
			const document = documents.get(textDocumentPosition.textDocument.uri);
			if (!document) {
				return [];
			}

			return await completionProvider.provideCompletionItems(document, textDocumentPosition.position);
		} catch (error) {
			connection.console.error(`[view.tree] Error providing completion: ${error}`);
			return [];
		}
	}
);

// This handler resolves additional information for the item selected in
// the completion list.
connection.onCompletionResolve(
	(item: CompletionItem): CompletionItem => {
		// Add additional information if needed
		return item;
	}
);

// Handle go to definition requests
connection.onDefinition(
	async (params: DefinitionParams): Promise<Location[]> => {
		if (!definitionProvider) {
			return [];
		}

		try {
			const document = documents.get(params.textDocument.uri);
			if (!document) {
				return [];
			}

			return await definitionProvider.provideDefinition(document, params.position);
		} catch (error) {
			connection.console.error(`[view.tree] Error providing definition: ${error}`);
			return [];
		}
	}
);

// Handle hover requests
connection.onHover(
	async (params: HoverParams): Promise<Hover | null> => {
		if (!hoverProvider) {
			return null;
		}

		try {
			const document = documents.get(params.textDocument.uri);
			if (!document) {
				return null;
			}

			return await hoverProvider.provideHover(document, params.position);
		} catch (error) {
			connection.console.error(`[view.tree] Error providing hover: ${error}`);
			return null;
		}
	}
);

// Make the text document manager listen on the connection
// for open, change and close text document events
documents.listen(connection);

// Listen on the connection
connection.listen();