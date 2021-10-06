import { Tabs } from 'antd';
import { default as MarkdownViewer } from 'markdown-to-jsx';
import React from 'react';

import css from './Markdown.module.scss';
import Spinner from './Spinner';

const { TabPane } = Tabs;
const MonacoEditor = React.lazy(() => import('components/MonacoEditor'));

interface Props {
  editing?: boolean;
  height?: string | number;
  markdown: string;
  onChange?: (editedMarkdown: string) => void;
}

enum TabType {
  Edit = 'edit',
  Preview = 'preview'
}

const Markdown: React.FC<Props> = ({ editing=false, height='100%', markdown, onChange }: Props) => {
  return (
    <div
      aria-label="markdown-editor"
      className={css.base}
      style={{ height, overflow: 'auto' }}>
      {editing ? (
        <Tabs>
          <TabPane key={TabType.Edit} tab="Edit">
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
          <TabPane key={TabType.Preview} tab="Preview">
            <MarkdownViewer options={{ disableParsingRawHTML: true }}>
              {markdown}
            </MarkdownViewer>
          </TabPane>
        </Tabs>
      ) : (
        <MarkdownViewer options={{ disableParsingRawHTML: true }}>
          {markdown}
        </MarkdownViewer>
      )}
    </div>);
};

export default Markdown;
