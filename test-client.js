const { spawn } = require('child_process');
const path = require('path');

class LSPTestClient {
    constructor() {
        this.messageId = 1;
        this.serverProcess = null;
        this.responses = new Map();
    }

    async start() {
        console.log('🚀 Starting LSP server...');
        
        const serverPath = path.join(__dirname, 'lib', 'server.js');
        this.serverProcess = spawn('node', [serverPath, '--stdio'], {
            stdio: ['pipe', 'pipe', 'pipe']
        });

        this.serverProcess.stderr.on('data', (data) => {
            console.log('📝 Server log:', data.toString());
        });

        let buffer = '';
        this.serverProcess.stdout.on('data', (data) => {
            buffer += data.toString();
            
            while (true) {
                const headerEnd = buffer.indexOf('\r\n\r\n');
                if (headerEnd === -1) break;
                
                const header = buffer.substring(0, headerEnd);
                const contentLengthMatch = header.match(/Content-Length: (\d+)/);
                
                if (!contentLengthMatch) break;
                
                const contentLength = parseInt(contentLengthMatch[1]);
                const messageStart = headerEnd + 4;
                
                if (buffer.length < messageStart + contentLength) break;
                
                const messageContent = buffer.substring(messageStart, messageStart + contentLength);
                buffer = buffer.substring(messageStart + contentLength);
                
                try {
                    const message = JSON.parse(messageContent);
                    this.handleResponse(message);
                } catch (error) {
                    console.error('❌ Failed to parse message:', error);
                }
            }
        });

        // Wait a bit for server to start
        await this.sleep(500);
    }

    sendMessage(method, params, id = null) {
        if (id === null) {
            id = this.messageId++;
        }

        const message = {
            jsonrpc: '2.0',
            id,
            method,
            params
        };

        const content = JSON.stringify(message);
        const header = `Content-Length: ${content.length}\r\n\r\n`;
        
        this.serverProcess.stdin.write(header + content);
        
        return new Promise((resolve, reject) => {
            this.responses.set(id, { resolve, reject });
            
            // Timeout after 5 seconds
            setTimeout(() => {
                if (this.responses.has(id)) {
                    this.responses.delete(id);
                    reject(new Error('Request timeout'));
                }
            }, 5000);
        });
    }

    handleResponse(message) {
        if (message.id && this.responses.has(message.id)) {
            const { resolve, reject } = this.responses.get(message.id);
            this.responses.delete(message.id);
            
            if (message.error) {
                reject(new Error(message.error.message));
            } else {
                resolve(message.result);
            }
        } else {
            console.log('📨 Notification:', JSON.stringify(message, null, 2));
        }
    }

    async initialize() {
        console.log('🔧 Initializing LSP server...');
        
        const result = await this.sendMessage('initialize', {
            processId: process.pid,
            clientInfo: {
                name: 'LSP Test Client',
                version: '1.0.0'
            },
            capabilities: {
                textDocument: {
                    completion: {
                        completionItem: {
                            snippetSupport: true
                        }
                    },
                    hover: {
                        contentFormat: ['markdown', 'plaintext']
                    },
                    definition: {
                        linkSupport: true
                    }
                }
            },
            workspaceFolders: [{
                uri: 'file://' + __dirname,
                name: 'test-workspace'
            }]
        });

        console.log('✅ Server capabilities:', Object.keys(result.capabilities));
        
        // Send initialized notification
        this.sendNotification('initialized', {});
        
        return result;
    }

    sendNotification(method, params) {
        const message = {
            jsonrpc: '2.0',
            method,
            params
        };

        const content = JSON.stringify(message);
        const header = `Content-Length: ${content.length}\r\n\r\n`;
        
        this.serverProcess.stdin.write(header + content);
    }

    async openDocument(uri, content) {
        console.log(`📄 Opening document: ${uri}`);
        
        this.sendNotification('textDocument/didOpen', {
            textDocument: {
                uri,
                languageId: 'tree',
                version: 1,
                text: content
            }
        });
    }

    async testCompletion(uri, line, character) {
        console.log(`🔍 Testing completion at ${line}:${character}...`);
        
        try {
            const result = await this.sendMessage('textDocument/completion', {
                textDocument: { uri },
                position: { line, character }
            });

            console.log(`✅ Completion items: ${result.length}`);
            if (result.length > 0) {
                console.log(`   First item: "${result[0].label}" (${result[0].kind})`);
                if (result.length > 1) {
                    console.log(`   Second item: "${result[1].label}" (${result[1].kind})`);
                }
            }
            return result;
        } catch (error) {
            console.log(`❌ Completion failed: ${error.message}`);
            return [];
        }
    }

    async testHover(uri, line, character) {
        console.log(`💡 Testing hover at ${line}:${character}...`);
        
        try {
            const result = await this.sendMessage('textDocument/hover', {
                textDocument: { uri },
                position: { line, character }
            });

            if (result) {
                console.log(`✅ Hover content available`);
                if (result.contents.value) {
                    const preview = result.contents.value.substring(0, 100);
                    console.log(`   Preview: "${preview}${result.contents.value.length > 100 ? '...' : ''}"`);
                }
            } else {
                console.log(`ℹ️  No hover information`);
            }
            return result;
        } catch (error) {
            console.log(`❌ Hover failed: ${error.message}`);
            return null;
        }
    }

    async testDefinition(uri, line, character) {
        console.log(`🎯 Testing go-to-definition at ${line}:${character}...`);
        
        try {
            const result = await this.sendMessage('textDocument/definition', {
                textDocument: { uri },
                position: { line, character }
            });

            if (result && result.length > 0) {
                console.log(`✅ Found ${result.length} definition(s)`);
                console.log(`   First: ${result[0].uri}`);
            } else {
                console.log(`ℹ️  No definitions found`);
            }
            return result;
        } catch (error) {
            console.log(`❌ Definition failed: ${error.message}`);
            return [];
        }
    }

    async runTests() {
        try {
            // Initialize server
            await this.initialize();
            
            // Test document content
            const testUri = 'file://' + path.join(__dirname, 'test.view.tree');
            const testContent = `$my_test_app $mol_page
\ttitle \\Test Application
\tbody /
\t\t<= content
\t\t<= footer
\t
\tcontent $mol_view
\t\tsub /
\t\t\t<= welcome_message
\t
\twelcome_message $mol_text
\t\ttext \\Hello World
\t
\tfooter $mol_view
\t\tdom_name \\footer
\t\tattr *
\t\t\tclass \\app-footer`;

            // Open test document
            await this.openDocument(testUri, testContent);
            
            // Wait for processing
            await this.sleep(1000);
            
            console.log('\n🧪 Running LSP feature tests...\n');
            
            // Test 1: Completion at beginning of component line
            await this.testCompletion(testUri, 0, 1);
            
            // Test 2: Completion for property
            await this.testCompletion(testUri, 1, 2);
            
            // Test 3: Completion for component reference
            await this.testCompletion(testUri, 5, 3);
            
            // Test 4: Hover on component name
            await this.testHover(testUri, 0, 5);
            
            // Test 5: Hover on property
            await this.testHover(testUri, 1, 2);
            
            // Test 6: Go to definition
            await this.testDefinition(testUri, 0, 5);
            
            console.log('\n🎉 All tests completed!');
            
        } catch (error) {
            console.error('❌ Test failed:', error);
        } finally {
            this.stop();
        }
    }

    stop() {
        console.log('🛑 Stopping LSP server...');
        if (this.serverProcess) {
            this.serverProcess.kill();
        }
    }

    sleep(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    }
}

// Run tests if called directly
if (require.main === module) {
    console.log('🧪 LSP Server Test Client\n');
    
    const client = new LSPTestClient();
    
    process.on('SIGINT', () => {
        console.log('\n👋 Interrupted by user');
        client.stop();
        process.exit(0);
    });
    
    client.start().then(() => {
        return client.runTests();
    }).catch(error => {
        console.error('❌ Failed to start tests:', error);
        client.stop();
        process.exit(1);
    });
}

module.exports = LSPTestClient;