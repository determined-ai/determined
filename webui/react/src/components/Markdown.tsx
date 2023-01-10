import type { TabsProps } from 'antd';
import { default as MarkdownViewer } from 'markdown-to-jsx';
import React, { useMemo } from 'react';

import Pivot from 'components/kit/Pivot';
import Spinner from 'shared/components/Spinner/Spinner';

import css from './Markdown.module.scss';

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
  const tabItems: TabsProps['items'] = useMemo(() => {
    return [
      {
        children: (
          <div className={css.noOverflow}>
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
          </div>
        ),
        key: TabType.Edit,
        label: 'Edit',
      },
      {
        children: <MarkdownRender markdown={markdown} onClick={onClick} />,
        key: TabType.Preview,
        label: 'Preview',
      },
    ];
  }, [markdown, onChange, onClick]);

  return (
    <div aria-label="markdown-editor" className={css.base} tabIndex={0}>
      {editing && !disabled ? (
        // TODO: Clean up once we standardize page layouts
        <div style={{ height: '100%', padding: 16 }}>
          <Pivot items={tabItems} />
        </div>
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
