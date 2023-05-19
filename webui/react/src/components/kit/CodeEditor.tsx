import { DownloadOutlined, FileOutlined } from '@ant-design/icons';
import { Tree } from 'antd';
import React, { lazy, Suspense, useCallback, useEffect, useMemo, useState } from 'react';

import Tooltip from 'components/kit/Tooltip';
import MonacoEditor from 'components/MonacoEditor';
import Section from 'components/Section';
import Message, { MessageType } from 'shared/components/Message';
import Spinner from 'shared/components/Spinner';
import { ValueOf } from 'shared/types';
import { ErrorType } from 'shared/utils/error';
import { TreeNode } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

const JupyterRenderer = lazy(() => import('./CodeEditor/IpynbRenderer'));

const { DirectoryTree } = Tree;

import css from './CodeEditor/CodeEditor.module.scss';

import './CodeEditor/index.scss';

export type Props = {
  files: TreeNode[];
  onSelectFile?: (arg0: string) => void;
  readonly?: boolean;
  selectedFilePath?: string;
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

const PageError = {
  Decode: 'Could not decode file.',
  Empty: 'File has no content.',
  Fetch: 'Unable to fetch file.',
  None: '',
} as const;

type PageError = ValueOf<typeof PageError>;

const Config = {
  Runtime: 'Runtime Configuration',
  Submitted: 'Submitted Configuration',
} as const;

type Config = ValueOf<typeof Config>;

const descForConfig = {
  [Config.Submitted]: 'original submitted config',
  [Config.Runtime]: 'after merge with defaults and templates',
};

const isConfig = (key: unknown): key is Config =>
  key === Config.Submitted || key === Config.Runtime;

/**
 * A component responsible to enable the user to view the code for a experiment.
 *
 * It renders a file tree and a selected file in the MonacoEditor
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

const CodeEditor: React.FC<Props> = ({ files, onSelectFile, readonly, selectedFilePath }) => {
  const [pageError, setPageError] = useState<PageError>(PageError.None);
  const sortedFiles = useMemo(() => [...files].sort(sortTree), [files]);
  const [activeFile, setActiveFile] = useState<TreeNode | null>(sortedFiles[0] || null);

  const viewMode = useMemo(() => (files.length === 1 ? 'editor' : 'split'), [files.length]);
  const editorMode = useMemo(() => {
    const isIpybnFile = /\.ipynb$/i.test(String(activeFile?.key || ''));
    return isIpybnFile ? 'ipynb' : 'monaco';
  }, [activeFile]);

  const fetchFile = useCallback(async (fileInfo: TreeNode) => {
    if (!fileInfo) return;
    setPageError(PageError.None);

    if (isConfig(fileInfo.key) || fileInfo.content !== NotLoaded) {
      setActiveFile(fileInfo);
      return;
    }

    let file,
      content: Loadable<string> = NotLoaded;
    try {
      file = await fileInfo.get?.(String(fileInfo.key));
    } catch (error) {
      handleError(error, {
        publicMessage: 'Failed to load selected file.',
        publicSubject: 'Unable to fetch the selected file.',
        silent: false,
        type: ErrorType.Api,
      });
      setPageError(PageError.Fetch);
    }
    if (!file) {
      setActiveFile({
        ...fileInfo,
        content: NotLoaded,
      });
      return;
    }

    try {
      const text = decodeURIComponent(escape(window.atob(file)));

      if (!text) setPageError(PageError.Empty); // Emmits a "Empty file" error message
      content = Loaded(text);
      setActiveFile({
        ...fileInfo,
        content,
      });
    } catch {
      setPageError(PageError.Decode);
    }
  }, []);

  useEffect(() => {
    if (selectedFilePath && activeFile?.key !== selectedFilePath) {
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
      if (matchTopFileOrFolder) {
        fetchFile(matchTopFileOrFolder);
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

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
        setPageError(PageError.Fetch);
        return;
      }

      if (targetNode.isLeaf) {
        onSelectFile?.(String(targetNode.key));
        fetchFile(targetNode);
      }
    },
    [activeFile?.key, fetchFile, files, onSelectFile],
  );

  const handleDownloadClick = useCallback(() => {
    if (!activeFile) return;

    const link = document.createElement('a');

    link.download = isConfig(activeFile?.key)
      ? activeFile.download || ''
      : String(activeFile.title);
    link.href = URL.createObjectURL(new Blob([Loadable.getOrElse('', activeFile?.content)]));
    link.dispatchEvent(new MouseEvent('click'));
    setTimeout(() => {
      URL.revokeObjectURL(link.href);
    }, 2000);
  }, [activeFile]);

  const getSyntaxHighlight = useCallback(() => {
    if (String(activeFile?.key).includes('.py')) return 'python';

    if (String(activeFile?.key).includes('.md')) return 'markdown';

    return 'yaml';
  }, [activeFile]);

  const classes = [
    css.fileTree,
    css.codeEditorBase,
    pageError ? css.noEditor : '',
    viewMode === 'editor' ? css.editorMode : '',
  ];

  let fileContent = <h5>Please, choose a file to preview.</h5>;
  if (pageError) {
    fileContent = (
      <Message
        style={{
          justifyContent: 'center',
          padding: '120px',
        }}
        title={pageError}
        type={MessageType.Alert}
      />
    );
  } else if (activeFile) {
    fileContent =
      editorMode === 'monaco' ? (
        <MonacoEditor
          height="100%"
          language={getSyntaxHighlight()}
          options={{
            minimap: {
              enabled: false,
            },
            occurrencesHighlight: false,
            readOnly: readonly,
            showFoldingControls: 'always',
          }}
          value={Loadable.getOrElse('', activeFile.content)}
        />
      ) : (
        <Suspense fallback={<Spinner tip="Loading ipynb viewer..." />}>
          <JupyterRenderer file={Loadable.getOrElse('', activeFile.content)} />
        </Suspense>
      );
  }

  return (
    <div className={classes.join(' ')}>
      <div className={viewMode === 'editor' ? css.hideElement : undefined}>
        <DirectoryTree
          className={css.fileTree}
          data-testid="fileTree"
          defaultExpandAll
          defaultSelectedKeys={[selectedFilePath || sortedFiles[0]?.key]}
          treeData={sortedFiles}
          onSelect={handleSelectFile}
        />
      </div>
      {!!activeFile?.key && (
        <div className={css.fileDir}>
          <div className={css.fileInfo}>
            <div className={css.buttonContainer}>
              <>
                {activeFile.icon ?? <FileOutlined />}
                <span className={css.filePath}>
                  <>{activeFile.title}</>
                </span>
                {isConfig(activeFile.key) && (
                  <span className={css.fileDesc}> {descForConfig[activeFile.key]}</span>
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
                      readonly && activeFile?.content !== NotLoaded
                        ? css.noBorderButton
                        : css.hideElement
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
        bodyScroll
        className={pageError ? css.pageError : css.editor}
        maxHeight>
        <Spinner spinning={activeFile?.content === NotLoaded}>{fileContent}</Spinner>
      </Section>
    </div>
  );
};

export default CodeEditor;
