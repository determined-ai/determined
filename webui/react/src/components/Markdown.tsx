import { default as MarkdownViewer } from 'markdown-to-jsx';
import React from 'react';

import Spinner from './Spinner';

const MonacoEditor = React.lazy(() => import('components/MonacoEditor'));

interface Props {
  editing: boolean;
  markdown: string;
}

const Markdown: React.FC<Props> = ({ editing, markdown }: Props) => {
  return (
    <div>{editing ?
      <React.Suspense
        fallback={<div><Spinner tip="Loading text editor..." /></div>}>
        <MonacoEditor />
      </React.Suspense> :
      <MarkdownViewer options={{ disableParsingRawHTML: true }}>{markdown}</MarkdownViewer>}
    </div>
  );
};

export default Markdown;
