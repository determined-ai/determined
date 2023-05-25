import { markdown, markdownLanguage } from '@codemirror/lang-markdown';
import { python } from '@codemirror/lang-python';
import { StreamLanguage } from '@codemirror/language';
import { languages } from '@codemirror/language-data';
import { yaml } from '@codemirror/legacy-modes/mode/yaml';
import ReactCodeMirror, { ReactCodeMirrorProps } from '@uiw/react-codemirror';
import React from 'react';

import useUI from 'shared/contexts/stores/UI';
import { DarkLight } from 'shared/themes';

interface Props extends ReactCodeMirrorProps {
  syntax: 'python' | 'markdown' | 'yaml';
}

const langs = {
  markdown: () => markdown({ base: markdownLanguage, codeLanguages: languages }),
  python,
  yaml: () => StreamLanguage.define(yaml),
};

const CodeMirrorEditor: React.FC<Props> = ({ syntax, ...props }) => {
  const { ui } = useUI();

  return (
    <ReactCodeMirror
      extensions={[langs[syntax]()]}
      theme={ui.darkLight === DarkLight.Dark ? 'dark' : 'light'}
      {...props}
    />
  );
};

export default CodeMirrorEditor;
