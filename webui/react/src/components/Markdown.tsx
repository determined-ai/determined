import { Tabs } from 'antd';
import { default as MarkdownViewer } from 'markdown-to-jsx';
import React from 'react';

import css from './Markdown.module.scss';
import Spinner from './Spinner';

const { TabPane } = Tabs;
const MonacoEditor = React.lazy(() => import('components/MonacoEditor'));

interface Props {
  editing?: boolean;
  markdown: string;
  onChange?: (editedMarkdown: string) => void;
}

interface RenderProps {
  markdown: string;
}

enum TabType {
  Edit = 'edit',
  Preview = 'preview'
}

const MarkdownRender: React.FC<RenderProps> = ({ markdown }) => {
  return (
    <div className={css.render}>
      <MarkdownViewer options={{ disableParsingRawHTML: true }}>
        {markdown}
      </MarkdownViewer>
    </div>
  );
};

const Markdown: React.FC<Props> = ({ editing = false, markdown, onChange }: Props) => {
  return (
    <div aria-label="markdown-editor" className={css.base}>
      {editing ? (
        <Tabs className="no-padding">
          <TabPane key={TabType.Edit} tab="Edit">
            <React.Suspense
              fallback={<div><Spinner tip="Loading text editor..." /></div>}>
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
                onChange={onChange}
              />
            </React.Suspense>
          </TabPane>
          <TabPane key={TabType.Preview} tab="Preview">
            <MarkdownRender markdown={markdown} />
          </TabPane>
        </Tabs>
      ) : (
        <MarkdownRender
          markdown={markdown || '######Add detailed description of this model...'}
        />
      )}
    </div>);
};

export default Markdown;
