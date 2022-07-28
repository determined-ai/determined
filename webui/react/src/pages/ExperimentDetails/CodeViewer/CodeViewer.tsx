import { DownloadOutlined, FileOutlined, LeftOutlined } from '@ant-design/icons';
import { Tooltip, Tree } from 'antd';
import { DataNode } from 'antd/lib/tree';
import classNames from 'classnames';
import yaml from 'js-yaml';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import MonacoEditor from 'components/MonacoEditor';
import Section from 'components/Section';
import useRecize from 'hooks/useResize';
import { handlePath, paths } from 'routes/utils';
import { getExperimentFileFromTree, getExperimentFileTree } from 'services/api';
import { V1FileNode as FileNode } from 'services/api-ts-sdk';
import Spinner from 'shared/components/Spinner';
import { RawJson } from 'shared/types';
import { isEqual } from 'shared/utils/data';

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

  const { width: documentWidth } = useRecize();

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
    () => documentWidth <= 1024 ? 'tree' : undefined,
  ); // To be used in the mobile view, switches the UI
  const fileTree = useMemo(() => {
    const navigateTree = (node: FileNode, key: string): DataNode => {
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
    };

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
  }, [ files, config ]);
  const fetchFiles = useCallback(
    async () => {
      const newFiles = await getExperimentFileTree({ experimentId });

      if (isEqual(newFiles, files)) return;

      setFiles(newFiles);
    },
    [ experimentId, files ],
  );

  const treeClasses = classNames({
    [ css.hideElement ]:
    (documentWidth <= 1024) && (viewMode === 'editor'),
  });
  const editorClasses = classNames({
    [ css.hideElement ]:
    (documentWidth <= 1024) && (viewMode === 'tree'),
  });

  // map the file tree
  useEffect(() => {
    fetchFiles();
    // TODO: have error handling added.
  }, [ fetchFiles ]);

  useEffect(() => { // when a file is picked, if on mobile, change the view
    if (documentWidth <= 1024) {
      setViewMode('editor');
    }
  }, [ fileInfo, documentWidth ]);

  const onSelectFile = async (
    keys: React.Key[],
    info: { [key:string]: unknown, node: DataNode },
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
      } catch (error) {
        setIsFetching(false);
        // TODO: have error handling added.
      }
    }
  };

  const editorLanguageSyntax = useMemo(() => {
    const fileExt = (fileInfo?.path || '').split('.')[1];

    if (fileExt === 'md') {
      return 'markdown';
    }

    if (fileExt === 'ts') {
      return 'typescript';
    }

    if (fileExt === 'py') {
      return 'python';
    }

    return fileExt || 'yaml'; // returns yaml as default, in case that it's a checkpoint file
  }, [ fileInfo ]);

  return (
    <section className={css.base}>
      <Section className={treeClasses} id="fileTree">
        <DirectoryTree
          className={css.fileTree}
          data-testid="fileTree"
          defaultExpandAll
          defaultSelectedKeys={(config && documentWidth > 1024) ? [ '0-0' ] : undefined}
          treeData={fileTree}
          onSelect={onSelectFile}
        />
      </Section>
      {
        !!fileInfo?.path && (
          <Spinner className={editorClasses} spinning={isFetching}>
            <section className={css.fileDir}>
              <div className={css.fileInfo}>
                <div className={css.buttonContainer}>
                  {
                    documentWidth <= 1024 && (
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
        className={editorClasses}
        id="editor"
        maxHeight>
        <Spinner spinning={isFetching}>
          {
            !isFetching && !fileInfo?.data
              ? <h5>Please, choose a file to preview.</h5>
              : (
                <MonacoEditor
                  height="100%"
                  language={editorLanguageSyntax}
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
