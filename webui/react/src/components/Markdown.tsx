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
      style={{ overflow: 'auto' }}>
      {editing ? (
        <Tabs>
          <TabPane key={TabType.Edit} tab="Edit">
            <React.Suspense
              fallback={<div><Spinner tip="Loading text editor..." /></div>}>
              <div style={{ height, minHeight: 200 }}>
                <MonacoEditor
                  defaultValue={markdown}
                  language="markdown"
                  options={{
                    folding: false,
                    hideCursorInOverviewRuler: true,
                    lineDecorationsWidth: 8,
                    lineNumbersMinChars: 4,
                    occurrencesHighlight: false,
                    quickSuggestions: false,
                    renderLineHighlight: 'none',
                    wordWrap: 'on',
                  }}
                  width="100%"
                  onChange={onChange} />
              </div>
            </React.Suspense>
          </TabPane>
          <TabPane key={TabType.Preview} style={{ height, minHeight: 200 }} tab="Preview">
            <MarkdownViewer options={{ disableParsingRawHTML: true }}>
              {markdown}
            </MarkdownViewer>
          </TabPane>
        </Tabs>
      ) : (
        <MarkdownViewer options={{ disableParsingRawHTML: true }}>
          {markdown === '' ? '######Add detailed description of this model...' : markdown}
        </MarkdownViewer>
      )}
    </div>);
};

export default Markdown;
