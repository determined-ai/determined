import { GridCell, GridCellKind } from '@glideapps/glide-data-grid';

import { ProjectExperiment } from 'types';

export const cell: GridCell = {
  allowOverlay: false,
  copyData: 'core-api-stage-3',
  cursor: 'pointer',
  data: {
    kind: 'link-cell',
    link: {
      href: '/experiments/7261',
      title: 'core-api-stage-3',
      unmanaged: false,
    },
    navigateOn: 'click',
    underlineOffset: 6,
  },
  kind: GridCellKind.Custom,
  readonly: true,
};

export const experiment: ProjectExperiment = {
  archived: false,
  checkpoints: 0,
  checkpointSize: 0,
  description: 'Continuation of trial 49300, experiment 7229',
  duration: 12,
  endTime: '2024-06-27T22:35:00.745298Z',
  forkedFrom: 7229,
  hyperparameters: {
    increment_by: {
      type: 'const',
      val: 1,
    },
    irrelevant1: {
      type: 'const',
      val: 1,
    },
    irrelevant2: {
      type: 'const',
      val: 1,
    },
  },
  id: 7261,
  jobId: '742ae9dc-e712-4348-9b15-a1b9f652d6d5',
  labels: [],
  name: 'core-api-stage-3',
  notes: '',
  numTrials: 1,
  parentArchived: false,
  progress: 0,
  projectId: 1,
  projectName: 'Uncategorized',
  projectOwnerId: 1,
  resourcePool: 'aux-pool',
  searcherType: 'single',
  startTime: '2024-06-27T22:34:49.194301Z',
  state: 'CANCELED',
  trialIds: [],
  unmanaged: false,
  userId: 1288,
  workspaceId: 1,
  workspaceName: 'Uncategorized',
};
