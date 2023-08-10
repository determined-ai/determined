import { DownloadOutlined, FileOutlined } from '@ant-design/icons';
import { markdown, markdownLanguage } from '@codemirror/lang-markdown';
import { python } from '@codemirror/lang-python';
import { StreamLanguage } from '@codemirror/language';
import { json } from '@codemirror/legacy-modes/mode/javascript';
import { yaml } from '@codemirror/legacy-modes/mode/yaml';
import ReactCodeMirror from '@uiw/react-codemirror';
import { Tree } from 'antd';
import React, { lazy, Suspense, useCallback, useMemo } from 'react';

import Message, { MessageType } from 'components/kit/internal/Message';
import Section from 'components/kit/internal/Section';
import { ErrorHandler } from 'components/kit/internal/types';
import { DarkLight, TreeNode, ValueOf } from 'components/kit/internal/types';
import Spinner from 'components/kit/Spinner';
import Tooltip from 'components/kit/Tooltip';
import useUI from 'stores/contexts/UI';
import { Loadable, NotLoaded } from 'utils/loadable';

const JupyterRenderer = lazy(() => import('./CodeEditor/IpynbRenderer'));

const { DirectoryTree } = Tree;

import css from './CodeEditor/CodeEditor.module.scss';

import './CodeEditor/index.scss';

const MARKDOWN_CONFIG = {
  autocompletion: false,
  foldGutter: false,
  highlightActiveLineGutter: false,
};

type ErrorMessage = {
  _tag: 'Error';
  message: string;
};
// TODO: consider lifting this to loadable proper as in WEB-1333
export type LoadableOrError<T> = Loadable<T> | ErrorMessage;

export const ErrorMessage = (message: string): ErrorMessage => ({
  _tag: 'Error' as const,
  message,
});

const isErrorMessage = (f: unknown): f is ErrorMessage =>
  !!f && typeof f === 'object' && '_tag' in f && f._tag === 'Error';

export type SingleFileProps = {
  files: [TreeNode];
  onSelectFile?: never;
  selectedFilePath?: never;
};

export type MultiFileProps = {
  files: TreeNode[];
  onSelectFile: (filePath: string) => void;
  selectedFilePath: string;
};

export type Props = (SingleFileProps | MultiFileProps) & {
  file: LoadableOrError<string>;
  onError: ErrorHandler; // only used to raise ipynb errors
  height?: string; // height of the container.
  onChange?: (fileContent: string) => void; // only use in single-file editing
  readonly?: boolean;
};

const sortTree = (a: TreeNode, b: TreeNode) => {
  if (a.children) a.children.sort(sortTree);

  if (b.children) b.children.sort(sortTree);
  // sorting first by having an extension or not, then by extension first
  // and finally alphabetically.
  const titleA = String(a.title);
  const titleB = String(b.title);

  if (isConfig(a.key) && !isConfig(b.key)) return -1;
  if (!isConfig(a.key) && isConfig(b.key)) return 1;
  // submitted before runtime for whatever reason
  if (isConfig(a.key) && isConfig(b.key)) return a.key < b.key ? 1 : -1;

  if (!a.isLeaf && b.isLeaf) return -1;

  if (a.isLeaf && !b.isLeaf) return 1;

  if (!a.isLeaf && !b.isLeaf) return titleA.localeCompare(titleB) - titleB.localeCompare(titleA);

  // had to use RegEx due to some files being ".<filename>"
  const [stringA, extensionA] = titleA.split(/^(?=[a-zA-Z])\./);
  const [stringB, extensionB] = titleB.split(/^(?=[a-zA-Z])\./);

  if (!extensionA && extensionB) return 1;

  if (!extensionB && extensionA) return -1;

  if (!extensionA && !extensionB)
    return stringA.localeCompare(stringB) - stringB.localeCompare(stringA);

  return extensionA.localeCompare(extensionB) - extensionB.localeCompare(extensionB);
};

const Config = {
  Runtime: 'Runtime Configuration',
  Submitted: 'Submitted Configuration',
} as const;

type Config = ValueOf<typeof Config>;

const isConfig = (key: unknown): key is Config =>
  key === Config.Submitted || key === Config.Runtime;

const langs = {
  json: () => StreamLanguage.define(json),
  markdown: () => markdown({ base: markdownLanguage }),
  python,
  yaml: () => StreamLanguage.define(yaml),
};

/**
 * A component responsible to enable the user to view the code for a experiment.
 *
 * It renders a file tree and a selected file in the CodeMirror editor.
 *
 * Props:
 *
 * files: an array of one or more files to display code;
 *
 * onSelectFile: called with filename when user changes files in the tree;
 *
 * readonly: prevent user from making changes to code files;
 *
 * selectedFilePath: gives path to the file to set as activeFile;
 */

const CodeEditor: React.FC<Props> = ({
  files,
  height = '100%',
  file,
  onChange,
  onError,
  onSelectFile,
  readonly,
  selectedFilePath = String(files[0]?.key),
}) => {
  const sortedFiles = useMemo(() => [...files].sort(sortTree), [files]);
  const { ui } = useUI();

  const viewMode = useMemo(() => (files.length === 1 ? 'editor' : 'split'), [files.length]);
  const activeFile = useMemo(() => {
    if (viewMode === 'editor') {
      return files[0];
    }
    if (!selectedFilePath) {
      return null;
    }
    const splitFilePath = selectedFilePath.split('/');
    let matchTopFileOrFolder = null;
    let fileDir = files;
    for (let dir = 0; dir < splitFilePath.length; dir++) {
      matchTopFileOrFolder = fileDir.find(
        (f) => f.key === splitFilePath[dir] || f.key === selectedFilePath,
      );
      if (matchTopFileOrFolder?.children) {
        fileDir = matchTopFileOrFolder.children;
      }
    }
    return matchTopFileOrFolder;
  }, [files, selectedFilePath, viewMode]);
  const editorMode = useMemo(() => {
    const isIpybnFile = /\.ipynb$/i.test(String(activeFile?.key || ''));
    return isIpybnFile ? 'ipynb' : 'codemirror';
  }, [activeFile?.key]);

  const syntax = useMemo(() => {
    if (String(activeFile?.key).includes('.py')) return 'python';
    if (String(activeFile?.key).includes('.json')) return 'json';
    if (String(activeFile?.key).includes('.md')) return 'markdown';

    return 'yaml';
  }, [activeFile?.key]);

  const handleSelectFile = useCallback(
    (_: React.Key[], info: { node: TreeNode }) => {
      const selectedKey = String(info.node.key);

      if (selectedKey === activeFile?.key) {
        // already selected
        return;
      }

      const nodeAddress = selectedKey.split('/');

      let targetNode = files.find((node) => node.title === nodeAddress[0]);
      for (const dir of nodeAddress.slice(1))
        targetNode = targetNode?.children?.find((file) => file.title === dir);

      if (!targetNode) {
        return;
      }

      if (targetNode.isLeaf) {
        onSelectFile?.(String(targetNode.key));
      }
    },
    [activeFile?.key, files, onSelectFile],
  );

  const handleDownloadClick = useCallback(() => {
    if (!Loadable.isLoadable(file) || !Loadable.isLoaded(file) || !activeFile) return;

    const link = document.createElement('a');

    link.download = isConfig(activeFile?.key)
      ? activeFile.download || ''
      : String(activeFile.title);
    link.href = URL.createObjectURL(new Blob([Loadable.getOrElse('', file)]));
    link.dispatchEvent(new MouseEvent('click'));
    setTimeout(() => {
      URL.revokeObjectURL(link.href);
    }, 2000);
  }, [activeFile, file]);

  const classes = [
    css.codeEditorBase,
    isErrorMessage(file) ? css.noEditor : '',
    viewMode === 'editor' ? css.editorMode : '',
  ];

  const sectionClasses = [isErrorMessage(file) ? css.pageError : css.editor];

  const treeClasses = [css.fileTree, viewMode === 'editor' ? css.hideElement : ''];

  let fileContent = <h5>Please, choose a file to preview.</h5>;
  if (isErrorMessage(file)) {
    fileContent = (
      <Message
        style={{
          justifyContent: 'center',
          padding: '120px',
        }}
        title={file.message}
        type={MessageType.Alert}
      />
    );
  } else if (activeFile) {
    fileContent =
      editorMode === 'codemirror' ? (
        <ReactCodeMirror
          basicSetup={syntax === 'markdown' ? MARKDOWN_CONFIG : undefined}
          extensions={[langs[syntax]()]}
          height="100%"
          readOnly={readonly}
          style={{ height: '100%' }}
          theme={ui.darkLight === DarkLight.Dark ? 'dark' : 'light'}
          value={Loadable.getOrElse('', file)}
          onChange={onChange}
        />
      ) : (
        <Suspense fallback={<Spinner spinning tip="Loading ipynb viewer..." />}>
          <JupyterRenderer file={Loadable.getOrElse('', file)} onError={onError} />
        </Suspense>
      );
  }

  return (
    <div className={classes.join(' ')} style={{ height }}>
      <DirectoryTree
        className={treeClasses.join(' ')}
        data-testid="fileTree"
        defaultExpandAll
        defaultSelectedKeys={[selectedFilePath || sortedFiles[0]?.key]}
        treeData={sortedFiles}
        onSelect={handleSelectFile}
      />
      {!!activeFile?.title && (
        <div className={css.fileDir}>
          <div className={css.fileInfo}>
            <div className={css.buttonContainer}>
              <>
                {activeFile.icon ?? <FileOutlined />}
                <span className={css.filePath}>
                  <>{activeFile.title}</>
                </span>
                {activeFile?.subtitle && (
                  <span className={css.fileDesc}> {activeFile?.subtitle}</span>
                )}
                {readonly && <span className={css.readOnly}>read-only</span>}
              </>
            </div>
            <div className={css.buttonsContainer}>
              {
                /**
                 * TODO: Add notebook integration
                 * <Button className={css.noBorderButton}>Open in Notebook</Button>
                 */
                <Tooltip content="Download File">
                  <DownloadOutlined
                    className={
                      readonly && file !== NotLoaded ? css.noBorderButton : css.hideElement
                    }
                    onClick={handleDownloadClick}
                  />
                </Tooltip>
              }
            </div>
          </div>
        </div>
      )}
      <Section
        bodyNoPadding
        bodyScroll={height === '100%'}
        className={sectionClasses.join(' ')}
        maxHeight>
        {/* directly checking tag because loadable.isLoaded only takes loadables */}
        <Spinner spinning={file === NotLoaded}>{fileContent}</Spinner>
      </Section>
    </div>
  );
};

export default CodeEditor;
