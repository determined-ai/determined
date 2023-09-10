import type { TabsProps } from 'antd';
import { default as MarkdownViewer } from 'markdown-to-jsx';
import React, { useMemo } from 'react';

import useResize from 'components/kit/internal/useResize';
import Pivot from 'components/kit/Pivot';
import Spinner from 'components/kit/Spinner';

import css from './Markdown.module.scss';

const CodeMirrorEditor = React.lazy(() => import('components/kit/internal/CodeMirrorEditor'));

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
  const { size } = useResize();
  const tabItems: TabsProps['items'] = useMemo(() => {
    return [
      {
        children: (
          <div className={css.noOverflow}>
            <React.Suspense
              fallback={
                <div>
                  <Spinner spinning tip="Loading text editor..." />
                </div>
              }>
              <CodeMirrorEditor
                basicSetup={{
                  autocompletion: false,
                  foldGutter: false,
                  highlightActiveLineGutter: false,
                }}
                height={`${size.height - 420}px`}
                syntax="markdown"
                value={markdown}
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
  }, [markdown, onChange, onClick, size]);

  return (
    <div aria-label="markdown-editor" className={css.base} tabIndex={0}>
      {editing && !disabled ? (
        <Pivot items={tabItems} />
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
