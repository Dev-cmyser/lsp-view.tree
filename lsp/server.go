package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
)

// LSP Message structures
type LSPMessage struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method,omitempty"`
	Params  interface{} `json:"params,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *LSPError   `json:"error,omitempty"`
}

type LSPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// LSP Protocol structures
type InitializeParams struct {
	ProcessID             *int                   `json:"processId"`
	ClientInfo            *ClientInfo            `json:"clientInfo,omitempty"`
	Locale                string                 `json:"locale,omitempty"`
	RootPath              *string                `json:"rootPath,omitempty"`
	RootURI               *string                `json:"rootUri"`
	InitializationOptions interface{}            `json:"initializationOptions,omitempty"`
	Capabilities          ClientCapabilities     `json:"capabilities"`
	Trace                 string                 `json:"trace,omitempty"`
	WorkspaceFolders      []WorkspaceFolder      `json:"workspaceFolders,omitempty"`
}

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type ClientCapabilities struct {
	Workspace    *WorkspaceClientCapabilities    `json:"workspace,omitempty"`
	TextDocument *TextDocumentClientCapabilities `json:"textDocument,omitempty"`
	Window       *WindowClientCapabilities       `json:"window,omitempty"`
	General      *GeneralClientCapabilities      `json:"general,omitempty"`
}

type WorkspaceClientCapabilities struct {
	ApplyEdit              bool                        `json:"applyEdit,omitempty"`
	WorkspaceEdit          *WorkspaceEditCapabilities  `json:"workspaceEdit,omitempty"`
	DidChangeConfiguration *DidChangeConfigurationCapabilities `json:"didChangeConfiguration,omitempty"`
	DidChangeWatchedFiles  *DidChangeWatchedFilesCapabilities `json:"didChangeWatchedFiles,omitempty"`
	Symbol                 *WorkspaceSymbolCapabilities `json:"symbol,omitempty"`
	ExecuteCommand         *ExecuteCommandCapabilities `json:"executeCommand,omitempty"`
	Configuration          bool                        `json:"configuration,omitempty"`
	WorkspaceFolders       bool                        `json:"workspaceFolders,omitempty"`
}

type WorkspaceEditCapabilities struct {
	DocumentChanges    bool     `json:"documentChanges,omitempty"`
	ResourceOperations []string `json:"resourceOperations,omitempty"`
	FailureHandling    string   `json:"failureHandling,omitempty"`
}

type DidChangeConfigurationCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

type DidChangeWatchedFilesCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

type WorkspaceSymbolCapabilities struct {
	DynamicRegistration bool                              `json:"dynamicRegistration,omitempty"`
	SymbolKind          *WorkspaceSymbolKindCapabilities  `json:"symbolKind,omitempty"`
	TagSupport          *WorkspaceSymbolTagCapabilities   `json:"tagSupport,omitempty"`
}

type WorkspaceSymbolKindCapabilities struct {
	ValueSet []int `json:"valueSet,omitempty"`
}

type WorkspaceSymbolTagCapabilities struct {
	ValueSet []int `json:"valueSet,omitempty"`
}

type ExecuteCommandCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

type TextDocumentClientCapabilities struct {
	Synchronization    *TextDocumentSyncCapabilities    `json:"synchronization,omitempty"`
	Completion         *CompletionCapabilities          `json:"completion,omitempty"`
	Hover              *HoverCapabilities               `json:"hover,omitempty"`
	SignatureHelp      *SignatureHelpCapabilities       `json:"signatureHelp,omitempty"`
	Declaration        *DeclarationCapabilities         `json:"declaration,omitempty"`
	Definition         *DefinitionCapabilities          `json:"definition,omitempty"`
	TypeDefinition     *TypeDefinitionCapabilities      `json:"typeDefinition,omitempty"`
	Implementation     *ImplementationCapabilities      `json:"implementation,omitempty"`
	References         *ReferencesCapabilities          `json:"references,omitempty"`
	DocumentHighlight  *DocumentHighlightCapabilities   `json:"documentHighlight,omitempty"`
	DocumentSymbol     *DocumentSymbolCapabilities      `json:"documentSymbol,omitempty"`
	CodeAction         *CodeActionCapabilities          `json:"codeAction,omitempty"`
	CodeLens           *CodeLensCapabilities            `json:"codeLens,omitempty"`
	DocumentLink       *DocumentLinkCapabilities        `json:"documentLink,omitempty"`
	ColorProvider      *DocumentColorCapabilities       `json:"colorProvider,omitempty"`
	Formatting         *DocumentFormattingCapabilities  `json:"formatting,omitempty"`
	RangeFormatting    *DocumentRangeFormattingCapabilities `json:"rangeFormatting,omitempty"`
	OnTypeFormatting   *DocumentOnTypeFormattingCapabilities `json:"onTypeFormatting,omitempty"`
	Rename             *RenameCapabilities              `json:"rename,omitempty"`
	PublishDiagnostics *PublishDiagnosticsCapabilities  `json:"publishDiagnostics,omitempty"`
	FoldingRange       *FoldingRangeCapabilities        `json:"foldingRange,omitempty"`
	SelectionRange     *SelectionRangeCapabilities      `json:"selectionRange,omitempty"`
}

type TextDocumentSyncCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	WillSave            bool `json:"willSave,omitempty"`
	WillSaveWaitUntil   bool `json:"willSaveWaitUntil,omitempty"`
	DidSave             bool `json:"didSave,omitempty"`
}

type CompletionCapabilities struct {
	DynamicRegistration bool                      `json:"dynamicRegistration,omitempty"`
	CompletionItem      *CompletionItemCapabilities `json:"completionItem,omitempty"`
	CompletionItemKind  *CompletionItemKindCapabilities `json:"completionItemKind,omitempty"`
	ContextSupport      bool                      `json:"contextSupport,omitempty"`
}

type CompletionItemCapabilities struct {
	SnippetSupport          bool     `json:"snippetSupport,omitempty"`
	CommitCharactersSupport bool     `json:"commitCharactersSupport,omitempty"`
	DocumentationFormat     []string `json:"documentationFormat,omitempty"`
	DeprecatedSupport       bool     `json:"deprecatedSupport,omitempty"`
	PreselectSupport        bool     `json:"preselectSupport,omitempty"`
	TagSupport              *CompletionItemTagCapabilities `json:"tagSupport,omitempty"`
	InsertReplaceSupport    bool     `json:"insertReplaceSupport,omitempty"`
	ResolveSupport          *CompletionItemResolveCapabilities `json:"resolveSupport,omitempty"`
	InsertTextModeSupport   *CompletionItemInsertTextModeCapabilities `json:"insertTextModeSupport,omitempty"`
}

type CompletionItemTagCapabilities struct {
	ValueSet []int `json:"valueSet"`
}

type CompletionItemResolveCapabilities struct {
	Properties []string `json:"properties"`
}

type CompletionItemInsertTextModeCapabilities struct {
	ValueSet []int `json:"valueSet"`
}

type CompletionItemKindCapabilities struct {
	ValueSet []int `json:"valueSet,omitempty"`
}

type HoverCapabilities struct {
	DynamicRegistration bool     `json:"dynamicRegistration,omitempty"`
	ContentFormat       []string `json:"contentFormat,omitempty"`
}

type SignatureHelpCapabilities struct {
	DynamicRegistration bool                             `json:"dynamicRegistration,omitempty"`
	SignatureInformation *SignatureInformationCapabilities `json:"signatureInformation,omitempty"`
	ContextSupport      bool                             `json:"contextSupport,omitempty"`
}

type SignatureInformationCapabilities struct {
	DocumentationFormat []string                           `json:"documentationFormat,omitempty"`
	ParameterInformation *ParameterInformationCapabilities `json:"parameterInformation,omitempty"`
	ActiveParameterSupport bool                           `json:"activeParameterSupport,omitempty"`
}

type ParameterInformationCapabilities struct {
	LabelOffsetSupport bool `json:"labelOffsetSupport,omitempty"`
}

type DeclarationCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	LinkSupport         bool `json:"linkSupport,omitempty"`
}

type DefinitionCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	LinkSupport         bool `json:"linkSupport,omitempty"`
}

type TypeDefinitionCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	LinkSupport         bool `json:"linkSupport,omitempty"`
}

type ImplementationCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	LinkSupport         bool `json:"linkSupport,omitempty"`
}

type ReferencesCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

type DocumentHighlightCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

type DocumentSymbolCapabilities struct {
	DynamicRegistration    bool                               `json:"dynamicRegistration,omitempty"`
	SymbolKind             *DocumentSymbolKindCapabilities    `json:"symbolKind,omitempty"`
	HierarchicalDocumentSymbolSupport bool                  `json:"hierarchicalDocumentSymbolSupport,omitempty"`
	TagSupport             *DocumentSymbolTagCapabilities     `json:"tagSupport,omitempty"`
	LabelSupport           bool                               `json:"labelSupport,omitempty"`
}

type DocumentSymbolKindCapabilities struct {
	ValueSet []int `json:"valueSet,omitempty"`
}

type DocumentSymbolTagCapabilities struct {
	ValueSet []int `json:"valueSet,omitempty"`
}

type CodeActionCapabilities struct {
	DynamicRegistration bool                           `json:"dynamicRegistration,omitempty"`
	CodeActionLiteralSupport *CodeActionLiteralCapabilities `json:"codeActionLiteralSupport,omitempty"`
	IsPreferredSupport  bool                           `json:"isPreferredSupport,omitempty"`
	DisabledSupport     bool                           `json:"disabledSupport,omitempty"`
	DataSupport         bool                           `json:"dataSupport,omitempty"`
	ResolveSupport      *CodeActionResolveCapabilities `json:"resolveSupport,omitempty"`
	HonorsChangeAnnotations bool                      `json:"honorsChangeAnnotations,omitempty"`
}

type CodeActionLiteralCapabilities struct {
	CodeActionKind *CodeActionKindCapabilities `json:"codeActionKind"`
}

type CodeActionKindCapabilities struct {
	ValueSet []string `json:"valueSet"`
}

type CodeActionResolveCapabilities struct {
	Properties []string `json:"properties"`
}

type CodeLensCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

type DocumentLinkCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	TooltipSupport      bool `json:"tooltipSupport,omitempty"`
}

type DocumentColorCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

type DocumentFormattingCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

type DocumentRangeFormattingCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

type DocumentOnTypeFormattingCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

type RenameCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	PrepareSupport      bool `json:"prepareSupport,omitempty"`
	PrepareSupportDefaultBehavior int `json:"prepareSupportDefaultBehavior,omitempty"`
	HonorsChangeAnnotations bool `json:"honorsChangeAnnotations,omitempty"`
}

type PublishDiagnosticsCapabilities struct {
	RelatedInformation      bool `json:"relatedInformation,omitempty"`
	TagSupport              *PublishDiagnosticsTagCapabilities `json:"tagSupport,omitempty"`
	VersionSupport          bool `json:"versionSupport,omitempty"`
	CodeDescriptionSupport  bool `json:"codeDescriptionSupport,omitempty"`
	DataSupport             bool `json:"dataSupport,omitempty"`
}

type PublishDiagnosticsTagCapabilities struct {
	ValueSet []int `json:"valueSet"`
}

type FoldingRangeCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	RangeLimit          int  `json:"rangeLimit,omitempty"`
	LineFoldingOnly     bool `json:"lineFoldingOnly,omitempty"`
}

type SelectionRangeCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

type WindowClientCapabilities struct {
	WorkDoneProgress bool `json:"workDoneProgress,omitempty"`
	ShowMessage      *ShowMessageRequestCapabilities `json:"showMessage,omitempty"`
	ShowDocument     *ShowDocumentCapabilities       `json:"showDocument,omitempty"`
}

type ShowMessageRequestCapabilities struct {
	MessageActionItem *ShowMessageRequestActionItemCapabilities `json:"messageActionItem,omitempty"`
}

type ShowMessageRequestActionItemCapabilities struct {
	AdditionalPropertiesSupport bool `json:"additionalPropertiesSupport,omitempty"`
}

type ShowDocumentCapabilities struct {
	Support bool `json:"support"`
}

type GeneralClientCapabilities struct {
	RegularExpressions *RegularExpressionsCapabilities `json:"regularExpressions,omitempty"`
	Markdown           *MarkdownCapabilities            `json:"markdown,omitempty"`
}

type RegularExpressionsCapabilities struct {
	Engine  string `json:"engine"`
	Version string `json:"version,omitempty"`
}

type MarkdownCapabilities struct {
	Parser  string   `json:"parser"`
	Version string   `json:"version,omitempty"`
	AllowedTags []string `json:"allowedTags,omitempty"`
}

type WorkspaceFolder struct {
	URI  string `json:"uri"`
	Name string `json:"name"`
}

type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   *ServerInfo        `json:"serverInfo,omitempty"`
}

type ServerCapabilities struct {
	TextDocumentSync                 interface{}                    `json:"textDocumentSync,omitempty"`
	CompletionProvider               *CompletionOptions             `json:"completionProvider,omitempty"`
	HoverProvider                    interface{}                    `json:"hoverProvider,omitempty"`
	SignatureHelpProvider            *SignatureHelpOptions          `json:"signatureHelpProvider,omitempty"`
	DeclarationProvider              interface{}                    `json:"declarationProvider,omitempty"`
	DefinitionProvider               interface{}                    `json:"definitionProvider,omitempty"`
	TypeDefinitionProvider           interface{}                    `json:"typeDefinitionProvider,omitempty"`
	ImplementationProvider           interface{}                    `json:"implementationProvider,omitempty"`
	ReferencesProvider               interface{}                    `json:"referencesProvider,omitempty"`
	DocumentHighlightProvider        interface{}                    `json:"documentHighlightProvider,omitempty"`
	DocumentSymbolProvider           interface{}                    `json:"documentSymbolProvider,omitempty"`
	CodeActionProvider               interface{}                    `json:"codeActionProvider,omitempty"`
	CodeLensProvider                 *CodeLensOptions               `json:"codeLensProvider,omitempty"`
	DocumentLinkProvider             *DocumentLinkOptions           `json:"documentLinkProvider,omitempty"`
	ColorProvider                    interface{}                    `json:"colorProvider,omitempty"`
	DocumentFormattingProvider       interface{}                    `json:"documentFormattingProvider,omitempty"`
	DocumentRangeFormattingProvider  interface{}                    `json:"documentRangeFormattingProvider,omitempty"`
	DocumentOnTypeFormattingProvider *DocumentOnTypeFormattingOptions `json:"documentOnTypeFormattingProvider,omitempty"`
	RenameProvider                   interface{}                    `json:"renameProvider,omitempty"`
	FoldingRangeProvider             interface{}                    `json:"foldingRangeProvider,omitempty"`
	ExecuteCommandProvider           *ExecuteCommandOptions         `json:"executeCommandProvider,omitempty"`
	SelectionRangeProvider           interface{}                    `json:"selectionRangeProvider,omitempty"`
	WorkspaceSymbolProvider          interface{}                    `json:"workspaceSymbolProvider,omitempty"`
	Workspace                        *WorkspaceServerCapabilities   `json:"workspace,omitempty"`
	Experimental                     interface{}                    `json:"experimental,omitempty"`
}

type CompletionOptions struct {
	TriggerCharacters   []string `json:"triggerCharacters,omitempty"`
	AllCommitCharacters []string `json:"allCommitCharacters,omitempty"`
	ResolveProvider     bool     `json:"resolveProvider,omitempty"`
	CompletionItem      *ServerCompletionItemOptions `json:"completionItem,omitempty"`
}

type ServerCompletionItemOptions struct {
	LabelDetailsSupport bool `json:"labelDetailsSupport,omitempty"`
}

type SignatureHelpOptions struct {
	TriggerCharacters   []string `json:"triggerCharacters,omitempty"`
	RetriggerCharacters []string `json:"retriggerCharacters,omitempty"`
}

type CodeLensOptions struct {
	ResolveProvider bool `json:"resolveProvider,omitempty"`
}

type DocumentLinkOptions struct {
	ResolveProvider bool `json:"resolveProvider,omitempty"`
}

type DocumentOnTypeFormattingOptions struct {
	FirstTriggerCharacter string   `json:"firstTriggerCharacter"`
	MoreTriggerCharacter  []string `json:"moreTriggerCharacter,omitempty"`
}

type ExecuteCommandOptions struct {
	Commands []string `json:"commands"`
}

type WorkspaceServerCapabilities struct {
	WorkspaceFolders *WorkspaceFoldersServerCapabilities `json:"workspaceFolders,omitempty"`
	FileOperations   *FileOperationOptions               `json:"fileOperations,omitempty"`
}

type WorkspaceFoldersServerCapabilities struct {
	Supported           bool   `json:"supported,omitempty"`
	ChangeNotifications interface{} `json:"changeNotifications,omitempty"`
}

type FileOperationOptions struct {
	DidCreate    *FileOperationRegistrationOptions `json:"didCreate,omitempty"`
	WillCreate   *FileOperationRegistrationOptions `json:"willCreate,omitempty"`
	DidRename    *FileOperationRegistrationOptions `json:"didRename,omitempty"`
	WillRename   *FileOperationRegistrationOptions `json:"willRename,omitempty"`
	DidDelete    *FileOperationRegistrationOptions `json:"didDelete,omitempty"`
	WillDelete   *FileOperationRegistrationOptions `json:"willDelete,omitempty"`
}

type FileOperationRegistrationOptions struct {
	Filters []FileOperationFilter `json:"filters"`
}

type FileOperationFilter struct {
	Scheme  string                `json:"scheme,omitempty"`
	Pattern FileOperationPattern  `json:"pattern"`
}

type FileOperationPattern struct {
	Glob    string                      `json:"glob"`
	Matches FileOperationPatternKind    `json:"matches,omitempty"`
	Options *FileOperationPatternOptions `json:"options,omitempty"`
}

type FileOperationPatternKind string

const (
	FileOperationPatternKindFile   FileOperationPatternKind = "file"
	FileOperationPatternKindFolder FileOperationPatternKind = "folder"
)

type FileOperationPatternOptions struct {
	IgnoreCase bool `json:"ignoreCase,omitempty"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// Text Document structures
type TextDocumentSyncKind int

const (
	TextDocumentSyncKindNone        TextDocumentSyncKind = 0
	TextDocumentSyncKindFull        TextDocumentSyncKind = 1
	TextDocumentSyncKindIncremental TextDocumentSyncKind = 2
)

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

type VersionedTextDocumentIdentifier struct {
	TextDocumentIdentifier
	Version int `json:"version"`
}

type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type DefinitionParams struct {
	TextDocumentPositionParams
	WorkDoneProgressParams
	PartialResultParams
}

type WorkDoneProgressParams struct {
	WorkDoneToken interface{} `json:"workDoneToken,omitempty"`
}

type PartialResultParams struct {
	PartialResultToken interface{} `json:"partialResultToken,omitempty"`
}

type HoverParams struct {
	TextDocumentPositionParams
	WorkDoneProgressParams
}

type Hover struct {
	Contents MarkupContent `json:"contents"`
	Range    *Range        `json:"range,omitempty"`
}

type MarkupContent struct {
	Kind  MarkupKind `json:"kind"`
	Value string     `json:"value"`
}

type MarkupKind string

const (
	MarkupKindPlainText MarkupKind = "plaintext"
	MarkupKindMarkdown  MarkupKind = "markdown"
)

type CompletionParams struct {
	TextDocumentPositionParams
	WorkDoneProgressParams
	PartialResultParams
	Context *CompletionContext `json:"context,omitempty"`
}

type CompletionContext struct {
	TriggerKind      CompletionTriggerKind `json:"triggerKind"`
	TriggerCharacter string                `json:"triggerCharacter,omitempty"`
}

type CompletionTriggerKind int

const (
	CompletionTriggerKindInvoked                CompletionTriggerKind = 1
	CompletionTriggerKindTriggerCharacter       CompletionTriggerKind = 2
	CompletionTriggerKindTriggerForIncompleteCompletions CompletionTriggerKind = 3
)

type CompletionItem struct {
	Label               string                 `json:"label"`
	LabelDetails        *CompletionItemLabelDetails `json:"labelDetails,omitempty"`
	Kind                CompletionItemKind     `json:"kind,omitempty"`
	Tags                []CompletionItemTag    `json:"tags,omitempty"`
	Detail              string                 `json:"detail,omitempty"`
	Documentation       interface{}            `json:"documentation,omitempty"`
	Deprecated          bool                   `json:"deprecated,omitempty"`
	Preselect           bool                   `json:"preselect,omitempty"`
	SortText            string                 `json:"sortText,omitempty"`
	FilterText          string                 `json:"filterText,omitempty"`
	InsertText          string                 `json:"insertText,omitempty"`
	InsertTextFormat    InsertTextFormat       `json:"insertTextFormat,omitempty"`
	InsertTextMode      InsertTextMode         `json:"insertTextMode,omitempty"`
	TextEdit            interface{}            `json:"textEdit,omitempty"`
	AdditionalTextEdits []TextEdit             `json:"additionalTextEdits,omitempty"`
	CommitCharacters    []string               `json:"commitCharacters,omitempty"`
	Command             *Command               `json:"command,omitempty"`
	Data                interface{}            `json:"data,omitempty"`
}

type CompletionItemLabelDetails struct {
	Detail      string `json:"detail,omitempty"`
	Description string `json:"description,omitempty"`
}

type CompletionItemKind int

const (
	CompletionItemKindText          CompletionItemKind = 1
	CompletionItemKindMethod        CompletionItemKind = 2
	CompletionItemKindFunction      CompletionItemKind = 3
	CompletionItemKindConstructor   CompletionItemKind = 4
	CompletionItemKindField         CompletionItemKind = 5
	CompletionItemKindVariable      CompletionItemKind = 6
	CompletionItemKindClass         CompletionItemKind = 7
	CompletionItemKindInterface     CompletionItemKind = 8
	CompletionItemKindModule        CompletionItemKind = 9
	CompletionItemKindProperty      CompletionItemKind = 10
	CompletionItemKindUnit          CompletionItemKind = 11
	CompletionItemKindValue         CompletionItemKind = 12
	CompletionItemKindEnum          CompletionItemKind = 13
	CompletionItemKindKeyword       CompletionItemKind = 14
	CompletionItemKindSnippet       CompletionItemKind = 15
	CompletionItemKindColor         CompletionItemKind = 16
	CompletionItemKindFile          CompletionItemKind = 17
	CompletionItemKindReference     CompletionItemKind = 18
	CompletionItemKindFolder        CompletionItemKind = 19
	CompletionItemKindEnumMember    CompletionItemKind = 20
	CompletionItemKindConstant      CompletionItemKind = 21
	CompletionItemKindStruct        CompletionItemKind = 22
	CompletionItemKindEvent         CompletionItemKind = 23
	CompletionItemKindOperator      CompletionItemKind = 24
	CompletionItemKindTypeParameter CompletionItemKind = 25
)

type CompletionItemTag int

const (
	CompletionItemTagDeprecated CompletionItemTag = 1
)

type InsertTextFormat int

const (
	InsertTextFormatPlainText InsertTextFormat = 1
	InsertTextFormatSnippet   InsertTextFormat = 2
)

type InsertTextMode int

const (
	InsertTextModeAsIs              InsertTextMode = 1
	InsertTextModeAdjustIndentation InsertTextMode = 2
)

type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

type Command struct {
	Title     string        `json:"title"`
	Command   string        `json:"command"`
	Arguments []interface{} `json:"arguments,omitempty"`
}

type CompletionList struct {
	IsIncomplete bool             `json:"isIncomplete"`
	Items        []CompletionItem `json:"items"`
}

// Diagnostic structures
type Diagnostic struct {
	Range              Range                  `json:"range"`
	Severity           DiagnosticSeverity     `json:"severity,omitempty"`
	Code               interface{}            `json:"code,omitempty"`
	CodeDescription    *CodeDescription       `json:"codeDescription,omitempty"`
	Source             string                 `json:"source,omitempty"`
	Message            string                 `json:"message"`
	Tags               []DiagnosticTag        `json:"tags,omitempty"`
	RelatedInformation []DiagnosticRelatedInformation `json:"relatedInformation,omitempty"`
	Data               interface{}            `json:"data,omitempty"`
}

type DiagnosticSeverity int

const (
	DiagnosticSeverityError       DiagnosticSeverity = 1
	DiagnosticSeverityWarning     DiagnosticSeverity = 2
	DiagnosticSeverityInformation DiagnosticSeverity = 3
	DiagnosticSeverityHint        DiagnosticSeverity = 4
)

type DiagnosticTag int

const (
	DiagnosticTagUnnecessary DiagnosticTag = 1
	DiagnosticTagDeprecated  DiagnosticTag = 2
)

type CodeDescription struct {
	Href string `json:"href"`
}

type DiagnosticRelatedInformation struct {
	Location Location `json:"location"`
	Message  string   `json:"message"`
}

type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Version     *int         `json:"version,omitempty"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// Document Change structures
type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

type TextDocumentContentChangeEvent struct {
	Range       *Range `json:"range,omitempty"`
	RangeLength *int   `json:"rangeLength,omitempty"`
	Text        string `json:"text"`
}

type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// Server struct and main implementation
type Server struct {
	reader io.Reader
	writer io.Writer

	// Client capabilities
	hasConfigurationCapability   bool
	hasWorkspaceFolderCapability bool

	// Workspace info
	workspaceRoot string

	// Document store
	documents sync.Map

	// Providers
	projectScanner     *ProjectScanner
	definitionProvider *DefinitionProvider
	completionProvider *CompletionProvider
	hoverProvider      *HoverProvider
	diagnosticProvider *DiagnosticProvider
}

type TextDocument struct {
	URI        string
	LanguageID string
	Version    int
	Text       string
}

func NewServer() *Server {
	return &Server{
		reader: os.Stdin,
		writer: os.Stdout,
	}
}

func (s *Server) Run() error {
	log.Println("[view.tree] Server starting...")
	
	reader := bufio.NewReader(s.reader)
	
	for {
		// Read headers until empty line
		var contentLength int
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}
			
			line = strings.TrimSpace(line)
			if line == "" {
				// Empty line marks end of headers
				break
			}
			
			if strings.HasPrefix(line, "Content-Length: ") {
				lengthStr := strings.TrimPrefix(line, "Content-Length: ")
				length, err := strconv.Atoi(strings.TrimSpace(lengthStr))
				if err != nil {
					log.Printf("[view.tree] Invalid Content-Length: %v", err)
					continue
				}
				contentLength = length
			}
		}
		
		if contentLength == 0 {
			log.Printf("[view.tree] No Content-Length header found")
			continue
		}
		
		// Read message content
		content := make([]byte, contentLength)
		_, err := io.ReadFull(reader, content)
		if err != nil {
			log.Printf("[view.tree] Error reading message content: %v", err)
			continue
		}
		
		if err := s.handleMessage(content); err != nil {
			log.Printf("[view.tree] Error handling message: %v", err)
		}
	}
}

func (s *Server) handleMessage(content []byte) error {
	var msg LSPMessage
	if err := json.Unmarshal(content, &msg); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}
	
	log.Printf("[view.tree] Received %s", msg.Method)
	
	switch msg.Method {
	case "initialize":
		return s.handleInitialize(msg)
	case "initialized":
		return s.handleInitialized(msg)
	case "textDocument/didOpen":
		return s.handleDidOpen(msg)
	case "textDocument/didChange":
		return s.handleDidChange(msg)
	case "textDocument/didClose":
		return s.handleDidClose(msg)
	case "textDocument/completion":
		return s.handleCompletion(msg)
	case "textDocument/definition":
		return s.handleDefinition(msg)
	case "textDocument/hover":
		return s.handleHover(msg)
	case "shutdown":
		return s.handleShutdown(msg)
	case "exit":
		os.Exit(0)
	default:
		log.Printf("[view.tree] Unhandled method: %s", msg.Method)
	}
	
	return nil
}

func (s *Server) sendResponse(id interface{}, result interface{}) error {
	response := LSPMessage{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	
	return s.sendMessage(response)
}

func (s *Server) sendNotification(method string, params interface{}) error {
	notification := LSPMessage{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	
	return s.sendMessage(notification)
}

func (s *Server) sendMessage(msg LSPMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	
	if _, err := s.writer.Write([]byte(header)); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}
	
	if _, err := s.writer.Write(data); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}
	
	return nil
}

func (s *Server) handleInitialize(msg LSPMessage) error {
	var params InitializeParams
	if err := s.unmarshalParams(msg.Params, &params); err != nil {
		return err
	}
	
	// Extract workspace root
	if params.RootURI != nil && *params.RootURI != "" {
		s.workspaceRoot = s.uriToFilePath(*params.RootURI)
	} else if params.RootPath != nil && *params.RootPath != "" {
		s.workspaceRoot = *params.RootPath
	} else if len(params.WorkspaceFolders) > 0 {
		s.workspaceRoot = s.uriToFilePath(params.WorkspaceFolders[0].URI)
	} else {
		s.workspaceRoot = "."
	}
	
	log.Printf("[view.tree] Workspace root set to: %s", s.workspaceRoot)
	
	// Check client capabilities
	if params.Capabilities.Workspace != nil {
		s.hasConfigurationCapability = params.Capabilities.Workspace.Configuration
		s.hasWorkspaceFolderCapability = params.Capabilities.Workspace.WorkspaceFolders
	}
	
	result := InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync: TextDocumentSyncKindIncremental,
			CompletionProvider: &CompletionOptions{
				ResolveProvider:   true,
				TriggerCharacters: []string{"$", "_", " ", "\t"},
			},
			DefinitionProvider: true,
			HoverProvider:      true,
		},
		ServerInfo: &ServerInfo{
			Name:    "view.tree LSP Server",
			Version: "1.0.0",
		},
	}
	
	if s.hasWorkspaceFolderCapability {
		if result.Capabilities.Workspace == nil {
			result.Capabilities.Workspace = &WorkspaceServerCapabilities{}
		}
		result.Capabilities.Workspace.WorkspaceFolders = &WorkspaceFoldersServerCapabilities{
			Supported: true,
		}
	}
	
	return s.sendResponse(msg.ID, result)
}

func (s *Server) handleInitialized(msg LSPMessage) error {
	log.Println("[view.tree] Client initialized")
	
	// Initialize providers with error recovery
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[view.tree] Panic in provider initialization: %v", r)
			}
		}()
		
		if err := s.initializeProviders(); err != nil {
			log.Printf("[view.tree] Failed to initialize providers: %v", err)
		}
	}()
	
	return nil
}

func (s *Server) initializeProviders() error {
	// Add panic recovery to prevent server crashes
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[view.tree] Panic during initialization: %v", r)
		}
	}()
	
	// Use workspace root from initialization
	workspaceRoot := s.workspaceRoot
	if workspaceRoot == "" {
		workspaceRoot = "."
	}
	
	log.Printf("[view.tree] Initializing with workspace: %s", workspaceRoot)
	
	// Initialize project scanner with error handling
	s.projectScanner = NewProjectScanner(workspaceRoot)
	if s.projectScanner == nil {
		log.Printf("[view.tree] Warning: Failed to create project scanner")
		return nil // Don't fail completely, just continue without scanning
	}
	
	// Initialize providers
	s.definitionProvider = NewDefinitionProvider(s.projectScanner)
	s.completionProvider = NewCompletionProvider(s.projectScanner)
	s.hoverProvider = NewHoverProvider(s.projectScanner)
	s.diagnosticProvider = NewDiagnosticProvider(s.projectScanner)
	
	// Start initial project scan with better error handling
	log.Println("[view.tree] Starting project scan...")
	if err := s.projectScanner.ScanProject(); err != nil {
		log.Printf("[view.tree] Project scan failed (continuing anyway): %v", err)
		// Don't return error - LSP should work even without successful project scan
	} else {
		log.Println("[view.tree] Project scan completed successfully")
	}
	
	log.Println("[view.tree] LSP server initialized successfully")
	return nil
}

func (s *Server) handleDidOpen(msg LSPMessage) error {
	var params DidOpenTextDocumentParams
	if err := s.unmarshalParams(msg.Params, &params); err != nil {
		return err
	}
	
	doc := &TextDocument{
		URI:        params.TextDocument.URI,
		LanguageID: params.TextDocument.LanguageID,
		Version:    params.TextDocument.Version,
		Text:       params.TextDocument.Text,
	}
	
	s.documents.Store(params.TextDocument.URI, doc)
	
	// Update project data incrementally
	if s.projectScanner != nil {
		uri := params.TextDocument.URI
		if strings.HasSuffix(uri, ".view.tree") || strings.HasSuffix(uri, ".ts") {
			filePath := s.uriToFilePath(uri)
			s.projectScanner.UpdateSingleFile(filePath, doc.Text)
		}
	}
	
	// Validate document
	s.validateTextDocument(doc)
	
	return nil
}

func (s *Server) handleDidChange(msg LSPMessage) error {
	var params DidChangeTextDocumentParams
	if err := s.unmarshalParams(msg.Params, &params); err != nil {
		return err
	}
	
	docInterface, ok := s.documents.Load(params.TextDocument.URI)
	if !ok {
		return fmt.Errorf("document not found: %s", params.TextDocument.URI)
	}
	
	doc := docInterface.(*TextDocument)
	doc.Version = params.TextDocument.Version
	
	// Apply changes
	for _, change := range params.ContentChanges {
		if change.Range == nil {
			// Full document update
			doc.Text = change.Text
		} else {
			// Incremental update
			doc.Text = s.applyTextChange(doc.Text, *change.Range, change.Text)
		}
	}
	
	s.documents.Store(params.TextDocument.URI, doc)
	
	// Update project data incrementally
	if s.projectScanner != nil {
		uri := params.TextDocument.URI
		if strings.HasSuffix(uri, ".view.tree") || strings.HasSuffix(uri, ".ts") {
			filePath := s.uriToFilePath(uri)
			s.projectScanner.UpdateSingleFile(filePath, doc.Text)
		}
	}
	
	// Validate document
	s.validateTextDocument(doc)
	
	return nil
}

func (s *Server) handleDidClose(msg LSPMessage) error {
	var params DidCloseTextDocumentParams
	if err := s.unmarshalParams(msg.Params, &params); err != nil {
		return err
	}
	
	s.documents.Delete(params.TextDocument.URI)
	return nil
}

func (s *Server) handleCompletion(msg LSPMessage) error {
	var params CompletionParams
	if err := s.unmarshalParams(msg.Params, &params); err != nil {
		return err
	}
	
	var items []CompletionItem
	
	if s.completionProvider != nil {
		docInterface, ok := s.documents.Load(params.TextDocument.URI)
		if ok {
			doc := docInterface.(*TextDocument)
			var err error
			items, err = s.completionProvider.ProvideCompletionItems(doc, params.Position)
			if err != nil {
				log.Printf("[view.tree] Error providing completion: %v", err)
			}
		}
	}
	
	return s.sendResponse(msg.ID, items)
}

func (s *Server) handleDefinition(msg LSPMessage) error {
	var params DefinitionParams
	if err := s.unmarshalParams(msg.Params, &params); err != nil {
		return err
	}
	
	var locations []Location
	
	if s.definitionProvider != nil {
		docInterface, ok := s.documents.Load(params.TextDocument.URI)
		if ok {
			doc := docInterface.(*TextDocument)
			var err error
			locations, err = s.definitionProvider.ProvideDefinition(doc, params.Position)
			if err != nil {
				log.Printf("[view.tree] Error providing definition: %v", err)
			}
		}
	}
	
	return s.sendResponse(msg.ID, locations)
}

func (s *Server) handleHover(msg LSPMessage) error {
	var params HoverParams
	if err := s.unmarshalParams(msg.Params, &params); err != nil {
		return err
	}
	
	var hover *Hover
	
	if s.hoverProvider != nil {
		docInterface, ok := s.documents.Load(params.TextDocument.URI)
		if ok {
			doc := docInterface.(*TextDocument)
			var err error
			hover, err = s.hoverProvider.ProvideHover(doc, params.Position)
			if err != nil {
				log.Printf("[view.tree] Error providing hover: %v", err)
			}
		}
	}
	
	return s.sendResponse(msg.ID, hover)
}

func (s *Server) handleShutdown(msg LSPMessage) error {
	log.Println("[view.tree] Shutting down...")
	return s.sendResponse(msg.ID, nil)
}

func (s *Server) validateTextDocument(doc *TextDocument) {
	if s.diagnosticProvider == nil || !strings.HasSuffix(doc.URI, ".view.tree") {
		return
	}
	
	diagnostics, err := s.diagnosticProvider.ProvideDiagnostics(doc)
	if err != nil {
		log.Printf("[view.tree] Error validating document: %v", err)
		return
	}
	
	params := PublishDiagnosticsParams{
		URI:         doc.URI,
		Version:     &doc.Version,
		Diagnostics: diagnostics,
	}
	
	if err := s.sendNotification("textDocument/publishDiagnostics", params); err != nil {
		log.Printf("[view.tree] Error sending diagnostics: %v", err)
	}
}

func (s *Server) applyTextChange(text string, changeRange Range, newText string) string {
	lines := strings.Split(text, "\n")
	
	// Convert positions to offsets
	startOffset := s.positionToOffset(lines, changeRange.Start)
	endOffset := s.positionToOffset(lines, changeRange.End)
	
	// Apply change
	before := text[:startOffset]
	after := text[endOffset:]
	
	return before + newText + after
}

func (s *Server) positionToOffset(lines []string, pos Position) int {
	offset := 0
	for i := 0; i < pos.Line && i < len(lines); i++ {
		offset += len(lines[i]) + 1 // +1 for newline
	}
	if pos.Line < len(lines) {
		offset += pos.Character
	}
	return offset
}

func (s *Server) uriToFilePath(uri string) string {
	// Simple URI to file path conversion
	// In a real implementation, this would be more robust
	if strings.HasPrefix(uri, "file://") {
		return strings.TrimPrefix(uri, "file://")
	}
	return uri
}

func (s *Server) unmarshalParams(params interface{}, target interface{}) error {
	data, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal params: %w", err)
	}
	
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal params: %w", err)
	}
	
	return nil
}