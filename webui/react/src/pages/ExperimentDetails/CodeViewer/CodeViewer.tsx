import { DownloadOutlined, FileOutlined, LeftOutlined } from '@ant-design/icons';
import { Tooltip, Tree } from 'antd';
import { DataNode } from 'antd/lib/tree';
import yaml from 'js-yaml';
import React, { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';

import MonacoEditor from 'components/MonacoEditor';
import Section from 'components/Section';
import useResize from 'hooks/useResize';
import { handlePath, paths } from 'routes/utils';
import { getExperimentFileFromTree, getExperimentFileTree } from 'services/api';
import { V1FileNode } from 'services/api-ts-sdk';
import Icon from 'shared/components/Icon';
import Message, { MessageType } from 'shared/components/Message';
import Spinner from 'shared/components/Spinner';
import { RawJson } from 'shared/types';
import { ErrorType } from 'shared/utils/error';
import handleError from 'utils/error';

const { DirectoryTree } = Tree;

import css from './CodeViewer.module.scss';

import './index.scss';

export type Props = {
  experimentId: number;
  runtimeConfig?: RawJson;
  submittedConfig: string;
}

interface TreeNode extends DataNode {
  /**
   * DataNode is the interface antd works with. DateNode properties we are interested in:
   *
   * key: we use V1FileNode.path
   * title: name of node
   * icon: custom Icon component
   */
  children?: TreeNode[]
  text?: string;
}

const sortTree = (a:TreeNode, b: TreeNode) => {
  if (a.children) a.children.sort(sortTree);

  if (b.children) b.children.sort(sortTree);
  // sorting first by having an extension or not, then by extension first
  // and finally alphabetically.
  const titleA = String(a.title);
  const titleB = String(b.title);

  if (!a.isLeaf && b.isLeaf) return -1;

  if (a.isLeaf && !b.isLeaf) return 1;

  if (!a.isLeaf && !b.isLeaf)
    return titleA.localeCompare(titleB) - titleB.localeCompare(titleA);

  // had to use RegEx due to some files being ".<filename>"
  const [ stringA, extensionA ] = titleA.split(/(?<=[a-zA-Z])\./);
  const [ stringB, extensionB ] = titleB.split(/(?<=[a-zA-Z])\./);

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

enum PageError {
  decode = 'Could not decode file.',
  empty = 'File has no content.',
  fetch = 'Unable to fetch file.',
  none = ''
}

enum Config {
  submitted = 'Submitted Configuration',
  runtime = 'Runtime Configuration'
}

const descForConfig = {
  [Config.submitted]: 'original submitted config',
  [Config.runtime]: 'after merge with defaults and templates',
};

const configIcon = <Icon name="settings" />;

const isConfig = (key: unknown): key is Config =>
  key === Config.submitted || key === Config.runtime;

/**
 * A component responsible to enable the user to view the code for a experiment.
 *
 * It renders a file tree and a selected file in the MonacoEditor
 *
 * Props:
 *
 * experimentID: the experiment ID;
 *
 * submittedConfig: the experiments.original_config property
 *
 * runtimeConfig: the config corresponding to the merged runtime config.
 */

const CodeViewer: React.FC<Props> = ({
  experimentId,
  submittedConfig: _submittedConfig,
  runtimeConfig: _runtimeConfig,
}) => {
  const resize = useResize();

  const submittedConfig = useMemo(() => {
    if (!_submittedConfig) return;

    const { hyperparameters, ...restConfig } = yaml.load(_submittedConfig) as RawJson;

    // don't ask me why this works.. it gets rid of the JSON though
    return yaml.dump({ ...restConfig, hyperparameters });
  }, [ _submittedConfig ]);

  const runtimeConfig:string = useMemo(() => {
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
  }, [ _runtimeConfig ]);

  const [ pageError, setPageError ] = useState<PageError>(PageError.none);

  const [ treeData, setTreeData ] = useState<TreeNode[]>([]);
  const [ activeFile, setActiveFile ] = useState<TreeNode>();
  const [ isFetchingFile, setIsFetchingFile ] = useState(false);
  const [ isFetchingTree, setIsFetchingTree ] = useState(false);
  const [ downloadInfo, setDownloadInfo ] = useState({
    fileName: '',
    url: '',
  });
  const configDownloadButton = useRef<HTMLAnchorElement>(null);
  const timeout = useRef<NodeJS.Timeout>();
  const [ viewMode, setViewMode ] = useState<'tree' | 'editor' | 'split'>(
    () => resize.width <= 1024 ? 'tree' : 'split',
  );

  const switchTreeViewToEditor = useCallback(
    () => setViewMode((view) => view === 'tree' ? 'editor' : view)
    , [],
  );
  const switchEditorViewToTree = useCallback(
    () => setViewMode((view) => view === 'editor' ? 'tree' : view)
    , [],
  );

  const switchSplitViewToTree = useCallback(
    () => setViewMode((view) => view === 'split' ? 'tree' : view)
    , [],
  );

  const handleSelectConfig = useCallback((c: Config) => {
    const configText = c === Config.submitted ? submittedConfig : runtimeConfig;

    if (configText) {
      setPageError(PageError.none);

    } else setPageError(PageError.fetch);
    setActiveFile({
      icon: configIcon,
      key: c,
      text: configText,
      title: c,
    });
    switchTreeViewToEditor();
  }, [ submittedConfig, runtimeConfig, switchTreeViewToEditor ]);

  const downloadHandler = useCallback(() => {
    timeout.current = setTimeout(() => {
      URL.revokeObjectURL(downloadInfo.url);
    }, 2000);
  }, [ downloadInfo.url ]);

  const fetchFileTree = useCallback(
    async () => {
      setIsFetchingTree(true);
      try {
        const fileTree = await getExperimentFileTree({ experimentId });
        setIsFetchingTree(false);

        const tree = fileTree
          .map<TreeNode>((node) => convertV1FileNodeToTreeNode(node))
          .sort(sortTree);

        if (runtimeConfig) tree.unshift({
          icon: configIcon,
          isLeaf: true,
          key: Config.runtime,
          title: Config.runtime,
        });

        if (submittedConfig) tree.unshift({
          icon: configIcon,
          isLeaf: true,
          key: Config.submitted,
          title: Config.submitted,
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
    },
    [ experimentId, runtimeConfig, submittedConfig ],
  );

  const fetchFile = useCallback(async (path, title) => {
    setPageError(PageError.none);

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
      setPageError(PageError.fetch);
    } finally {
      setIsFetchingFile(false);
    }

    let text = '';
    try {
      text = decodeURIComponent(escape(window.atob(file)));

      if (!text) setPageError(PageError.empty); // Emmits a "Empty file" error message
    } catch {
      setPageError(PageError.decode);
    }
    setActiveFile({
      key: path,
      text,
      title,
    });
  }, [ experimentId ]);

  const handleSelectFile = useCallback(async (
    _,
    info: { node: DataNode },
  ) => {
    const selectedKey = String(info.node.key);
    const selectedTitle = info.node.title;

    if (selectedKey === activeFile?.key) {
      if (info.node.isLeaf) switchTreeViewToEditor();
      return;
    }

    if (isConfig(selectedKey)) {
      handleSelectConfig(selectedKey);
      return;
    }

    const nodeAddress = selectedKey.split('/');

    let targetNode = treeData.find((node) => node.title === nodeAddress[0]);
    for (const dir of nodeAddress.slice(1))
      targetNode = targetNode?.children?.find((file) => file.title === dir);

    if (!targetNode) {
      setPageError(PageError.fetch);
      return;
    }

    if (targetNode.isLeaf) {
      setIsFetchingFile(true);
      await fetchFile(selectedKey, selectedTitle);
      switchTreeViewToEditor();
    }
  }, [
    fetchFile,
    activeFile?.key,
    handleSelectConfig,
    treeData,
    switchTreeViewToEditor,
  ]);

  const getSyntaxHighlight = useCallback(() => {
    if (String(activeFile?.key).includes('.py')) return 'python';

    if (String(activeFile?.key).includes('.md')) return 'markdown';

    return 'yaml';
  }, [ activeFile ]);

  const handleDownloadClick = useCallback((e) => {
    if (!activeFile) return;

    const filePath = String(activeFile?.key);
    if (filePath.includes('Configuration')) {
      const isRuntimeConf = filePath.includes('runtime');
      const url = isRuntimeConf
        ? URL.createObjectURL(new Blob([ runtimeConfig ]))
        : URL.createObjectURL(new Blob([ submittedConfig as string ]));

      setDownloadInfo({
        fileName: isRuntimeConf
          ? 'runtimeConfiguration.yaml'
          : 'generatedConfiguration.yaml',
        url,
      });
    } else {
      handlePath(e, {
        external: true,
        path: paths.experimentFileFromTree(
          experimentId,
          String(activeFile?.key),
        ),
      });
    }
  }, [ activeFile, runtimeConfig, submittedConfig, experimentId ]);

  useEffect(() => {
    if (submittedConfig) {
      handleSelectConfig(Config.submitted);
    } else {
      handleSelectConfig(Config.runtime);
    }
  }, [ handleSelectConfig, submittedConfig ]);

  useEffect(() => {
    if (resize.width <= 1024) {
      switchSplitViewToTree();
    } else {
      setViewMode('split');
    }
  }, [ resize.width, switchSplitViewToTree ]);

  // map the file tree
  useEffect(() => {
    fetchFileTree();
  }, [ fetchFileTree ]);

  // clear the timeout ref from memory
  useEffect(() => {
    return () => {
      if (timeout.current) clearTimeout(timeout.current);
    };
  }, []);

  useLayoutEffect(() => {
    if (
      configDownloadButton.current
      && downloadInfo.url
      && downloadInfo.fileName
    ) configDownloadButton.current.click();
  }, [ downloadInfo ]);

  const classes = [ css.base, pageError ? css.noEditor : '' ];

  return (
    <section className={classes.join(' ')}>
      <Section className={viewMode === 'editor' ? css.hideElement : undefined} id="fileTree">
        <Spinner spinning={isFetchingTree}>
          <DirectoryTree
            className={css.fileTree}
            data-testid="fileTree"
            defaultExpandAll
            defaultSelectedKeys={viewMode ? [ Config.submitted ] : undefined}
            treeData={treeData}
            onSelect={handleSelectFile}
          />
        </Spinner>
      </Section>
      {
        !!activeFile?.key && (
          <section className={
            viewMode === 'tree'
              ? css.hideElement : css.fileDir}>
            <div className={css.fileInfo}>
              <div className={css.buttonContainer}>
                {
                  viewMode === 'editor' && (
                    <LeftOutlined
                      className={css.leftChevron}
                      onClick={switchEditorViewToTree}
                    />
                  )
                }
                {activeFile.icon ?? <FileOutlined />}
                <span className={css.filePath}>{activeFile.title}</span>
                {isConfig(activeFile.key) && (
                  <span className={css.fileDesc}>  {descForConfig[activeFile.key]}</span>
                )}
              </div>
              <div className={css.buttonsContainer}>
                {
                /**
                  * TODO: Add notebook integration
                  * <Button className={css.noBorderButton}>Open in Notebook</Button>
                  */
                  <Tooltip title="Download File">
                    <DownloadOutlined
                      className={css.noBorderButton}
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
          </section>
        )
      }
      <Section
        bodyNoPadding
        bodyScroll
        className={viewMode === 'tree' ? css.hideElement : pageError ? css.pageError : css.editor}
        maxHeight>
        <Spinner spinning={isFetchingFile}>
          {
            pageError ? (
              <Message
                style={{
                  justifyContent: 'center',
                  padding: '120px',
                }}
                title={pageError}
                type={MessageType.Alert}
              />
            )
              : !isFetchingFile && !activeFile?.text
                ? <h5>Please, choose a file to preview.</h5>
                : (
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
                )
          }
        </Spinner>
      </Section>
    </section>
  );
};

export default CodeViewer;
