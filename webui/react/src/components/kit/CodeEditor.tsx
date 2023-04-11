import { DownloadOutlined, FileOutlined } from '@ant-design/icons';
import { Tree } from 'antd';
import React, {
  lazy,
  Suspense,
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from 'react';

import Tooltip from 'components/kit/Tooltip';
import MonacoEditor from 'components/MonacoEditor';
import Section from 'components/Section';
import { handlePath } from 'routes/utils';
import Message, { MessageType } from 'shared/components/Message';
import Spinner from 'shared/components/Spinner';
import { ValueOf } from 'shared/types';
import { ErrorType } from 'shared/utils/error';
import { AnyMouseEvent } from 'shared/utils/routes';
import { TreeNode } from 'types';
import handleError from 'utils/error';
<<<<<<< HEAD
<<<<<<< HEAD
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';
=======
import { Loadable } from 'utils/loadable';
>>>>>>> e5d871c08 (readonly, re-sync, prep for loadable)
=======
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';
>>>>>>> 110e8f963 (single trial experiment tree)

const JupyterRenderer = lazy(() => import('./CodeEditor/IpynbRenderer'));

const { DirectoryTree } = Tree;

import css from './CodeEditor/CodeEditor.module.scss';

import './CodeEditor/index.scss';

<<<<<<< HEAD
<<<<<<< HEAD
=======
type FileInfo = {
  content: Loadable<string>;
  name: string;
};

>>>>>>> e5d871c08 (readonly, re-sync, prep for loadable)
export type Props = {
<<<<<<< HEAD
  files: TreeNode[];
=======
  files: FileInfo[];
>>>>>>> 773beb50c (start working on readonly)
=======
export type Props = {
  files: TreeNode[];
>>>>>>> 110e8f963 (single trial experiment tree)
  onSelectFile?: (arg0: string) => void;
  readonly?: boolean;
  selectedFilePath?: string;
};

const DEFAULT_DOWNLOAD_INFO = {
  fileName: '',
  url: '',
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

<<<<<<< HEAD
<<<<<<< HEAD
=======
const convertV1FileNodeToTreeNode = (node: V1FileNode): TreeNode => ({
  children: node.files?.map((n) => convertV1FileNodeToTreeNode(n)) ?? [],
  isLeaf: !node.isDir,
  key: node.path ?? '',
  title: node.name,
});

const convertFileInfoToTreeNode = (file: FileInfo): TreeNode => ({
  children: [],
  isLeaf: true,
  key: file.name ?? '',
  text: Loadable.match(file.content, {
    Loaded: (content) => content,
    NotLoaded: () => '',
  }),
  title: file.name,
});

>>>>>>> e5d871c08 (readonly, re-sync, prep for loadable)
=======
>>>>>>> 110e8f963 (single trial experiment tree)
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

<<<<<<< HEAD
<<<<<<< HEAD
const CodeEditor: React.FC<Props> = ({ files, onSelectFile, readonly, selectedFilePath }) => {
=======
const CodeEditor: React.FC<Props> = ({
  files,
  onSelectFile,
  readonly,
  selectedFilePath,
}) => {
  // const firstConfig = useMemo(
  //   () => (_submittedConfig ? Config.Submitted : Config.Runtime),
  //   [_submittedConfig],
  // );
  const [fileViewerInfo, setFileViewerInfo] = useState<{ filePath: string; fileText?: string }>({
    filePath: selectedFilePath || '',
  });

  // const submittedConfig = useMemo(() => {
  //   if (!_submittedConfig) return;
  //
  //   const { hyperparameters, ...restConfig } = yaml.load(_submittedConfig) as RawJson;
  //
  //   // don't ask me why this works.. it gets rid of the JSON though
  //   return yaml.dump({ ...restConfig, hyperparameters });
  // }, [_submittedConfig]);

  // const runtimeConfig: string = useMemo(() => {
  //   /**
  //    * strip registry_auth from config for display
  //    * as well as workspace/project names
  //    */
  //
  //   if (_runtimeConfig) {
  //     const {
  //       environment: { registry_auth, ...restEnvironment },
  //       workspace,
  //       project,
  //       ...restConfig
  //     } = _runtimeConfig;
  //     return yaml.dump({ environment: restEnvironment, ...restConfig });
  //   }
  //   return '';
  // }, [_runtimeConfig]);

>>>>>>> 773beb50c (start working on readonly)
  const [pageError, setPageError] = useState<PageError>(PageError.None);

<<<<<<< HEAD
  const [activeFile, setActiveFile] = useState<TreeNode | null>(files[0] || null);
  const [downloadInfo, setDownloadInfo] = useState(DEFAULT_DOWNLOAD_INFO);
  const configDownloadButton = useRef<HTMLAnchorElement>(null);
  const timeout = useRef<NodeJS.Timeout>();

  const viewMode = useMemo(() => (files.length === 1 ? 'editor' : 'split'), [files.length]);
  const editorMode = useMemo(() => {
    const isIpybnFile = /\.ipynb$/i.test(String(activeFile?.key || ''));
    return isIpybnFile ? 'ipynb' : 'monaco';
  }, [activeFile]);
=======
  const [treeData, setTreeData] = useState<TreeNode[]>([]);
  const [activeFile, setActiveFile] = useState<TreeNode | null>(
    files.length === 1 ? convertFileInfoToTreeNode(files[0]) : null,
  );
  const [isFetchingFile, setIsFetchingFile] = useState(false);
  const [isFetchingTree, setIsFetchingTree] = useState(false);
=======
const CodeEditor: React.FC<Props> = ({ files, onSelectFile, readonly, selectedFilePath }) => {
  const [pageError, setPageError] = useState<PageError>(PageError.None);

  const [activeFile, setActiveFile] = useState<TreeNode | null>(files.length > 0 ? files[0] : null);
>>>>>>> 110e8f963 (single trial experiment tree)
  const [downloadInfo, setDownloadInfo] = useState(DEFAULT_DOWNLOAD_INFO);
  const configDownloadButton = useRef<HTMLAnchorElement>(null);
  const timeout = useRef<NodeJS.Timeout>();

<<<<<<< HEAD
  // const handleSelectConfig = useCallback(
  //   (c: Config) => {
  //     if (files.length) return;
  //     const configText = c === Config.Submitted ? submittedConfig : runtimeConfig;
  //
  //     if (configText) {
  //       setPageError(PageError.None);
  //     } else setPageError(PageError.Fetch);
  //
  //     setActiveFile({
  //       icon: configIcon,
  //       key: c,
  //       text: configText,
  //       title: c,
  //     });
  //   },
  //   [files, submittedConfig, runtimeConfig],
  // );
>>>>>>> e5d871c08 (readonly, re-sync, prep for loadable)
=======
  const viewMode = useMemo(() => (files.length === 1 ? 'editor' : 'split'), [files.length]);
  const editorMode = useMemo(() => {
    const isIpybnFile = /\.ipynb$/i.test(String(activeFile?.key || ''));
    return isIpybnFile ? 'ipynb' : 'monaco';
  }, [activeFile]);
>>>>>>> 110e8f963 (single trial experiment tree)

  const downloadHandler = useCallback(() => {
    timeout.current = setTimeout(() => {
      URL.revokeObjectURL(downloadInfo.url);
    }, 2000);
  }, [downloadInfo.url]);

<<<<<<< HEAD
<<<<<<< HEAD
=======
>>>>>>> 110e8f963 (single trial experiment tree)
  const fetchFile = useCallback(async (fileInfo: TreeNode) => {
    if (!fileInfo) return;
    setPageError(PageError.None);

    if (isConfig(fileInfo.key) || fileInfo.content !== NotLoaded) {
      setActiveFile(fileInfo);
      return;
<<<<<<< HEAD
    }

    let file,
      content: Loadable<string> = NotLoaded;
    try {
      file = await fileInfo.get?.(String(fileInfo.key));
    } catch (error) {
<<<<<<< HEAD
      handleError(error, {
        publicMessage: 'Failed to load selected file.',
        publicSubject: 'Unable to fetch the selected file.',
        silent: false,
        type: ErrorType.Api,
      });
      setPageError(PageError.Fetch);
    }
    if (!file) {
=======
  const fetchFileTree = useCallback(async () => {
    if (files && files.length > 1) {
      setTreeData(files.map(convertFileInfoToTreeNode).sort(sortTree));
=======
>>>>>>> 110e8f963 (single trial experiment tree)
    }

<<<<<<< HEAD
  const fetchFile = useCallback(
    async (path: string, title: string) => {
      if (!files.length) return;
      setPageError(PageError.None);

      const file = '';
      try {
        // file = await getExperimentFileFromTree({ experimentId, path });
      } catch (error) {
        handleError(error, {
          publicMessage: 'Failed to load selected file.',
          publicSubject: 'Unable to fetch the selected file.',
          silent: false,
          type: ErrorType.Api,
        });
        setPageError(PageError.Fetch);
      } finally {
        setIsFetchingFile(false);
      }

      let text = '';
      try {
        text = decodeURIComponent(escape(window.atob(file)));

        if (!text) setPageError(PageError.Empty); // Emmits a "Empty file" error message
      } catch {
        setPageError(PageError.Decode);
      }
>>>>>>> e5d871c08 (readonly, re-sync, prep for loadable)
      setActiveFile({
        ...fileInfo,
        content: NotLoaded,
      });
<<<<<<< HEAD
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

  const treeData = useMemo(() => {
    if (selectedFilePath && activeFile?.key !== selectedFilePath) {
      const matchTopFileOrFolder = files.find((f) => f.key === selectedFilePath);
=======
    let file,
      content: Loadable<string> = NotLoaded;
    try {
      file = await fileInfo.get?.(String(fileInfo.key));
    } catch (error) {
      console.error(error);
=======
>>>>>>> e525324d8 (ExperimentCodeViewer allows single- and multi- to load)
      handleError(error, {
        publicMessage: 'Failed to load selected file.',
        publicSubject: 'Unable to fetch the selected file.',
        silent: false,
        type: ErrorType.Api,
      });
      setPageError(PageError.Fetch);
    }

    try {
      const text = decodeURIComponent(escape(window.atob(file || '')));

      if (!text) setPageError(PageError.Empty); // Emmits a "Empty file" error message
      content = Loaded(text);
    } catch {
      setPageError(PageError.Decode);
    }
    setActiveFile({
      ...fileInfo,
      content,
    });
  }, []);

  const treeData = useMemo(() => {
<<<<<<< HEAD
    if (selectedFilePath) {
      const matchTopFileOrFolder = files.find((f) => f.key === selectedFilePath.split('/')[0]);
>>>>>>> 110e8f963 (single trial experiment tree)
=======
    if (selectedFilePath && activeFile?.key !== selectedFilePath) {
      const matchTopFileOrFolder = files.find((f) => f.key === selectedFilePath);
>>>>>>> e525324d8 (ExperimentCodeViewer allows single- and multi- to load)
      if (matchTopFileOrFolder) {
        fetchFile(matchTopFileOrFolder);
      }
    }
    return files.sort(sortTree);
<<<<<<< HEAD
<<<<<<< HEAD
  }, [files, selectedFilePath, activeFile?.key, fetchFile]);
=======
    },
    [files],
  );
>>>>>>> e5d871c08 (readonly, re-sync, prep for loadable)
=======
  }, [files, fetchFile, selectedFilePath]);
>>>>>>> 110e8f963 (single trial experiment tree)
=======
  }, [files, selectedFilePath, activeFile?.key, fetchFile]);
>>>>>>> e525324d8 (ExperimentCodeViewer allows single- and multi- to load)

  const handleSelectFile = useCallback(
    (_: React.Key[], info: { node: TreeNode }) => {
      const selectedKey = String(info.node.key);

      if (selectedKey === activeFile?.key) {
        // already selected
        return;
      }

      const nodeAddress = selectedKey.split('/');

      let targetNode = treeData.find((node) => node.title === nodeAddress[0]);
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
    [activeFile?.key, fetchFile, treeData, onSelectFile],
  );

  const getSyntaxHighlight = useCallback(() => {
    if (String(activeFile?.key).includes('.py')) return 'python';

    if (String(activeFile?.key).includes('.md')) return 'markdown';

    return 'yaml';
  }, [activeFile]);

  const handleDownloadClick = useCallback(
    (e: AnyMouseEvent) => {
      if (!activeFile) return;

      const filePath = String(activeFile?.key);
<<<<<<< HEAD
<<<<<<< HEAD
<<<<<<< HEAD
      if (activeFile.content !== NotLoaded) {
        const url = URL.createObjectURL(new Blob([Loadable.getOrElse('', activeFile.content)]));
        setDownloadInfo({
          fileName: isConfig(filePath) ? activeFile.download || '' : String(activeFile.title),
          url,
        });
      } else if (activeFile.download) {
        handlePath(e, {
          external: true,
          path: activeFile.download,
        });
=======
      // if (isConfig(filePath)) {
      //   // const isRuntimeConf = filePath === Config.Runtime;
      //   // const url = isRuntimeConf
      //     // ? URL.createObjectURL(new Blob([runtimeConfig]))
      //     // : URL.createObjectURL(new Blob([submittedConfig as string]));
      //
      //   setDownloadInfo({
      //     fileName: isRuntimeConf
      //       ? `${experimentId}_submitted_configuration.yaml`
      //       : `${experimentId}_runtime_configuration.yaml`,
      //     url,
      //   });
      // } else
      if (activeFile.key && activeFile.text) {
        const url = URL.createObjectURL(new Blob([activeFile.text]));
=======
      if (isConfig(filePath) && activeFile.content !== NotLoaded) {
        const url = URL.createObjectURL(
          Loadable.match(activeFile.content, {
            Loaded: (content) => new Blob([content]),
            NotLoaded: () => new Blob(),
          }),
        );
>>>>>>> 110e8f963 (single trial experiment tree)
=======
      if (activeFile.content !== NotLoaded) {
        const url = URL.createObjectURL(new Blob([Loadable.getOrElse('', activeFile.content)]));
>>>>>>> ddb0557b9 (test and download button fixes)
        setDownloadInfo({
          fileName: isConfig(filePath) ? activeFile.download || '' : String(activeFile.title),
          url,
        });
<<<<<<< HEAD
<<<<<<< HEAD
>>>>>>> e5d871c08 (readonly, re-sync, prep for loadable)
=======
      } else if (activeFile.key) {
=======
      } else if (activeFile.download) {
>>>>>>> ddb0557b9 (test and download button fixes)
        handlePath(e, {
          external: true,
          path: activeFile.download,
        });
>>>>>>> 110e8f963 (single trial experiment tree)
      }
    },
    [activeFile],
  );

<<<<<<< HEAD
<<<<<<< HEAD
=======
  // map the file tree
  useEffect(() => {
    fetchFileTree();
  }, [fetchFileTree]);

  // Set the selected node based on the active settings
  useEffect(() => {
    if (!fileViewerInfo.filePath) return;

    if (activeFile?.key !== fileViewerInfo.filePath) {
      onSelectFile?.(fileViewerInfo.filePath);

      if (isConfig(fileViewerInfo.filePath)) {
        // handleSelectConfig(fileViewerInfo.filePath);
      } else {
        const path = fileViewerInfo.filePath.split('/');
        const fileName = path[path.length - 1];

        if (files.length) {
          setActiveFile({
            key: path[path.length - 1],
            text: fileViewerInfo.fileText,
            title: path,
          } as TreeNode);
        } else {
          setIsFetchingFile(true);
          fetchFile(fileViewerInfo.filePath, fileName);
        }
      }
    }
  }, [
    treeData,
    files.length,
    fileViewerInfo,
    activeFile,
    fetchFile,
    // handleSelectConfig,
    onSelectFile,
  ]);

  // Set the code renderer to ipynb if needed
  useEffect(() => {
    const hasActiveFile = activeFile?.text;
    const isSameFile = activeFile?.key === fileViewerInfo.filePath;
    const isIpybnFile = /\.ipynb$/i.test(fileViewerInfo.filePath);

    if (hasActiveFile && isSameFile && isIpybnFile) {
      setEditorMode('ipynb');
    } else {
      setEditorMode('monaco');
    }
  }, [fileViewerInfo.filePath, activeFile]);

>>>>>>> e5d871c08 (readonly, re-sync, prep for loadable)
=======
>>>>>>> 110e8f963 (single trial experiment tree)
  useLayoutEffect(() => {
    if (configDownloadButton.current && downloadInfo.url && downloadInfo.fileName)
      configDownloadButton.current.click();
  }, [downloadInfo]);

  // clear the timeout ref from memory
  useEffect(() => {
    return () => {
      if (timeout.current) clearTimeout(timeout.current);
    };
  }, []);

  const classes = [
    css.fileTree,
    css.codeEditorBase,
    pageError ? css.noEditor : '',
    viewMode === 'editor' ? css.editorMode : '',
  ];

  return (
    <div className={classes.join(' ')}>
<<<<<<< HEAD
      <div className={viewMode === 'editor' ? css.hideElement : undefined}>
        <DirectoryTree
          className={css.fileTree}
          data-testid="fileTree"
          defaultExpandAll
          defaultSelectedKeys={[selectedFilePath ? selectedFilePath.split('/')[0] : files[0]?.key]}
          treeData={treeData}
          onSelect={handleSelectFile}
        />
=======
      <div className={viewMode === 'editor' ? css.hideElement : undefined} id="file-tree">
<<<<<<< HEAD
        <Spinner spinning={isFetchingTree}>
          <DirectoryTree
            className={css.fileTree}
            data-testid="fileTree"
            defaultExpandAll
            defaultSelectedKeys={
              viewMode
                ? // this is to ensure that, at least, the most parent node gets highlighted...
                  [fileViewerInfo.filePath.split('/')[0] ?? ''] //firstConfig]
                : undefined
            }
            treeData={treeData}
            onSelect={handleSelectFile}
          />
        </Spinner>
>>>>>>> e5d871c08 (readonly, re-sync, prep for loadable)
=======
        <DirectoryTree
          className={css.fileTree}
          data-testid="fileTree"
          defaultExpandAll
          defaultSelectedKeys={[selectedFilePath ? selectedFilePath.split('/')[0] : files[0]?.key]}
          treeData={treeData}
          onSelect={handleSelectFile}
        />
>>>>>>> 110e8f963 (single trial experiment tree)
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
                <Tooltip title="Download File">
                  <DownloadOutlined
<<<<<<< HEAD
                    className={
                      readonly && activeFile?.content !== NotLoaded
                        ? css.noBorderButton
                        : css.hideElement
                    }
=======
                    className={readonly ? css.noBorderButton : css.hideElement}
>>>>>>> 773beb50c (start working on readonly)
                    onClick={handleDownloadClick}
                  />
                  {/* this is an invisible button to programatically download the config files */}
                  <a
                    aria-disabled
                    className={css.hideElement}
                    download={downloadInfo.fileName}
                    href={downloadInfo.url}
                    ref={configDownloadButton}
                    onClick={downloadHandler}
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
        <Spinner spinning={activeFile?.content === NotLoaded}>
          {pageError ? (
            <Message
              style={{
                justifyContent: 'center',
                padding: '120px',
              }}
              title={pageError}
              type={MessageType.Alert}
            />
          ) : !activeFile ? (
            <h5>Please, choose a file to preview.</h5>
          ) : editorMode === 'monaco' ? (
            <MonacoEditor
              height="100%"
              language={getSyntaxHighlight()}
              options={{
                minimap: {
<<<<<<< HEAD
                  enabled: false,
=======
                  enabled: ['split', 'editor'].includes(viewMode) && !!activeFile?.content,
                  showSlider: 'mouseover',
                  size: 'fit',
>>>>>>> 110e8f963 (single trial experiment tree)
                },
                occurrencesHighlight: false,
                readOnly: readonly,
                showFoldingControls: 'always',
              }}
<<<<<<< HEAD
              value={Loadable.getOrElse('', activeFile.content)}
            />
          ) : (
            <Suspense fallback={<Spinner tip="Loading ipynb viewer..." />}>
              <JupyterRenderer file={Loadable.getOrElse('', activeFile.content)} />
=======
              value={Loadable.match(activeFile.content, {
                Loaded: (code) => code,
                NotLoaded: () => '',
              })}
            />
          ) : (
            <Suspense fallback={<Spinner tip="Loading ipynb viewer..." />}>
              <JupyterRenderer
                file={Loadable.match(activeFile.content, {
                  Loaded: (code) => code,
                  NotLoaded: () => '',
                })}
              />
>>>>>>> 110e8f963 (single trial experiment tree)
            </Suspense>
          )}
        </Spinner>
      </Section>
    </div>
  );
};

export default CodeEditor;
