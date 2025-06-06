#!/usr/bin/env node

const { spawn } = require('child_process');
const path = require('path');

console.log('🧪 Simple LSP Server Test\n');

// Start the LSP server
console.log('🚀 Starting server...');
const serverPath = path.join(__dirname, 'lib', 'server.js');
const server = spawn('node', [serverPath, '--stdio'], {
    stdio: ['pipe', 'pipe', 'pipe']
});

let messageId = 1;
let buffer = '';

// Handle server output
server.stdout.on('data', (data) => {
    buffer += data.toString();
    
    // Process complete messages
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
            console.log('📨 Response:', message.id ? `ID ${message.id}` : 'notification');
            if (message.result && message.result.capabilities) {
                console.log('✅ Server initialized with capabilities:', Object.keys(message.result.capabilities));
            }
            // Show any server logs that come through stdout
            if (message.method === 'window/logMessage' || message.method === 'textDocument/publishDiagnostics') {
                console.log('📋 Server message:', JSON.stringify(message, null, 2));
            }
        } catch (error) {
            console.error('❌ Parse error:', error.message);
            console.error('Raw content:', messageContent);
        }
    }
});

server.stderr.on('data', (data) => {
    const logText = data.toString().trim();
    if (logText) {
        console.log('📝 Server log:', logText);
    }
});

// Also capture stdout for any debug messages
server.stdout.setEncoding('utf8');

function sendMessage(method, params) {
    const message = {
        jsonrpc: '2.0',
        id: messageId++,
        method,
        params
    };

    const content = JSON.stringify(message);
    const header = `Content-Length: ${content.length}\r\n\r\n`;
    
    console.log(`📤 Sending: ${method}`);
    server.stdin.write(header + content);
}

// Test sequence
setTimeout(() => {
    console.log('\n🔧 Testing initialization...');
    sendMessage('initialize', {
        processId: process.pid,
        capabilities: {
            textDocument: {
                completion: {},
                hover: {},
                definition: {}
            }
        },
        workspaceFolders: [{
            uri: 'file://' + __dirname,
            name: 'test'
        }]
    });
}, 100);

// Send initialized notification first
setTimeout(() => {
    console.log('\n📤 Sending initialized notification...');
    const initMessage = {
        jsonrpc: '2.0',
        method: 'initialized',
        params: {}
    };
    const content = JSON.stringify(initMessage);
    const header = `Content-Length: ${content.length}\r\n\r\n`;
    server.stdin.write(header + content);
}, 1500);

// Test completion after initialization
setTimeout(() => {
    console.log('\n🔍 Testing completion...');
    sendMessage('textDocument/completion', {
        textDocument: { uri: 'file://test.view.tree' },
        position: { line: 0, character: 1 }
    });
}, 2500);

// Cleanup
setTimeout(() => {
    console.log('\n✅ Basic test completed!');
    console.log('💡 For full testing, run: node test-client.js');
    server.kill();
    process.exit(0);
}, 4000);

process.on('SIGINT', () => {
    console.log('\n👋 Interrupted');
    server.kill();
    process.exit(0);
});