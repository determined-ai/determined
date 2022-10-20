import { Tabs } from 'antd';
import { default as MarkdownViewer } from 'markdown-to-jsx';
import React from 'react';

import Spinner from 'shared/components/Spinner/Spinner';

import css from './Markdown.module.scss';

const { TabPane } = Tabs;
const MonacoEditor = React.lazy(() => import('components/MonacoEditor'));

interface Props {
  disabled?: boolean;
  editing?: boolean;
  markdown: string;
  onChange?: (editedMarkdown: string) => void;
  onClick?: (e: React.MouseEvent) => void;
}

interface RenderProps {
  markdown: string;
  onClick?: (e: React.MouseEvent) => void;
  placeholder?: string;
}

const TabType = {
  Edit: 'edit',
  Preview: 'preview',
} as const;

const MarkdownRender: React.FC<RenderProps> = ({ markdown, placeholder, onClick }) => {
  const showPlaceholder = !markdown && placeholder;
  return (
    <div className={css.render} onClick={onClick}>
      {showPlaceholder ? (
        <div className={css.placeholder}>{placeholder}</div>
      ) : (
        <MarkdownViewer options={{ disableParsingRawHTML: true }}>{markdown}</MarkdownViewer>
      )}
    </div>
  );
};

const Markdown: React.FC<Props> = ({
  disabled = false,
  editing = false,
  markdown,
  onChange,
  onClick,
}: Props) => {
  return (
    <div aria-label="markdown-editor" className={css.base} tabIndex={0}>
      {editing && !disabled ? (
        <Tabs className="no-padding">
          <TabPane className={css.noOverflow} key={TabType.Edit} tab="Edit">
            <React.Suspense
              fallback={
                <div>
                  <Spinner tip="Loading text editor..." />
                </div>
              }>
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
            <MarkdownRender markdown={markdown} onClick={onClick} />
          </TabPane>
        </Tabs>
      ) : (
        <MarkdownRender
          markdown={markdown}
          placeholder={disabled ? 'No note present.' : 'Add notes...'}
          onClick={onClick}
        />
      )}
    </div>
  );
};

export default Markdown;
