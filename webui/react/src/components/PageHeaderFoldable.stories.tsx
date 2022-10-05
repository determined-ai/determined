import { ComponentStory, Meta } from '@storybook/react';
import React from 'react';

import Icon from 'shared/components/Icon';

import PageHeaderFoldable, { Option } from './PageHeaderFoldable';

const options: Option[] = [
  {
    key: 'unarchive',
    label: 'Unarchive',
    onClick: () => undefined,
  },
  {
    key: 'continue-trial',
    label: 'Continue Trial',
    onClick: () => undefined,
  },
  {
    icon: <Icon name="fork" size="small" />,
    key: 'delete',
    label: 'Delete',
    onClick: () => undefined,
  },
  {
    key: 'hyperparameter-search',
    label: 'Hyperparameter Search',
    onClick: () => undefined,
  },
  {
    icon: <Icon name="download" size="small" />,
    key: 'download-model',
    label: 'Download Experiment Code',
    onClick: () => undefined,
  },
  {
    icon: <Icon name="fork" size="small" />,
    key: 'fork',
    label: 'Fork',
    onClick: () => undefined,
  },
  {
    key: 'move',
    label: 'Move',
    onClick: () => undefined,
  },
  {
    icon: <Icon name="tensor-board" size="small" />,
    key: 'tensorboard',
    label: 'TensorBoard',
    onClick: () => undefined,
  },
  {
    key: 'archive',
    label: 'Archive',
    onClick: () => undefined,
  },
];

export default {
  component: PageHeaderFoldable,
  title: 'Determined/PageHeaderFoldable',
} as Meta<typeof PageHeaderFoldable>;

export const Default: ComponentStory<typeof PageHeaderFoldable> = () => (
  <div style={{ width: 600 }}>
    <PageHeaderFoldable
      foldableContent={<div>Foldable Content</div>}
      leftContent={<div>Left content</div>}
      options={options}
    />
  </div>
);
