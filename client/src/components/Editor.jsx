import React, { useEffect, useRef } from 'react';
import Convert from 'ansi-to-html';

const Terminal = ({ messages }) => {
    const convert = new Convert({
        fg: '#FFF',
        bg: '#000',
        newline: true,
        escapeXML: true
    });

    const terminalRef = useRef(null);

    // Auto-scroll to bottom when new messages arrive
    useEffect(() => {
        if (terminalRef.current) {
            terminalRef.current.scrollTop = terminalRef.current.scrollHeight;
        }
    }, [messages]);

    return (
        <div className="terminal" ref={terminalRef}>
            {messages.map((msg, i) => (
                <div
                    key={i}
                    className={msg.isOutput ? 'output' : 'error'}
                    dangerouslySetInnerHTML={{
                        __html: convert.toHtml(msg.text)
                    }}
                />
            ))}
        </div>
    );
};

export default Terminal;