import React, { useState, useEffect, useRef } from 'react';
import './App.css';

// Message types
const MESSAGE_TYPE = {
  CODE: 'CODE:',
  EXEC_TERMINATED: 'EXEC_TERMINATED',
  EXEC_TIMEOUT: 'EXEC_TIMEOUT',
  CONTAINER_ID: 'container_id:',
  ERROR: 'error:'
};

function App() {
  const [messages, setMessages] = useState([]);
  const [jsCode, setJsCode] = useState('');
  const [inputValue, setInputValue] = useState('');
  const [containerId, setContainerId] = useState(null);
  const [isConnected, setIsConnected] = useState(false);
  const [isRunning, setIsRunning] = useState(false);
  const [executionCount, setExecutionCount] = useState(0);
  const [executionTime, setExecutionTime] = useState(0);
  const ws = useRef(null);
  const timerRef = useRef(null);

  const defaultJsCode = `const readline = require('readline');
const rl = readline.createInterface({
  input: process.stdin,
  output: process.stdout
});

rl.question('What is your name? ', (name) => {
  console.log(\`Hello, \${name}!\`);
  rl.question('How old are you? ', (age) => {
    console.log(\`\${name} is \${age} years old.\`);
    rl.close();
  });
});`;

  // Initialize with default code
  useEffect(() => {
    setJsCode(defaultJsCode);
    return cleanup;
  }, []);

  // Cleanup function
  const cleanup = () => {
    if (ws.current) {
      ws.current.close();
    }
    if (timerRef.current) {
      clearInterval(timerRef.current);
    }
  };

  // Connect to WebSocket
  const connectWebSocket = () => {
    cleanup();
    setMessages([]);
    setExecutionCount(0);
    setExecutionTime(0);

    ws.current = new WebSocket('ws://localhost:3000/ws');

    ws.current.onopen = () => {
      setIsConnected(true);
      addMessage('Connected to server. Ready to execute JavaScript.');
    };

    ws.current.onmessage = (event) => {
      const data = event.data;

      if (data.startsWith(MESSAGE_TYPE.CONTAINER_ID)) {
        const id = data.replace(MESSAGE_TYPE.CONTAINER_ID, '');
        setContainerId(id);
        addMessage(`Container started: ${id}`);
      }
      else if (data.startsWith(MESSAGE_TYPE.ERROR)) {
        addMessage(data, false);
        stopExecution();
      }
      else if (data === MESSAGE_TYPE.EXEC_TERMINATED) {
        addMessage('Program execution completed');
        stopExecution();
      }
      else if (data === MESSAGE_TYPE.EXEC_TIMEOUT) {
        addMessage('Program execution timed out', false);
        stopExecution();
      }
      else {
        addMessage(data);
      }
    };

    ws.current.onerror = (error) => {
      addMessage(`Error: ${error.message}`, false);
      stopExecution();
    };

    ws.current.onclose = () => {
      stopExecution();
      addMessage('Connection closed');
    };
  };

  // Execute code with CODE: prefix
  const executeCode = () => {
    if (!ws.current || ws.current.readyState !== WebSocket.OPEN) {
      addMessage('Not connected to server', false);
      return;
    }

    if (!jsCode.trim()) {
      addMessage('Please enter some JavaScript code', false);
      return;
    }

    startExecution();
    // Send code with CODE: prefix
    ws.current.send(`${MESSAGE_TYPE.CODE}${jsCode}`);
  };

  // Start execution timer and state
  const startExecution = () => {
    setIsRunning(true);
    setExecutionCount(prev => prev + 1);
    setMessages(prev => [...prev, { text: 'Executing JavaScript...', isOutput: true }]);
    setExecutionTime(0);
    setInputValue('');

    timerRef.current = setInterval(() => {
      setExecutionTime(prev => prev + 1);
    }, 1000);
  };

  // Stop execution and cleanup
  const stopExecution = () => {
    setIsRunning(false);
    if (timerRef.current) {
      clearInterval(timerRef.current);
      timerRef.current = null;
    }
  };

  // Send input to running program
  const sendInput = () => {
    if (!ws.current || ws.current.readyState !== WebSocket.OPEN) {
      addMessage('Not connected to server', false);
      return;
    }

    if (!inputValue.trim()) return;

    // Send raw input (no prefix)
    ws.current.send(inputValue + '\n');
    setInputValue('');
  };

  // Add message to terminal
  const addMessage = (text, isOutput = true) => {
    setMessages(prev => [...prev, { text, isOutput }]);
  };

  // Handle Enter key for input
  const handleKeyPress = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendInput();
    }
  };

  // Format time display
  const formatTime = (seconds) => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
  };

  return (
    <div className="App">
      <header className="App-header">
        <h1>JavaScript Docker Runner</h1>

        <div className="controls">
          {!isConnected ? (
            <button onClick={connectWebSocket}>Connect</button>
          ) : (
            <button onClick={cleanup}>Disconnect</button>
          )}
        </div>

        <div className="code-editor">
          <textarea
            value={jsCode}
            onChange={(e) => setJsCode(e.target.value)}
            placeholder="Enter JavaScript code"
            disabled={isRunning}
            rows={10}
          />
          <div className="editor-buttons">
            <button
              onClick={executeCode}
              disabled={!isConnected || isRunning || !jsCode.trim()}
            >
              {isRunning ? 'Running...' : 'Execute'}
            </button>
            <button
              onClick={() => setJsCode(defaultJsCode)}
              disabled={isRunning}
            >
              Reset Code
            </button>
          </div>
        </div>

        <div className="terminal-container">
          <div className="terminal-header">
            <span>Terminal</span>
            {isRunning && (
              <div className="execution-info">
                <span>Executing #{executionCount}</span>
                <span>Time: {formatTime(executionTime)}</span>
                <button onClick={stopExecution} className="stop-button">
                  Stop
                </button>
              </div>
            )}
          </div>
          <div className="terminal">
            {messages.map((msg, i) => (
              <div key={i} className={msg.isOutput ? 'output' : 'error'}>
                {msg.text}
              </div>
            ))}
          </div>
        </div>

        {isRunning && (
          <div className="input-area">
            <input
              type="text"
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              onKeyDown={handleKeyPress}
              placeholder="Enter input for the program..."
              autoFocus
            />
            <button onClick={sendInput} disabled={!inputValue.trim()}>
              Send
            </button>
          </div>
        )}
      </header>
    </div>
  );
}

export default App;