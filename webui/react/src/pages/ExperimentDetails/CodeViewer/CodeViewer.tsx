import { Tree } from 'antd';
import { DataNode } from 'antd/lib/tree';
import yaml from 'js-yaml';
import React, { useEffect, useState } from 'react';

import MonacoEditor from 'components/MonacoEditor';
import Section from 'components/Section';
import Spinner from 'shared/components/Spinner';
import { ExperimentBase } from 'types';

const { DirectoryTree } = Tree;

import css from './CodeViewer.module.scss';
import './index.scss';

type Props = {
  experiment: ExperimentBase;
}

type FileNode = {
  ContentLength: number;
  ContentType?: string;
  IsDir: boolean;
  ModifiedTime: string;
  Path: string;
  files?: FileNode[];
}

/**
 * the following mocked const is assuming this data structure from the backend
 */

const backendResponse: { files: FileNode[] } = {
  files: [
    {
      ContentLength: 0,
      files: [
        {
          ContentLength: 434,
          ContentType: 'text/plain; charset=utf-8',
          IsDir: false,
          ModifiedTime: '2022-01-03 18:58:09 -0600 CST',
          Path: 'example_folder1/file_a.yaml',
        },
        {
          ContentLength: 434,
          ContentType: 'text/plain; charset=utf-8',
          IsDir: false,
          ModifiedTime: '2022-01-03 18:58:09 -0600 CST',
          Path: 'example_folder1/file_b.yaml',
        },
        {
          ContentLength: 434,
          ContentType: 'text/plain; charset=utf-8',
          IsDir: false,
          ModifiedTime: '2022-01-03 18:58:09 -0600 CST',
          Path: 'example_folder1/file_c.yaml',
        },
      ],
      IsDir: true,
      ModifiedTime: '2022-01-03 18:58:09 -0600 CST',
      Path: 'example_folder1',
    },
    {
      ContentLength: 0,
      files: [
        {
          ContentLength: 0,
          files: [
            {
              ContentLength: 434,
              ContentType: 'text/plain; charset=utf-8',
              IsDir: false,
              ModifiedTime: '2022-01-03 18:58:09 -0600 CST',
              Path: 'example_folder2/example_folder3/file_d.yaml',
            },
            {
              ContentLength: 434,
              ContentType: 'text/plain; charset=utf-8',
              IsDir: false,
              ModifiedTime: '2022-01-03 18:58:09 -0600 CST',
              Path: 'example_folder2/example_folder3/file_e.yaml',
            },
          ],
          IsDir: true,
          ModifiedTime: '2022-01-03 18:58:09 -0600 CST',
          Path: 'example_folder2/example_folder3',
        },
        {
          ContentLength: 434,
          ContentType: 'text/plain; charset=utf-8',
          IsDir: false,
          ModifiedTime: '2022-01-03 18:58:09 -0600 CST',
          Path: 'example_folder2/file_f.yaml',
        },
        {
          ContentLength: 434,
          ContentType: 'text/plain; charset=utf-8',
          IsDir: false,
          ModifiedTime: '2022-01-03 18:58:09 -0600 CST',
          Path: 'example_folder2/file_g.yaml',
        },
      ],
      IsDir: true,
      ModifiedTime: '2022-01-03 18:58:09 -0600 CST',
      Path: 'example_folder2',
    },
    {
      ContentLength: 434,
      ContentType: 'text/plain; charset=utf-8',
      IsDir: false,
      ModifiedTime: '2022-01-03 18:58:09 -0600 CST',
      Path: 'file_h.yaml',
    },
    {
      ContentLength: 434,
      ContentType: 'text/plain; charset=utf-8',
      IsDir: false,
      ModifiedTime: '2022-01-03 18:58:09 -0600 CST',
      Path: 'file_i.yaml',
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
  const navigateTree = (node: FileNode, key: string) => {
    treeMap.set(key, node.Path);

    if (node.files) {
      node.files.forEach((chNode, idx) => navigateTree(chNode, `${key}-${idx}`));
    }
  };

  useEffect(() => {
    if (experiment.configRaw) {
      getFile();
    }

    return () => {
      setPublicConfig({});
    };
  }, []);

  useEffect(() => {
    backendResponse.files.forEach((node, idx) => navigateTree(node, `0-${idx}`));
  }, []);

  // eslint-disable-next-line @typescript-eslint/ban-types
  const onSelectFile = (keys: React.Key[]) => {
    // TODO: after backend integration, check data structure and create implementation
    // to navigate it, check if we need the second "info" arg..
    const filePath = treeMap.get(String(keys[0])) as string;

    if (filePath.includes('.')) { // check if the selected node is a file
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
      <Section>
        <DirectoryTree
          className={css.fileTree}
          defaultExpandAll
          treeData={treeData}
          onSelect={onSelectFile}
        />
      </Section>
      <Section bodyNoPadding bodyScroll maxHeight>
        <Spinner spinning={isFetching}>
          <MonacoEditor
            height="100%"
            options={{
              occurrencesHighlight: false,
              readOnly: true,
            }}
            value={yaml.dump(publicConfig)}
          />
        </Spinner>
      </Section>
    </section>
  );
};

export default CodeViewer;
