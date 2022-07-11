import { FileOutlined } from '@ant-design/icons';
import { Tree } from 'antd';
import Button from 'antd/es/button';
import { DataNode } from 'antd/lib/tree';
import yaml from 'js-yaml';
import React, { useEffect, useState } from 'react';

import MonacoEditor from 'components/MonacoEditor';
import Section from 'components/Section';
import { getExperimentFileTree } from 'services/api';
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

/**
 * the following mocked const is assuming this data structure from the backend
 */

const backendResponse: { files: FileNode[] } = {
  files: [
    {
      content_length: 0,
      files: [
        {
          content_length: 434,
          content_type: 'text/plain; charset=utf-8',
          is_dir: false,
          modified_time: '2022-01-03 18:58:09 -0600 CST',
          name: 'file_a.yaml',
          path: 'example_folder1/file_a.yaml',
        },
        {
          content_length: 434,
          content_type: 'text/plain; charset=utf-8',
          is_dir: false,
          modified_time: '2022-01-03 18:58:09 -0600 CST',
          name: 'file_b.yaml',
          path: 'example_folder1/file_b.yaml',
        },
        {
          content_length: 434,
          content_type: 'text/plain; charset=utf-8',
          is_dir: false,
          modified_time: '2022-01-03 18:58:09 -0600 CST',
          name: 'file_c.yaml',
          path: 'example_folder1/file_c.yaml',
        },
      ],
      is_dir: true,
      modified_time: '2022-01-03 18:58:09 -0600 CST',
      name: 'example_folder1',
      path: 'example_folder1',
    },
    {
      content_length: 0,
      files: [
        {
          content_length: 0,
          files: [
            {
              content_length: 434,
              content_type: 'text/plain; charset=utf-8',
              is_dir: false,
              modified_time: '2022-01-03 18:58:09 -0600 CST',
              name: 'file_d.yaml',
              path: 'example_folder2/example_folder3/file_d.yaml',
            },
            {
              content_length: 434,
              content_type: 'text/plain; charset=utf-8',
              is_dir: false,
              modified_time: '2022-01-03 18:58:09 -0600 CST',
              name: 'file_e.yaml',
              path: 'example_folder2/example_folder3/file_e.yaml',
            },
          ],
          is_dir: true,
          modified_time: '2022-01-03 18:58:09 -0600 CST',
          name: '',
          path: 'example_folder2/example_folder3',
        },
        {
          content_length: 434,
          content_type: 'text/plain; charset=utf-8',
          is_dir: false,
          modified_time: '2022-01-03 18:58:09 -0600 CST',
          name: 'file_f.yaml',
          path: 'example_folder2/file_f.yaml',
        },
        {
          content_length: 434,
          content_type: 'text/plain; charset=utf-8',
          is_dir: false,
          modified_time: '2022-01-03 18:58:09 -0600 CST',
          name: 'file_g.yaml',
          path: 'example_folder2/file_g.yaml',
        },
      ],
      is_dir: true,
      modified_time: '2022-01-03 18:58:09 -0600 CST',
      name: 'example_folder2',
      path: 'example_folder2',
    },
    {
      content_length: 434,
      content_type: 'text/plain; charset=utf-8',
      is_dir: false,
      modified_time: '2022-01-03 18:58:09 -0600 CST',
      name: 'file_h.yaml',
      path: 'file_h.yaml',
    },
    {
      content_length: 434,
      content_type: 'text/plain; charset=utf-8',
      is_dir: false,
      modified_time: '2022-01-03 18:58:09 -0600 CST',
      name: 'file_i.yaml',
      path: 'file_i.yaml',
    },
  ],
};

const treeData: DataNode[] = [ // TODO: this has to go after integration
  {
    children: [
      {
        isLeaf: true,
        key: '0-0-0',
        title: 'file_a.yaml',
      },
      {
        isLeaf: true,
        key: '0-0-1',
        title: 'file_b.yaml',
      },
      {
        isLeaf: true,
        key: '0-0-2',
        title: 'file_c.yaml',
      },
    ],
    key: '0-0', // following the type DataNode where the first 0 is the root and 0 is the node
    title: 'example_folder1',
  },
  {
    children: [
      {
        children: [
          {
            isLeaf: true,
            key: '0-1-0-0',
            title: 'file_d.yaml',
          },
          {
            isLeaf: true,
            key: '0-1-0-1',
            title: 'file_e.yaml',
          },
        ],
        isLeaf: false,
        key: '0-1-0',
        title: 'example_folder3',
      },
      {
        isLeaf: true,
        key: '0-1-1',
        title: 'file_f.yaml',
      },
      {
        isLeaf: true,
        key: '0-1-2',
        title: 'file_g.yaml',
      },
    ],
    key: '0-1', // following the type DataNode where the first 0 is the root and 1 is the node
    title: 'example_folder2',
  },
  {
    isLeaf: true,
    key: '0-2',
    title: 'file_h.yaml',
  },
  {
    isLeaf: true,
    key: '0-3',
    title: 'file_i.yaml',
  },
];

const CodeViewer: React.FC<Props> = ({ experiment }) => {
  // TODO: remove this after integration (taken from the config tab)
  const [ publicConfig, setPublicConfig ] = useState({});
  const [ treeMap ] = useState(() => new Map<string, string>());
  const [ isFetching, setIsFetching ] = useState(false);
  const [ fileDir, setFileDir ] = useState('');
  const [ fileName, setFileName ] = useState('');

  // TODO: after integration, create a proper fn for data fetching.
  const getFile = () => {
    const {
      environment: { registry_auth, ...restEnvironment },
      workspace,
      project,
      ...restConfig
    } = experiment.configRaw;
    setPublicConfig({ environment: restEnvironment, ...restConfig });
  };

  useEffect(() => {
    const navigateTree = (node: FileNode, key: string) => {
      treeMap.set(key, node.path);

      if (node.files) {
        node.files.forEach((chNode, idx) => navigateTree(chNode, `${key}-${idx}`));
      }
    };

    backendResponse.files.forEach((node, idx) => navigateTree(node, `0-${idx}`));

    // (async () => {
    //   const foo = await getExperimentFileTree({ experimentId: experiment.id });
    //   console.log('file tree', foo);
    // })();
  }, [ treeMap ]);

  // eslint-disable-next-line @typescript-eslint/ban-types
  const onSelectFile = (keys: React.Key[], info: { [key:string]: unknown, node: DataNode }) => {
    // TODO: after backend integration, check data structure and create implementation
    // to navigate it
    const filePath = treeMap.get(String(keys[0])) as string;

    if (filePath.includes('.')) { // check if the selected node is a file
      setFileDir(filePath);
      setFileName(info.node.title as string);
      setIsFetching(true);
      setPublicConfig({});

      setTimeout(() => {
        getFile();

        setIsFetching(false);
      }, 1500);
    }
  };

  return (
    <section className={css.base}>
      <Section id="fileTree">
        <DirectoryTree
          className={css.fileTree}
          defaultExpandAll
          treeData={treeData}
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
            !isFetching && !Object.keys(publicConfig).length
              ? <h5>Please, choose a file to preview.</h5>
              : (
                <MonacoEditor
                  height="100%"
                  language="yaml"
                  options={{
                    minimap: {
                      enabled: !!Object.keys(publicConfig).length,
                      showSlider: 'mouseover',
                      size: 'fit',
                    },
                    occurrencesHighlight: false,
                    readOnly: true,
                  }}
                  value={yaml.dump(publicConfig)}
                />
              )
          }
        </Spinner>
      </Section>
    </section>
  );
};

export default CodeViewer;
