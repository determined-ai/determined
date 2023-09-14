import { markdown, markdownLanguage } from '@codemirror/lang-markdown';
import { python } from '@codemirror/lang-python';
import { StreamLanguage } from '@codemirror/language';
import { yaml } from '@codemirror/legacy-modes/mode/yaml';
import ReactCodeMirror, { ReactCodeMirrorProps } from '@uiw/react-codemirror';
import React from 'react';

import { DarkLight } from 'components/kit/internal/types';
import useUI from 'components/kit/Theme';

interface Props extends ReactCodeMirrorProps {
  syntax: 'python' | 'markdown' | 'yaml';
}

const langs = {
  markdown: () => markdown({ base: markdownLanguage }),
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
