import React, { useCallback, useEffect, useRef } from 'react';
import ReactMonacoEditor, { MonacoEditorProps } from 'react-monaco-editor';

import useResize from 'hooks/useResize';
import useUI from 'shared/contexts/stores/UI';
import { DarkLight } from 'shared/themes';

import css from './MonacoEditor.module.scss';

const PADDING = 8;

const MonacoEditor: React.FC<MonacoEditorProps> = ({
  height = '100%',
  language = 'yaml',
  options = {},
  ...props
}: MonacoEditorProps) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const editorRef = useRef<ReactMonacoEditor>(null);
  const resize = useResize(containerRef);
  const { ui } = useUI();

  const handleEditorDidMount = useCallback((editor) => editor.focus(), []);

  useEffect(() => {
    editorRef.current?.editor?.layout();
  }, [resize]);

  return (
    <div className={css.base} ref={containerRef}>
      <ReactMonacoEditor
        editorDidMount={handleEditorDidMount}
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
        theme={ui.darkLight === DarkLight.Dark ? 'vs-dark' : 'vs-light'}
        {...props}
      />
    </div>
  );
};

export default MonacoEditor;
