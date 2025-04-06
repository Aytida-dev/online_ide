import React from "react";
import MonacoEditor from "react-monaco-editor";

function Editor({ code, onChange }) {
    return (
        <MonacoEditor
            width="800"
            height="400"
            language="javascript"
            theme="vs-dark"
            value={code}
            onChange={onChange}
        />
    );
}

export default Editor;