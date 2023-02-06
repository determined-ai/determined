// monaco languages
import 'monaco-editor/esm/vs/basic-languages/yaml/yaml.contribution';
import 'monaco-editor/esm/vs/basic-languages/python/python.contribution';
import 'monaco-editor/esm/vs/basic-languages/markdown/markdown.contribution';
// monaco features
import 'monaco-editor/esm/vs/editor/contrib/codelens/codelensController';
import 'monaco-editor/esm/vs/editor/contrib/find/findController';
import 'monaco-editor/esm/vs/editor/contrib/parameterHints/parameterHints';
import 'monaco-editor/esm/vs/editor/contrib/suggest/suggestController';
import 'monaco-editor/esm/vs/editor/contrib/wordHighlighter/wordHighlighter';
import 'monaco-editor/esm/vs/editor/standalone/browser/quickAccess/standaloneGotoSymbolQuickAccess';
import type * as monacoEditor from 'monaco-editor/esm/vs/editor/editor.api';
import React, { useCallback, useEffect, useRef } from 'react';
import ReactMonacoEditor, { MonacoEditorProps } from 'react-monaco-editor';

import useResize from 'hooks/useResize';
import useUI from 'shared/contexts/stores/UI';
import { DarkLight } from 'shared/themes';

import css from './MonacoEditor.module.scss';

/**
 * NOTE: non-basic modes like diffs might need manual loading of workers. refer
 * to
 * https://github.com/microsoft/monaco-editor/blob/main/docs/integrate-esm.md#using-vite
 */

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

  const handleEditorDidMount = useCallback(
    (editor: monacoEditor.editor.IStandaloneCodeEditor) => editor.focus(),
    [],
  );

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
