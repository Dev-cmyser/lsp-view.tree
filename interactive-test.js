#!/usr/bin/env node

const { spawn } = require('child_process');
const path = require('path');
const readline = require('readline');

class InteractiveLSPTester {
    constructor() {
        this.server = null;
        this.messageId = 1;
        this.pendingRequests = new Map();
        this.isInitialized = false;
        this.buffer = '';
        
        this.rl = readline.createInterface({
            input: process.stdin,
            output: process.stdout
        });
    }

    async start() {
        console.log('🧪 Interactive LSP Server Tester\n');
        
        try {
            await this.startServer();
            await this.initializeServer();
            await this.showMenu();
        } catch (error) {
            console.error('❌ Failed to start:', error.message);
            this.cleanup();
        }
    }

    async startServer() {
        console.log('🚀 Starting LSP server...');
        
        const serverPath = path.join(__dirname, 'lib', 'server.js');
        this.server = spawn('node', [serverPath, '--stdio'], {
            stdio: ['pipe', 'pipe', 'pipe']
        });

        this.server.stderr.on('data', (data) => {
            const logText = data.toString().trim();
            if (logText && logText.includes('[view.tree]')) {
                console.log(`📝 ${logText}`);
            }
        });

        this.server.stdout.on('data', (data) => {
            this.buffer += data.toString();
            this.processMessages();
        });

        this.server.on('error', (error) => {
            console.error('❌ Server error:', error);
        });

        this.server.on('exit', (code) => {
            console.log(`🛑 Server exited with code ${code}`);
        });

        // Wait for server to start
        await new Promise(resolve => setTimeout(resolve, 500));
        console.log('✅ Server started');
    }

    processMessages() {
        while (true) {
            const headerEnd = this.buffer.indexOf('\r\n\r\n');
            if (headerEnd === -1) break;
            
            const header = this.buffer.substring(0, headerEnd);
            const contentLengthMatch = header.match(/Content-Length: (\d+)/);
            
            if (!contentLengthMatch) break;
            
            const contentLength = parseInt(contentLengthMatch[1]);
            const messageStart = headerEnd + 4;
            
            if (this.buffer.length < messageStart + contentLength) break;
            
            const messageContent = this.buffer.substring(messageStart, messageStart + contentLength);
            this.buffer = this.buffer.substring(messageStart + contentLength);
            
            try {
                const message = JSON.parse(messageContent);
                this.handleMessage(message);
            } catch (error) {
                console.error('❌ Parse error:', error.message);
            }
        }
    }

    handleMessage(message) {
        if (message.id && this.pendingRequests.has(message.id)) {
            const { resolve, method } = this.pendingRequests.get(message.id);
            this.pendingRequests.delete(message.id);
            
            if (message.error) {
                console.log(`❌ ${method} failed:`, message.error.message);
                resolve(null);
            } else {
                resolve(message.result);
            }
        } else if (message.method) {
            console.log(`📨 Notification: ${message.method}`);
        }
    }

    async sendRequest(method, params) {
        const id = this.messageId++;
        const message = {
            jsonrpc: '2.0',
            id,
            method,
            params
        };

        const content = JSON.stringify(message);
        const header = `Content-Length: ${content.length}\r\n\r\n`;
        
        this.server.stdin.write(header + content);
        
        return new Promise(resolve => {
            this.pendingRequests.set(id, { resolve, method });
            setTimeout(() => {
                if (this.pendingRequests.has(id)) {
                    this.pendingRequests.delete(id);
                    console.log(`⏰ ${method} timed out`);
                    resolve(null);
                }
            }, 5000);
        });
    }

    sendNotification(method, params) {
        const message = {
            jsonrpc: '2.0',
            method,
            params
        };

        const content = JSON.stringify(message);
        const header = `Content-Length: ${content.length}\r\n\r\n`;
        
        this.server.stdin.write(header + content);
    }

    async initializeServer() {
        console.log('🔧 Initializing server...');
        
        const result = await this.sendRequest('initialize', {
            processId: process.pid,
            clientInfo: {
                name: 'Interactive LSP Tester',
                version: '1.0.0'
            },
            capabilities: {
                textDocument: {
                    completion: { completionItem: { snippetSupport: true } },
                    hover: { contentFormat: ['markdown', 'plaintext'] },
                    definition: { linkSupport: true }
                }
            },
            workspaceFolders: [{
                uri: 'file://' + __dirname,
                name: 'test-workspace'
            }]
        });

        if (result) {
            console.log('✅ Server initialized with capabilities:', Object.keys(result.capabilities));
            this.sendNotification('initialized', {});
            this.isInitialized = true;
        } else {
            throw new Error('Failed to initialize server');
        }
    }

    async showMenu() {
        while (true) {
            console.log('\n📋 LSP Test Menu:');
            console.log('1. Test completion');
            console.log('2. Test hover');
            console.log('3. Test go-to-definition');
            console.log('4. Open document');
            console.log('5. Create sample document');
            console.log('6. Show server status');
            console.log('7. Exit');
            
            const choice = await this.askQuestion('\nChoose option (1-7): ');
            
            try {
                switch (choice.trim()) {
                    case '1':
                        await this.testCompletion();
                        break;
                    case '2':
                        await this.testHover();
                        break;
                    case '3':
                        await this.testDefinition();
                        break;
                    case '4':
                        await this.openDocument();
                        break;
                    case '5':
                        await this.createSampleDocument();
                        break;
                    case '6':
                        await this.showStatus();
                        break;
                    case '7':
                        this.cleanup();
                        return;
                    default:
                        console.log('❌ Invalid choice');
                }
            } catch (error) {
                console.log('❌ Error:', error.message);
            }
        }
    }

    async testCompletion() {
        console.log('\n🔍 Testing Completion');
        
        const uri = await this.askQuestion('Document URI (or press Enter for default): ') || 
                    'file://' + path.join(__dirname, 'test.view.tree');
        const line = parseInt(await this.askQuestion('Line number (0-based): ')) || 0;
        const character = parseInt(await this.askQuestion('Character position: ')) || 1;
        
        console.log(`\n📤 Requesting completion at ${line}:${character} in ${uri}`);
        
        const result = await this.sendRequest('textDocument/completion', {
            textDocument: { uri },
            position: { line, character }
        });
        
        if (result && result.length > 0) {
            console.log(`✅ Found ${result.length} completion items:`);
            result.slice(0, 10).forEach((item, i) => {
                console.log(`   ${i + 1}. "${item.label}" (${this.getCompletionKindName(item.kind)})`);
                if (item.detail) console.log(`      Detail: ${item.detail}`);
            });
            if (result.length > 10) {
                console.log(`   ... and ${result.length - 10} more`);
            }
        } else {
            console.log('ℹ️  No completion items found');
        }
    }

    async testHover() {
        console.log('\n💡 Testing Hover');
        
        const uri = await this.askQuestion('Document URI (or press Enter for default): ') || 
                    'file://' + path.join(__dirname, 'test.view.tree');
        const line = parseInt(await this.askQuestion('Line number (0-based): ')) || 0;
        const character = parseInt(await this.askQuestion('Character position: ')) || 5;
        
        console.log(`\n📤 Requesting hover at ${line}:${character} in ${uri}`);
        
        const result = await this.sendRequest('textDocument/hover', {
            textDocument: { uri },
            position: { line, character }
        });
        
        if (result && result.contents) {
            console.log('✅ Hover information:');
            if (result.contents.value) {
                console.log(result.contents.value);
            } else if (typeof result.contents === 'string') {
                console.log(result.contents);
            } else {
                console.log(JSON.stringify(result.contents, null, 2));
            }
        } else {
            console.log('ℹ️  No hover information');
        }
    }

    async testDefinition() {
        console.log('\n🎯 Testing Go-to-Definition');
        
        const uri = await this.askQuestion('Document URI (or press Enter for default): ') || 
                    'file://' + path.join(__dirname, 'test.view.tree');
        const line = parseInt(await this.askQuestion('Line number (0-based): ')) || 0;
        const character = parseInt(await this.askQuestion('Character position: ')) || 5;
        
        console.log(`\n📤 Requesting definition at ${line}:${character} in ${uri}`);
        
        const result = await this.sendRequest('textDocument/definition', {
            textDocument: { uri },
            position: { line, character }
        });
        
        if (result && result.length > 0) {
            console.log(`✅ Found ${result.length} definition(s):`);
            result.forEach((def, i) => {
                console.log(`   ${i + 1}. ${def.uri}`);
                console.log(`      Range: ${def.range.start.line}:${def.range.start.character} - ${def.range.end.line}:${def.range.end.character}`);
            });
        } else {
            console.log('ℹ️  No definitions found');
        }
    }

    async openDocument() {
        console.log('\n📄 Opening Document');
        
        const uri = await this.askQuestion('Document URI: ');
        if (!uri) {
            console.log('❌ URI is required');
            return;
        }
        
        const content = await this.askQuestion('Content (or press Enter for sample): ') ||
            `$sample_component $mol_view\n\ttitle \\Sample\n\tbody /\n\t\t<= content\n\tcontent $mol_text\n\t\ttext \\Hello World`;
        
        console.log(`\n📤 Opening document: ${uri}`);
        
        this.sendNotification('textDocument/didOpen', {
            textDocument: {
                uri,
                languageId: 'tree',
                version: 1,
                text: content
            }
        });
        
        console.log('✅ Document opened');
        
        // Wait a bit for processing
        await new Promise(resolve => setTimeout(resolve, 1000));
        console.log('ℹ️  Document should now be available for testing');
    }

    async createSampleDocument() {
        console.log('\n📝 Creating Sample Document');
        
        const samples = [
            {
                name: 'Basic component',
                content: `$my_app $mol_page\n\ttitle \\My Application\n\tbody /\n\t\t<= content\n\tcontent $mol_view\n\t\tsub /\n\t\t\t<= welcome_text\n\twelcome_text $mol_text\n\t\ttext \\Hello World`
            },
            {
                name: 'Complex component',
                content: `$complex_app $mol_page\n\tdom_name \\main\n\tattr *\n\t\tclass \\app-main\n\t\tid \\main-app\n\tbody /\n\t\t<= header\n\t\t<= content\n\t\t<= footer\n\theader $mol_view\n\t\tdom_name \\header\n\t\tsub /\n\t\t\t<= title_text\n\ttitle_text $mol_text\n\t\ttext <= app_title\n\tcontent $mol_list\n\t\trows /\n\t\t\t<= items\n\tfooter $mol_view\n\t\tvisible <= show_footer`
            },
            {
                name: 'Form component',
                content: `$contact_form $mol_form\n\tfields /\n\t\t<= name_field\n\t\t<= email_field\n\t\t<= submit_btn\n\tname_field $mol_string\n\t\thint \\Your name\n\t\tvalue <=> contact_name\n\temail_field $mol_string\n\t\thint \\Email address\n\t\tvalue <=> contact_email\n\tsubmit_btn $mol_button\n\t\ttitle \\Send Message\n\t\tenabled <= form_valid\n\t\tclick <= handle_submit`
            }
        ];
        
        console.log('\nAvailable samples:');
        samples.forEach((sample, i) => {
            console.log(`${i + 1}. ${sample.name}`);
        });
        
        const choice = await this.askQuestion('Choose sample (1-3): ');
        const sampleIndex = parseInt(choice) - 1;
        
        if (sampleIndex < 0 || sampleIndex >= samples.length) {
            console.log('❌ Invalid choice');
            return;
        }
        
        const sample = samples[sampleIndex];
        const uri = 'file://' + path.join(__dirname, 'sample.view.tree');
        
        console.log(`\n📤 Creating ${sample.name} at ${uri}`);
        
        this.sendNotification('textDocument/didOpen', {
            textDocument: {
                uri,
                languageId: 'tree',
                version: 1,
                text: sample.content
            }
        });
        
        console.log('✅ Sample document created');
        console.log('\nSample content:');
        console.log('─'.repeat(40));
        console.log(sample.content.replace(/\\t/g, '  '));
        console.log('─'.repeat(40));
        
        await new Promise(resolve => setTimeout(resolve, 1000));
        console.log('ℹ️  You can now test completion, hover, etc. on this document');
    }

    async showStatus() {
        console.log('\n📊 Server Status');
        console.log(`Initialized: ${this.isInitialized ? '✅' : '❌'}`);
        console.log(`Server process: ${this.server ? '✅ Running' : '❌ Not running'}`);
        console.log(`Pending requests: ${this.pendingRequests.size}`);
        
        if (this.pendingRequests.size > 0) {
            console.log('Pending:');
            for (const [id, req] of this.pendingRequests) {
                console.log(`  - ${id}: ${req.method}`);
            }
        }
    }

    getCompletionKindName(kind) {
        const kinds = {
            1: 'Text', 2: 'Method', 3: 'Function', 4: 'Constructor', 5: 'Field',
            6: 'Variable', 7: 'Class', 8: 'Interface', 9: 'Module', 10: 'Property',
            11: 'Unit', 12: 'Value', 13: 'Enum', 14: 'Keyword', 15: 'Snippet',
            16: 'Color', 17: 'File', 18: 'Reference', 19: 'Folder', 20: 'EnumMember',
            21: 'Constant', 22: 'Struct', 23: 'Event', 24: 'Operator', 25: 'TypeParameter'
        };
        return kinds[kind] || `Unknown(${kind})`;
    }

    askQuestion(question) {
        return new Promise(resolve => {
            this.rl.question(question, resolve);
        });
    }

    cleanup() {
        console.log('\n👋 Cleaning up...');
        if (this.server) {
            this.server.kill();
        }
        this.rl.close();
        process.exit(0);
    }
}

// Handle interrupts
process.on('SIGINT', () => {
    console.log('\n🛑 Interrupted by user');
    process.exit(0);
});

// Start interactive tester
if (require.main === module) {
    const tester = new InteractiveLSPTester();
    tester.start().catch(error => {
        console.error('❌ Fatal error:', error);
        process.exit(1);
    });
}

module.exports = InteractiveLSPTester;