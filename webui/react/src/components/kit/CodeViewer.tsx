import { DownloadOutlined, FileOutlined, LeftOutlined } from '@ant-design/icons';
import { Tree } from 'antd';
import { DataNode } from 'antd/lib/tree';
import { string } from 'io-ts';
import yaml from 'js-yaml';
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
import useResize from 'hooks/useResize';
import { SettingsConfig, useSettings } from 'hooks/useSettings';
import { handlePath, paths } from 'routes/utils';
import { getExperimentFileFromTree, getExperimentFileTree } from 'services/api';
import { V1FileNode } from 'services/api-ts-sdk';
import Icon from 'shared/components/Icon';
import Message, { MessageType } from 'shared/components/Message';
import Spinner from 'shared/components/Spinner';
import { RawJson, ValueOf } from 'shared/types';
import { ErrorType } from 'shared/utils/error';
import { AnyMouseEvent } from 'shared/utils/routes';
import handleError from 'utils/error';

const JupyterRenderer = lazy(() => import('./CodeViewer/IpynbRenderer'));

const { DirectoryTree } = Tree;

import css from './CodeViewer/CodeViewer.module.scss';

import './CodeViewer/index.scss';

type FileInfo = {
  content: string;
  name?: string;
};

export type Props = {
  experimentId?: number;
  files?: FileInfo[];
  runtimeConfig?: RawJson;
  submittedConfig?: string;
};

interface TreeNode extends DataNode {
  /**
   * DataNode is the interface antd works with. DateNode properties we are interested in:
   *
   * key: we use V1FileNode.path
   * title: name of node
   * icon: custom Icon component
   */
  children?: TreeNode[];
  text?: string;
}

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

const convertV1FileNodeToTreeNode = (node: V1FileNode): TreeNode => ({
  children: node.files?.map((n) => convertV1FileNodeToTreeNode(n)) ?? [],
  isLeaf: !node.isDir,
  key: node.path ?? '',
  title: node.name,
});

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

const configIcon = <Icon name="settings" />;

const isConfig = (key: unknown): key is Config =>
  key === Config.Submitted || key === Config.Runtime;

/**
 * A component responsible to enable the user to view the code for a experiment.
 *
 * It renders a file tree and a selected file in the MonacoEditor
 *
 * Props:
 *
 * experimentID: the experiment ID;
 *
 * files: an array of one or more files to display code;
 *
 * submittedConfig: the experiments.original_config property
 *
 * runtimeConfig: the config corresponding to the merged runtime config.
 */

const CodeViewer: React.FC<Props> = ({
  experimentId,
  files,
  submittedConfig: _submittedConfig,
  runtimeConfig: _runtimeConfig,
}) => {
  const resize = useResize();
  const firstConfig = useMemo(
    () => (_submittedConfig ? Config.Submitted : Config.Runtime),
    [_submittedConfig],
  );
  const configForExperiment = (experimentId: number): SettingsConfig<{ filePath: string }> => ({
    settings: {
      filePath: {
        defaultValue: firstConfig,
        storageKey: 'filePath',
        type: string,
      },
    },
    storagePath: `selected-file-${experimentId}`,
  });

  const { settings, updateSettings } = useSettings<{ filePath: string }>(
    configForExperiment(experimentId ?? 0),
  );

  const submittedConfig = useMemo(() => {
    if (!_submittedConfig) return;

    const { hyperparameters, ...restConfig } = yaml.load(_submittedConfig) as RawJson;

    // don't ask me why this works.. it gets rid of the JSON though
    return yaml.dump({ ...restConfig, hyperparameters });
  }, [_submittedConfig]);

  const runtimeConfig: string = useMemo(() => {
    /**
     * strip registry_auth from config for display
     * as well as workspace/project names
     */

    if (_runtimeConfig) {
      const {
        environment: { registry_auth, ...restEnvironment },
        workspace,
        project,
        ...restConfig
      } = _runtimeConfig;
      return yaml.dump({ environment: restEnvironment, ...restConfig });
    }
    return '';
  }, [_runtimeConfig]);

  const [pageError, setPageError] = useState<PageError>(PageError.None);

  const [treeData, setTreeData] = useState<TreeNode[]>([]);
  const [activeFile, setActiveFile] = useState<TreeNode>();
  const [isFetchingFile, setIsFetchingFile] = useState(false);
  const [isFetchingTree, setIsFetchingTree] = useState(false);
  const [downloadInfo, setDownloadInfo] = useState(DEFAULT_DOWNLOAD_INFO);
  const configDownloadButton = useRef<HTMLAnchorElement>(null);
  const timeout = useRef<NodeJS.Timeout>();
  const [viewMode, setViewMode] = useState<'tree' | 'editor' | 'split'>(() =>
    resize.width <= 1024 ? 'tree' : 'split',
  );
  const [editorMode, setEditorMode] = useState<'monaco' | 'ipynb'>('monaco');

  const switchTreeViewToEditor = useCallback(
    () => setViewMode((view) => (view === 'tree' ? 'editor' : view)),
    [],
  );
  const switchEditorViewToTree = useCallback(
    () => setViewMode((view) => (view === 'editor' ? 'tree' : view)),
    [],
  );
  const switchSplitViewToTree = useCallback(
    () => setViewMode((view) => (view === 'split' ? 'tree' : view)),
    [],
  );

  const handleSelectConfig = useCallback(
    (c: Config) => {
      const configText = c === Config.Submitted ? submittedConfig : runtimeConfig;

      if (configText) {
        setPageError(PageError.None);
      } else setPageError(PageError.Fetch);

      setActiveFile({
        icon: configIcon,
        key: c,
        text: configText,
        title: c,
      });
      switchTreeViewToEditor();
    },
    [submittedConfig, runtimeConfig, switchTreeViewToEditor],
  );

  const downloadHandler = useCallback(() => {
    timeout.current = setTimeout(() => {
      URL.revokeObjectURL(downloadInfo.url);
    }, 2000);
  }, [downloadInfo.url]);

  const fetchFileTree = useCallback(async () => {
    if (!experimentId) return;
    setIsFetchingTree(true);
    try {
      const fileTree = await getExperimentFileTree({ experimentId });
      setIsFetchingTree(false);

      const tree = fileTree
        .map<TreeNode>((node) => convertV1FileNodeToTreeNode(node))
        .sort(sortTree);

      if (runtimeConfig)
        tree.unshift({
          icon: configIcon,
          isLeaf: true,
          key: Config.Runtime,
          title: Config.Runtime,
        });

      if (submittedConfig)
        tree.unshift({
          icon: configIcon,
          isLeaf: true,
          key: Config.Submitted,
          title: Config.Submitted,
        });

      setTreeData(tree);
    } catch (error) {
      setIsFetchingTree(false);
      handleError(error, {
        publicMessage: 'Failed to load file tree.',
        publicSubject: 'Unable to fetch the model file tree.',
        silent: false,
        type: ErrorType.Api,
      });
    }
  }, [experimentId, runtimeConfig, submittedConfig]);

  const fetchFile = useCallback(
    async (path: string, title: string) => {
      setPageError(PageError.None);

      let file = '';
      try {
        file = await getExperimentFileFromTree({ experimentId, path });
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
      setActiveFile({
        key: path,
        text,
        title,
      });
    },
    [experimentId],
  );

  const handleSelectFile = useCallback(
    (_: React.Key[], info: { node: DataNode }) => {
      const selectedKey = String(info.node.key);

      if (selectedKey === activeFile?.key) {
        if (info.node.isLeaf) switchTreeViewToEditor();
        return;
      }

      if (isConfig(selectedKey)) {
        updateSettings({ filePath: String(info.node.key) });
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
        updateSettings({ filePath: String(info.node.key) });
      }
    },
    [activeFile?.key, treeData, switchTreeViewToEditor, updateSettings],
  );

  const getSyntaxHighlight = useCallback(() => {
    if (String(activeFile?.key).includes('.py')) return 'python';

    if (String(activeFile?.key).includes('.md')) return 'markdown';

    return 'yaml';
  }, [activeFile]);

  const handleDownloadClick = useCallback(
    (e: AnyMouseEvent) => {
      if (!activeFile || !experimentId) return;

      const filePath = String(activeFile?.key);
      if (isConfig(filePath)) {
        const isRuntimeConf = filePath === Config.Runtime;
        const url = isRuntimeConf
          ? URL.createObjectURL(new Blob([runtimeConfig]))
          : URL.createObjectURL(new Blob([submittedConfig as string]));

        setDownloadInfo({
          fileName: isRuntimeConf
            ? `${experimentId}_submitted_configuration.yaml`
            : `${experimentId}_runtime_configuration.yaml`,
          url,
        });
      } else {
        handlePath(e, {
          external: true,
          path: paths.experimentFileFromTree(experimentId, String(activeFile?.key)),
        });
      }
    },
    [activeFile, runtimeConfig, submittedConfig, experimentId],
  );

  // map the file tree
  useEffect(() => {
    fetchFileTree();
  }, [fetchFileTree]);

  // Set the selected node based on the active settings
  useEffect(() => {
    if (!settings.filePath) return;

    if (settings.filePath && activeFile?.key !== settings.filePath) {
      if (isConfig(settings.filePath)) {
        handleSelectConfig(settings.filePath);
      } else {
        const path = settings.filePath.split('/');
        const fileName = path[path.length - 1];

        setIsFetchingFile(true);
        fetchFile(settings.filePath, fileName);
        switchTreeViewToEditor();
      }
    }
  }, [
    treeData,
    settings.filePath,
    activeFile,
    fetchFile,
    handleSelectConfig,
    switchTreeViewToEditor,
  ]);

  // Set the code renderer to ipynb if needed
  useEffect(() => {
    const hasActiveFile = activeFile?.text;
    const isSameFile = activeFile?.key === settings.filePath;
    const isIpybnFile = /\.ipynb$/i.test(settings.filePath);

    if (hasActiveFile && isSameFile && isIpybnFile) {
      setEditorMode('ipynb');
    } else {
      setEditorMode('monaco');
    }
  }, [settings, activeFile]);

  useLayoutEffect(() => {
    if (configDownloadButton.current && downloadInfo.url && downloadInfo.fileName)
      configDownloadButton.current.click();
  }, [downloadInfo]);

  useEffect(() => {
    if (resize.width <= 1024) {
      switchSplitViewToTree();
    } else {
      setViewMode('split');
    }
  }, [resize.width, switchSplitViewToTree]);

  // clear the timeout ref from memory
  useEffect(() => {
    return () => {
      if (timeout.current) clearTimeout(timeout.current);
    };
  }, []);

  const classes = [css.base, pageError || isFetchingFile ? css.noEditor : ''];

  return (
    <section className={classes.join(' ')}>
      <Section className={viewMode === 'editor' ? css.hideElement : undefined} id="file-tree">
        <Spinner spinning={isFetchingTree}>
          <DirectoryTree
            className={css.fileTree}
            data-testid="fileTree"
            defaultExpandAll
            defaultSelectedKeys={
              viewMode
                ? // this is to ensure that, at least, the most parent node gets highlighted...
                  [settings.filePath.split('/')[0] ?? firstConfig]
                : undefined
            }
            treeData={treeData}
            onSelect={handleSelectFile}
          />
        </Spinner>
      </Section>
      {!!activeFile?.key && (
        <section className={viewMode === 'tree' ? css.hideElement : css.fileDir}>
          <div className={css.fileInfo}>
            <div className={css.buttonContainer}>
              <>
                {viewMode === 'editor' && (
                  <LeftOutlined className={css.leftChevron} onClick={switchEditorViewToTree} />
                )}
                {activeFile.icon ?? <FileOutlined />}
                <span className={css.filePath}>
                  <>{activeFile.title}</>
                </span>
                {isConfig(activeFile.key) && (
                  <span className={css.fileDesc}> {descForConfig[activeFile.key]}</span>
                )}
              </>
            </div>
            <div className={css.buttonsContainer}>
              {
                /**
                 * TODO: Add notebook integration
                 * <Button className={css.noBorderButton}>Open in Notebook</Button>
                 */
                <Tooltip title="Download File">
                  <DownloadOutlined className={css.noBorderButton} onClick={handleDownloadClick} />
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
        </section>
      )}
      <Section
        bodyNoPadding
        bodyScroll
        className={viewMode === 'tree' ? css.hideElement : pageError ? css.pageError : css.editor}
        maxHeight>
        <Spinner spinning={isFetchingFile}>
          {pageError ? (
            <Message
              style={{
                justifyContent: 'center',
                padding: '120px',
              }}
              title={pageError}
              type={MessageType.Alert}
            />
          ) : !isFetchingFile && !activeFile?.text ? (
            <h5>Please, choose a file to preview.</h5>
          ) : editorMode === 'monaco' ? (
            <MonacoEditor
              height="100%"
              language={getSyntaxHighlight()}
              options={{
                minimap: {
                  enabled: viewMode === 'split' && !!activeFile?.text?.length,
                  showSlider: 'mouseover',
                  size: 'fit',
                },
                occurrencesHighlight: false,
                readOnly: true,
                showFoldingControls: 'always',
              }}
              value={activeFile?.text}
            />
          ) : (
            <Suspense fallback={<Spinner tip="Loading ipynb viewer..." />}>
              <JupyterRenderer file={activeFile?.text || ''} />
            </Suspense>
          )}
        </Spinner>
      </Section>
    </section>
  );
};

export default CodeViewer;
