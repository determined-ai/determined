import { langs } from '@uiw/codemirror-extensions-langs';
import ReactCodeMirror, { ReactCodeMirrorProps } from '@uiw/react-codemirror';
// import { markdown, markdownLanguage } from '@codemirror/lang-markdown';
// import { python } from '@codemirror/lang-python';
// import { languages } from '@codemirror/language-data';
import React from 'react';

import useUI from 'shared/contexts/stores/UI';
import { DarkLight } from 'shared/themes';

interface Props extends ReactCodeMirrorProps {
  syntax: 'python' | 'markdown' | 'yaml';
}

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
