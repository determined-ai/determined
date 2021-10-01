import { Tabs } from 'antd';
import { default as MarkdownViewer } from 'markdown-to-jsx';
import React from 'react';

import css from './Markdown.module.scss';
import Spinner from './Spinner';

const { TabPane } = Tabs;
const MonacoEditor = React.lazy(() => import('components/MonacoEditor'));

interface Props {
  editing?: boolean;
  height?: string;
  markdown: string;
  onChange?: (editedMarkdown: string) => void;
}

const Markdown: React.FC<Props> = ({ editing=false, height='80vh', markdown, onChange }: Props) => {
  return (
    editing ?<Tabs>
      <TabPane key="1" tab="Edit">
        <React.Suspense
          fallback={<div><Spinner tip="Loading text editor..." /></div>}>
          <MonacoEditor
            defaultValue={markdown}
            height={height}
            language="markdown"
            options={{
              wordWrap: 'on',
              wrappingIndent: 'indent',
            }}
            width="100%"
            onChange={onChange} />
        </React.Suspense>
      </TabPane>
      <TabPane key="2" tab="Preview">
        <div className={css.base} style={{ height }}>
          <MarkdownViewer options={{ disableParsingRawHTML: true }}>
            {markdown}
          </MarkdownViewer>
        </div>
      </TabPane>
    </Tabs> :
      <div className={css.base} style={{ height, overflow: 'auto' }}>
        <MarkdownViewer options={{ disableParsingRawHTML: true }}>
          {markdown}
        </MarkdownViewer>
      </div>

  );
};

export default Markdown;
