import { Tree } from 'antd';
import { Key } from 'antd/lib/table/interface';
import { DataNode } from 'antd/lib/tree';
import yaml from 'js-yaml';
import React from 'react';

import MonacoEditor from 'components/MonacoEditor';
import Section from 'components/Section';
import { ExperimentBase } from 'types';

const { DirectoryTree } = Tree;

import css from './CodeViewer.module.scss';
import './index.scss';

type Props = {
  experiment: ExperimentBase;
}

const treeData: DataNode[] = [
  {
    children: [
      {
        isLeaf: true,
        key: '0-0-0',
        title: 'leaf 0-0',
      },
      {
        isLeaf: true,
        key: '0-0-1',
        title: 'leaf 0-1',
      },
    ],
    key: '0-0',
    title: 'parent 0',
  },
  {
    children: [
      {
        isLeaf: true,
        key: '0-1-0',
        title: 'leaf 1-0',
      },
      {
        isLeaf: true,
        key: '0-1-1',
        title: 'leaf 1-1',
      },
    ],
    key: '0-1',
    title: 'parent 1',
  },
];

const CodeViewer: React.FC<Props> = ({ experiment }) => {
  let publicConfig = {};
  if (experiment.configRaw) {
    const {
      environment: { registry_auth, ...restEnvironment },
      workspace,
      project,
      ...restConfig
    } = experiment.configRaw;
    publicConfig = { environment: restEnvironment, ...restConfig };
  }

  // eslint-disable-next-line @typescript-eslint/ban-types
  const onSelect = (keys: Key[], info: object) => {
    // eslint-disable-next-line no-console
    console.log('Trigger Select', keys, info);
  };

  return (
    <section className={css.base}>
      <Section>
        <DirectoryTree
          className={css.fileTree}
          defaultExpandAll
          treeData={treeData}
          onSelect={onSelect}
        />
      </Section>
      <Section bodyNoPadding bodyScroll maxHeight>
        <MonacoEditor
          height="100%"
          options={{
            occurrencesHighlight: false,
            readOnly: true,
          }}
          value={yaml.dump(publicConfig)}
        />
      </Section>
    </section>
  );
};

export default CodeViewer;
