/* eslint-disable max-len */
import { DownloadOutlined, FileOutlined, LeftOutlined } from '@ant-design/icons';
import { Tooltip, Tree } from 'antd';
import { DataNode } from 'antd/lib/tree';
import classNames from 'classnames';
import yaml from 'js-yaml';
import React, { useEffect, useState } from 'react';

import MonacoEditor from 'components/MonacoEditor';
import Section from 'components/Section';
import useRecize from 'hooks/useResize';
import { handlePath } from 'routes/utils';
import { getExperimentFileFromTree, getExperimentFileTree } from 'services/api';
import { FileNode } from 'services/api-ts-sdk';
import Spinner from 'shared/components/Spinner';
import { RawJson } from 'shared/types';

const { DirectoryTree } = Tree;

import css from './CodeViewer.module.scss';
import './index.scss';

export type Props = {
  configRaw?: RawJson;
  experimentId: number;
}

type FileInfo = {
  name: string;
  path: string;
  data: string;
};

/**
 * A component responsible to enable the user to view the code for a experiment.
 * It renders a file tree and a selected file in the MonacoEditor
 * Props:
 * experimentID: the experiment ID;
 * configRaw: the experiment.configRaw property to be used to render a Config yaml file;
 *
 * Original ticket DET-7466
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
  const [ fileTree, setFileTree ] = useState<DataNode[]>([]); // Data structure to be used by the Tree
  const [ treeMap ] = useState(() => new Map<string, string>()); // Map structure from the API
  const [ isFetching, setIsFetching ] = useState(false);
  const [ fileInfo, setFileInfo ] = useState<FileInfo>();
  const [ viewMode, setViewMode ] = useState<'tree' | 'editor' | undefined>(
    () => documentWidth <= 1024 ? 'tree' : undefined,
  ); // To be used in the mobile view, switches the UI

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
    try {
      (async () => {
        const files = await getExperimentFileTree({ experimentId });

        const navigateTree = (node: FileNode, key: string): DataNode => {
          treeMap.set(key, node.path);

          const newNode: DataNode = {
            className: 'treeNode',
            isLeaf: true,
            key,
            title: node.name,
          };

          if (node.files?.length) {
            newNode.children = node.files.map((chNode, idx) => navigateTree(chNode, `${key}-${idx}`));
            newNode.isLeaf = false;
          }

          return newNode;
        };

        if (config) {
          setFileTree([
            {
              className: 'treeNode',
              isLeaf: true,
              key: '0-0',
              title: 'Configuration',
            },
            ...files.map<DataNode>((node, idx) => navigateTree(node, `0-${idx + 1}`)),
          ]);

          setFileInfo({
            name: 'Configuration',
            path: 'Configuration',
            data: yaml.dump(config)
          });

          if (documentWidth <= 1024) { // if it's in mobile view and we have a config file available, render it as default
            setViewMode('editor');
          }
        } else {
          setFileTree(files.map<DataNode>((node, idx) => navigateTree(node, `0-${idx}`)));
        }
      })();
    } catch (error) {
      throw new Error(error as string);
    }
  }, [ treeMap, config, documentWidth, experimentId ]);

  const onSelectFile = async (
    keys: React.Key[],
    info: { [key:string]: unknown, node: DataNode },
  ) => {
    if (info.node.title === 'Configuration') {
      setFileInfo({
        name: 'Configuration',
        path: 'Configuration',
        data: yaml.dump(config)
      });

      return;
    }

    const filePath = treeMap.get(String(keys[0])) as string;

    if (filePath.includes('.')) { // check if the selected node is a file
      setIsFetching(true);

      try {
        const file = await getExperimentFileFromTree({ experimentId, filePath });

        setIsFetching(false);
        setFileInfo({
          name: info.node.title as string,
          path: filePath,
          data: decodeURIComponent(escape(window.atob(file)))
        });

        if (documentWidth <= 1024) {
          setViewMode('editor');
        }
      } catch (error) {
        setIsFetching(false);

        throw new Error(error as string);
      }
    }
  };

  const setEditorLanguageSyntax = () => {
    const fileExt = (fileInfo?.path || '').split('.')[1];

    if (fileExt === 'md') {
      return 'markdown';
    }

    if (fileExt === 'ts') {
      return 'typescript';
    }

    return fileExt;
  };

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
                          onClick={e => handlePath(e, {
                            path: `/experiments/${experimentId}/file/download?path=${fileInfo.path}`,
                            external: true
                          })}/>
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
                  language={setEditorLanguageSyntax()}
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
