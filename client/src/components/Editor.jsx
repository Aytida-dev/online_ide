import React, { useState, useEffect, useRef } from 'react';
import Editor from '@monaco-editor/react';
import { Terminal } from 'xterm';
import 'xterm/css/xterm.css';

const CodeEditor = () => {
    const [code, setCode] = useState('');
    const termRef = useRef(null);
    const ws = useRef(null);

    useEffect(() => {
        // Initialize terminal
        termRef.current = new Terminal();
        termRef.current.open(document.getElementById('terminal'));

        // Connect to WebSocket
        ws.current = new WebSocket('ws://localhost:8080/ws');

        ws.current.onmessage = (event) => {
            termRef.current.write(event.data);
        };

        return () => ws.current.close();
    }, []);

    const runCode = () => {
        if (ws.current.readyState === WebSocket.OPEN) {
            ws.current.send(JSON.stringify({ type: 'code', content: code }));
        }
    };

    return (
        <div>
            <Editor
                height="60vh"
                defaultLanguage="javascript"
                defaultValue="// Write your code here"
                onChange={(value) => setCode(value)}
            />
            <button onClick={runCode}>Run</button>
            <div id="terminal" style={{ height: '30vh' }} />
        </div>
    );
};

export default CodeEditor;