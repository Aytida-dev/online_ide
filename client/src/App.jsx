import React, { useState, useEffect, useRef } from 'react';
import './App.css';

// Message types
const MESSAGE_TYPE = {
  CODE: 'CODE:',
  EXEC_TERMINATED: 'EXEC_TERMINATED',
  EXEC_TIMEOUT: 'EXEC_TIMEOUT',
  CONTAINER_ID: 'container_id:',
  ERROR: 'error:',
  STOP: 'STOP'
};

// Language configurations
const LANGUAGES = {
  js: {
    name: 'JavaScript',
  },
  py: {
    name: 'Python',
  },
  c: {
    name: 'C',

  },
  cpp: {
    name: 'C++',

  },
  ts: {
    name: 'TypeScript',

  },
  "py-ml": {
    name: 'Python with ml',
  },

};

function App() {
  const [messages, setMessages] = useState([]);
  const [code, setCode] = useState('');
  const [inputValue, setInputValue] = useState('');
  const [containerId, setContainerId] = useState(null);
  const [isConnected, setIsConnected] = useState(false);
  const [isRunning, setIsRunning] = useState(false);
  const [executionCount, setExecutionCount] = useState(0);
  const [executionTime, setExecutionTime] = useState(0);
  const [currentLanguage, setCurrentLanguage] = useState('js');
  const ws = useRef(null);
  const timerRef = useRef(null);

  // Initialize with default code for current language



  // Cleanup function
  const cleanup = async () => {
    if (ws.current && ws.current.readyState === WebSocket.OPEN) {
      ws.current.close();
    }
    if (timerRef.current) {
      clearInterval(timerRef.current);
      timerRef.current = null;
    }

  };

  // Connect to WebSocket with language parameter
  const connectWebSocket = async (language = currentLanguage) => {
    await cleanup();
    setMessages([]);
    setExecutionCount(0);
    setExecutionTime(0);

    const wsUrl = new URL(import.meta.env.VITE_API_URL || 'ws://localhost:3000/ws');
    wsUrl.searchParams.set('language', language);

    ws.current = new WebSocket(wsUrl.toString());

    ws.current.onopen = () => {
      addMessage(`Connecting to server (${LANGUAGES[language].name}) to execute code.`);
    };

    ws.current.onmessage = (event) => {
      const data = event.data;

      if (data.startsWith(MESSAGE_TYPE.CONTAINER_ID)) {
        const id = data.replace(MESSAGE_TYPE.CONTAINER_ID, '');
        setContainerId(id);
        setIsConnected(true);
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
      setIsConnected(false);
      setContainerId(null);
      addMessage('Connection closed');
    };
  };

  // Change language and reconnect
  const changeLanguage = async (language) => {
    if (language === currentLanguage) return;

    setCurrentLanguage(language);
    await connectWebSocket(language);
  };

  // Execute code with CODE: prefix
  const executeCode = () => {
    if (!ws.current || ws.current.readyState !== WebSocket.OPEN) {
      addMessage('Not connected to server', false);
      return;
    }

    if (!code.trim()) {
      addMessage('Please enter some code', false);
      return;
    }

    startExecution();
    ws.current.send(`${MESSAGE_TYPE.CODE}${code}`);
  };

  // Send STOP command to server
  const stopExecution = () => {
    if (isRunning && ws.current && ws.current.readyState === WebSocket.OPEN) {
      ws.current.send(MESSAGE_TYPE.STOP);
    }

    setIsRunning(false);
    if (timerRef.current) {
      clearInterval(timerRef.current);
      timerRef.current = null;
    }
  };

  // Start execution timer and state
  const startExecution = () => {
    setIsRunning(true);
    setExecutionCount(prev => prev + 1);
    setMessages(prev => [...prev, { text: `Executing ${LANGUAGES[currentLanguage].name} code...`, isOutput: true }]);
    setExecutionTime(0);
    setInputValue('');

    timerRef.current = setInterval(() => {
      setExecutionTime(prev => prev + 1);
    }, 1000);
  };

  // Send input to running program
  const sendInput = () => {
    if (!ws.current || ws.current.readyState !== WebSocket.OPEN) {
      addMessage('Not connected to server', false);
      return;
    }

    if (!inputValue.trim()) return;

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
        <h1>Multi-Language Code Runner</h1>

        <div className="controls">
          {!isConnected ? (
            <button onClick={() => connectWebSocket()}>Connect</button>
          ) : (
            <button onClick={cleanup}>Disconnect</button>
          )}

          <select
            value={currentLanguage}
            onChange={(e) => changeLanguage(e.target.value)}
            disabled={isConnected && isRunning}
          >
            {Object.entries(LANGUAGES).map(([key, lang]) => (
              <option key={key} value={key}>{lang.name}</option>
            ))}
          </select>
        </div>

        <div className="code-editor">
          <textarea
            value={code}
            onChange={(e) => setCode(e.target.value)}
            placeholder={`Enter ${LANGUAGES[currentLanguage].name} code`}
            disabled={isRunning}
            rows={10}
          />
          <div className="editor-buttons">
            <button
              onClick={executeCode}
              disabled={!isConnected || isRunning || !code.trim()}
            >
              {isRunning ? 'Running...' : 'Execute'}
            </button>
            <button
              onClick={() => setCode("")}
              disabled={isRunning}
            >
              Reset Code
            </button>
          </div>
        </div>

        <div className="terminal-container">
          <div className="terminal-header">
            <span>Terminal ({LANGUAGES[currentLanguage].name})</span>
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