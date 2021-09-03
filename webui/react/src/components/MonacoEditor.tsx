import React, { useEffect, useRef } from 'react';
import ReactMonacoEditor, { MonacoEditorProps } from 'react-monaco-editor';

import useResize from 'hooks/useResize';

import css from './MonacoEditor.module.scss';

const PADDING = 8;

const MonacoEditor: React.FC<MonacoEditorProps> = ({
  height = '100%',
  language = 'yaml',
  options = {},
  theme = 'vs-light',
  ...props
}: MonacoEditorProps) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const editorRef = useRef<ReactMonacoEditor>(null);
  const resize = useResize(containerRef);

  useEffect(() => {
    editorRef.current?.editor?.layout();
  }, [ resize ]);

  return (
    <div className={css.base} ref={containerRef}>
      <ReactMonacoEditor
        height={height}
        language={language}
        options={{
          minimap: { enabled: false },
          padding: { bottom: PADDING, top: PADDING },
          scrollBeyondLastLine: false,
          selectOnLineNumbers: true,
          ...options,
        }}
        ref={editorRef}
        theme={theme}
        {...props}
      />
    </div>
  );
};

export default MonacoEditor;
