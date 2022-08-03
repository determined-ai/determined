import { DownloadOutlined, FileOutlined, LeftOutlined } from '@ant-design/icons';
import { Tooltip, Tree } from 'antd';
import { DataNode } from 'antd/lib/tree';
import yaml from 'js-yaml';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import MonacoEditor from 'components/MonacoEditor';
import Section from 'components/Section';
import useResize from 'hooks/useResize';
import { handlePath, paths } from 'routes/utils';
import { getExperimentFileFromTree, getExperimentFileTree } from 'services/api';
import { V1FileNode as FileNode } from 'services/api-ts-sdk';
import Spinner from 'shared/components/Spinner';
import { RawJson } from 'shared/types';
import { isEqual } from 'shared/utils/data';
import { ErrorType } from 'shared/utils/error';
import handleError from 'utils/error';

const { DirectoryTree } = Tree;

import css from './CodeViewer.module.scss';
import './index.scss';

export type Props = {
  configRaw?: RawJson;
  experimentId: number;
}

type FileInfo = {
  data: string;
  name: string;
  path: string;
};

/**
 * A component responsible to enable the user to view the code for a experiment.
 *
 * It renders a file tree and a selected file in the MonacoEditor
 *
 * Props:
 *
 * experimentID: the experiment ID;
 *
 * configRaw: the experiment.configRaw property to be used to render a Config yaml file;
 */
const CodeViewer: React.FC<Props> = ({ experimentId, configRaw }) => {
  const resize = useResize();

  const [ config ] = useState(() => {
    /**
   * strip registry_auth from config for display
   * as well as workspace/project names
   */
    if (configRaw) {
      const {
        environment: { registry_auth, ...restEnvironment },
        workspace,
        project,
        ...restConfig
      } = configRaw;
      return { environment: restEnvironment, ...restConfig };
    }
  });
  // Data structure to be used by the Tree
  const [ files, setFiles ] = useState<FileNode[]>([]);
  const [ isFetching, setIsFetching ] = useState(false);
  const [ fileInfo, setFileInfo ] = useState<FileInfo>();
  const [ viewMode, setViewMode ] = useState<'tree' | 'editor' | undefined>(
    () => resize.width <= 1024 ? 'tree' : undefined,
  ); // To be used in the mobile view, switches the UI
  const navigateTree = useCallback((node: FileNode, key: string): DataNode => {
    const newNode: DataNode = {
      className: 'treeNode',
      isLeaf: true,
      key,
      title: node.name,
    };

    if (node.files?.length) {
      newNode.children = node.files.map(
        (chNode) => navigateTree(chNode, chNode.path || ''),
      );
      newNode.isLeaf = false;
    }

    return newNode;
  }, []);
  const onSelectFile = useCallback(async (
    keys: React.Key[],
    info: { [key: string]: unknown, node: DataNode },
  ) => {
    if (info.node.title === fileInfo?.name) return; // avoid making unecessary processing.

    if (info.node.title === 'Configuration') {
      setFileInfo({
        data: yaml.dump(config),
        name: 'Configuration',
        path: 'Configuration',
      });

      return;
    }

    const filePath = String(info.node.key);

    // check if the selected node is a file
    if (!files.find((file) => file.path === filePath)?.isDir) {
      setIsFetching(true);

      try {
        const file = await getExperimentFileFromTree({ experimentId, path: filePath });

        setIsFetching(false);
        setFileInfo({
          data: decodeURIComponent(escape(window.atob(file))),
          name: info.node.title as string,
          path: filePath,
        });

        if (resize.width <= 1024) {
          setViewMode('editor');
        }

      } catch (error) {
        setIsFetching(false);

        handleError(error, {
          publicMessage: 'Failed to load selected file.',
          publicSubject: 'Unable to fetch the selected file.',
          silent: false,
          type: ErrorType.Api,
        });
      }
    }
  }, [ resize.width, setViewMode, config, experimentId, fileInfo?.name, files ]);
  const fetchFiles = useCallback(
    async () => {
      try {
        const newFiles = await getExperimentFileTree({ experimentId });

        if (isEqual(newFiles, files)) return;

        setFiles(newFiles);
      } catch (error) {
        handleError(error, {
          publicMessage: 'Failed to load file tree.',
          publicSubject: 'Unable to fetch the model file tree.',
          silent: false,
          type: ErrorType.Api,
        });
      }
    },
    [ experimentId, files ],
  );
  const fileTree = useMemo(() => {
    if (config) {
      setFileInfo({
        data: yaml.dump(config),
        name: 'Configuration',
        path: 'Configuration',
      });

      return [
        {
          className: 'treeNode',
          isLeaf: true,
          key: 'configuration',
          title: 'Configuration',
        },
        ...files.map<DataNode>((node) => navigateTree(node, node.path || '')),
      ];
    }

    return files.map<DataNode>((node) => navigateTree(node, node.path || ''));
  }, [ files, config, navigateTree ]);

  const treeClasses: string[] = [];
  const editorClasses: string[] = [];

  if ((resize.width <= 1024) && (viewMode === 'editor')) {
    treeClasses.push(css.hideElement);
    editorClasses.pop();
  }

  if ((resize.width <= 1024) && (viewMode === 'tree')) {
    treeClasses.pop();
    editorClasses.push(css.hideElement);
  }

  // map the file tree
  useEffect(() => {
    fetchFiles();
    // TODO: have error handling added.
  }, [ fetchFiles ]);

  return (
    <section className={css.base}>
      <Section className={treeClasses.join(' ')} id="fileTree">
        <DirectoryTree
          className={css.fileTree}
          data-testid="fileTree"
          defaultExpandAll
          defaultSelectedKeys={(config && resize.width > 1024) ? [ '0-0' ] : undefined}
          treeData={fileTree}
          onSelect={onSelectFile}
        />
      </Section>
      {
        !!fileInfo?.path && (
          <Spinner className={editorClasses.join(' ')} spinning={isFetching}>
            <section className={css.fileDir}>
              <div className={css.fileInfo}>
                <div className={css.buttonContainer}>
                  {
                    resize.width <= 1024 && (
                      <LeftOutlined
                        className={css.leftChevron}
                        onClick={() => setViewMode('tree')}
                      />
                    )
                  }
                  <FileOutlined />
                  <span className={css.filePath}>{fileInfo.name}</span>
                </div>
                <div className={css.buttonsContainer}>
                  {/* <Button className={css.noBorderButton}>Open in Notebook</Button>
                  TODO: this will be added in the future*/}
                  {
                    !fileInfo.path.includes('Configuration') && (
                      <Tooltip title="Download File">
                        <DownloadOutlined
                          className={css.noBorderButton}
                          onClick={(e) => handlePath(e, {
                            external: true,
                            path: paths.experimentFileFromTree(experimentId, fileInfo.path),
                          })}
                        />
                      </Tooltip>
                    )
                  }
                </div>
              </div>
            </section>
          </Spinner>
        )
      }
      <Section
        bodyNoPadding
        bodyScroll
        className={editorClasses.join(' ')}
        id="editor"
        maxHeight>
        <Spinner spinning={isFetching}>
          {
            !isFetching && !fileInfo?.data
              ? <h5>Please, choose a file to preview.</h5>
              : (
                <MonacoEditor
                  height="100%"
                  language="yaml"
                  options={{
                    minimap: {
                      enabled: !!fileInfo?.data.length,
                      showSlider: 'mouseover',
                      size: 'fit',
                    },
                    occurrencesHighlight: false,
                    readOnly: true,
                  }}
                  value={fileInfo?.data}
                />
              )
          }
        </Spinner>
      </Section>
    </section>
  );
};

export default CodeViewer;
