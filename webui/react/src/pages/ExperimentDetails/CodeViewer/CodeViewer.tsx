import { FileOutlined } from '@ant-design/icons';
import { Tree } from 'antd';
import Button from 'antd/es/button';
import { DataNode } from 'antd/lib/tree';
import yaml from 'js-yaml';
import React, { useEffect, useState } from 'react';

import MonacoEditor from 'components/MonacoEditor';
import Section from 'components/Section';
import { getExperimentFileFromTree, getExperimentFileTree } from 'services/api';
import { FileNode } from 'services/api-ts-sdk';
import Icon from 'shared/components/Icon';
import Spinner from 'shared/components/Spinner';
import { ExperimentBase } from 'types';

const { DirectoryTree } = Tree;

import css from './CodeViewer.module.scss';
import './index.scss';

type Props = {
  experiment: ExperimentBase;
}

const CodeViewer: React.FC<Props> = ({ experiment }) => {
  const [ fileData, setFileData ] = useState<string>();
  const [ files, setFiles ] = useState<FileNode[]>([]);
  const [ fileTree, setFileTree ] = useState<DataNode[]>([]);
  const [ treeMap ] = useState(() => new Map<string, string>());
  const [ isFetching, setIsFetching ] = useState(false);
  const [ fileDir, setFileDir ] = useState('');
  const [ fileName, setFileName ] = useState('');

  // get the file tree from backend
  useEffect(() => {
    (async () => {
      const files = await getExperimentFileTree({ experimentId: experiment.id });

      setFiles(files);
    })();
    return () => {
      setFiles([]);
    };
  }, [ experiment.id ]);

  // map the file tree
  useEffect(() => {
    const navigateTree = (node: FileNode, key: string): DataNode => {
      treeMap.set(key, node.path);

      const newNode: DataNode = {
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

    setFileTree(files.map<DataNode>((node, idx) => navigateTree(node, `0-${idx}`)));
  }, [ treeMap, files ]);

  // eslint-disable-next-line @typescript-eslint/ban-types
  const onSelectFile = async (
    keys: React.Key[],
    info: { [key:string]: unknown, node: DataNode },
  ) => {
    // TODO: after backend integration, check data structure and create implementation
    // to navigate it
    const filePath = treeMap.get(String(keys[0])) as string;

    if (filePath.includes('.')) { // check if the selected node is a file
      setIsFetching(true);

      try {
        const file = await getExperimentFileFromTree({ experimentId: experiment.id, filePath });

        setIsFetching(false);
        setFileData(file);
        setFileDir(filePath);
        setFileName(info.node.title as string);
      } catch (error) {
        setIsFetching(false);

        throw new Error(error as string);
      }
    }
  };

  return (
    <section className={css.base}>
      <Section id="fileTree">
        <DirectoryTree
          className={css.fileTree}
          defaultExpandAll
          treeData={fileTree}
          onSelect={onSelectFile}
        />
      </Section>
      {
        !!fileDir && (
          <Spinner spinning={isFetching}>
            <section className={css.fileDir}>
              <div className={css.fileInfo}>
                <div>
                  <FileOutlined />
                  <span className={css.filePath}>{fileName}</span>
                </div>
                <div className={css.buttonsContainer}>
                  <Button className={css.noBorderButton}>Open in Notebook</Button>
                  <Button
                    className={css.noBorderButton}
                    ghost
                    icon={<Icon name="overflow-vertical" />}
                  />
                </div>
              </div>
            </section>
          </Spinner>
        )
      }
      <Section bodyNoPadding bodyScroll id="editor" maxHeight>
        <Spinner spinning={isFetching}>
          {
            !isFetching && !fileData
              ? <h5>Please, choose a file to preview.</h5>
              : (
                <MonacoEditor
                  height="100%"
                  language="yaml"
                  options={{
                    minimap: {
                      enabled: !!fileData?.length,
                      showSlider: 'mouseover',
                      size: 'fit',
                    },
                    occurrencesHighlight: false,
                    readOnly: true,
                  }}
                  value={yaml.dump(fileData)}
                />
              )
          }
        </Spinner>
      </Section>
    </section>
  );
};

export default CodeViewer;
