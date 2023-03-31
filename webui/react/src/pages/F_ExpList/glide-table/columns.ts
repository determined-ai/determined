import {
  CompactSelection,
  GridCell,
  GridCellKind,
  SizedGridColumn,
} from '@glideapps/glide-data-grid';
import { NavigateFunction } from 'react-router-dom';

import { paths } from 'routes/utils';
import { getColor, getInitials } from 'shared/components/Avatar';
import { DarkLight } from 'shared/themes';
import { humanReadableBytes } from 'shared/utils/string';
import { getStateColorCssVar } from 'themes';
import { DetailedUser, ExperimentItem } from 'types';
import { Loadable } from 'utils/loadable';
import { getDisplayName } from 'utils/user';

import { getDurationInEnglish, getTimeInEnglish } from './utils';

const experimentColumns = [
  'archived',
  'checkpointCount',
  'checkpointSize',
  'description',
  'duration',
  'forkedFrom',
  'id',
  'name',
  'progress',
  'resourcePool',
  'searcherType',
  'searcherMetricValue',
  'selected',
  'startTime',
  'state',
  'tags',
  'numTrials',
  'user',
] as const;

export type ExperimentColumn = (typeof experimentColumns)[number];

export const defaultExperimentColumns: ExperimentColumn[] = [
  'id',
  'description',
  'tags',
  'forkedFrom',
  'progress',
  'startTime',
  'state',
  'searcherType',
  'user',
  'duration',
  'numTrials',
  'resourcePool',
  'checkpointSize',
  'checkpointCount',
  'searcherMetricValue',
];

export type ColumnDef = SizedGridColumn & {
  id: ExperimentColumn;
  isNumerical?: boolean;
  renderer: (record: ExperimentItem, idx: number) => GridCell;
};
interface Params {
  columnWidths: Record<ExperimentColumn, number>;
  navigate: NavigateFunction;
  bodyStyles: CSSStyleDeclaration;
  rowSelection: CompactSelection;
  darkLight: DarkLight;
  users: Loadable<DetailedUser[]>;
  selectAll: boolean;
}
export const getColumnDefs = ({
  columnWidths,
  navigate,
  bodyStyles,
  rowSelection,
  darkLight,
  users,
  selectAll,
}: Params): Record<ExperimentColumn, ColumnDef> => ({
  archived: {
    id: 'archived',
    renderer: (record: ExperimentItem) => ({
      allowOverlay: false,
      data: String(record.archived),
      displayData: record.archived ? '📦' : '',
      kind: GridCellKind.Text,
    }),
    title: 'Archived',
    width: columnWidths.archived,
  },
  checkpointCount: {
    id: 'checkpointCount',
    isNumerical: true,
    renderer: (record: ExperimentItem) => ({
      allowOverlay: false,
      data: Number(record.checkpointCount),
      displayData: String(record.checkpointCount),
      kind: GridCellKind.Number,
    }),
    title: 'Checkpoint Count',
    width: columnWidths.checkpointCount,
  },
  checkpointSize: {
    id: 'checkpointSize',
    isNumerical: true,
    renderer: (record: ExperimentItem) => ({
      allowOverlay: false,
      data: record.checkpointSize ? humanReadableBytes(record.checkpointSize) : '',
      displayData: record.checkpointSize ? humanReadableBytes(record.checkpointSize) : '',
      kind: GridCellKind.Text,
    }),
    title: 'Checkpoint Size',
    width: columnWidths.checkpointSize,
  },
  description: {
    id: 'description',
    renderer: (record: ExperimentItem) => ({
      allowOverlay: false,
      data: String(record.description),
      displayData: String(record.description),
      kind: GridCellKind.Text,
    }),
    title: 'Description',
    width: columnWidths.description,
  },
  duration: {
    id: 'duration',
    isNumerical: true,
    renderer: (record: ExperimentItem) => ({
      allowOverlay: false,
      data: `${getDurationInEnglish(record)}`,
      displayData: `${getDurationInEnglish(record)}`,
      kind: GridCellKind.Text,
    }),
    title: 'Duration',
    width: columnWidths.duration,
  },
  forkedFrom: {
    id: 'forkedFrom',
    renderer: (record: ExperimentItem) => ({
      allowOverlay: false,
      copyData: String(record.forkedFrom ?? ''),
      data: {
        kind: 'links-cell',
        links:
          record.forkedFrom !== undefined
            ? [
                {
                  onClick: () =>
                    record.forkedFrom && navigate(paths.experimentDetails(record.forkedFrom)),
                  title: String(record.forkedFrom ?? ''),
                },
              ]
            : [],
        navigateOn: 'click',
        underlineOffset: 6,
      },
      kind: GridCellKind.Custom,
      readonly: true,
    }),
    title: 'Forked From',
    width: columnWidths.forkedFrom,
  },
  id: {
    id: 'id',
    renderer: (record: ExperimentItem) => ({
      allowOverlay: false,
      copyData: String(record.id),
      data: {
        kind: 'links-cell',
        links: [
          {
            onClick: () => navigate(paths.experimentDetails(record.id)),
            title: String(record.id),
          },
        ],
        navigateOn: 'click',
        underlineOffset: 6,
      },
      kind: GridCellKind.Custom,
      readonly: true,
    }),
    title: 'ID',
    width: columnWidths.id,
  },
  name: {
    id: 'name',
    renderer: (record: ExperimentItem) => ({
      allowOverlay: false,
      copyData: String(record.name),
      data: {
        kind: 'links-cell',
        links: [
          {
            onClick: () => navigate(paths.experimentDetails(record.id)),
            title: String(record.name),
          },
        ],
        navigateOn: 'click',
        underlineOffset: 6,
      },
      kind: GridCellKind.Custom,
      readonly: true,
    }),
    themeOverride: { horizontalBorderColor: '#225588' },
    title: 'Name',
    width: columnWidths.name,
  },
  numTrials: {
    id: 'numTrials',
    isNumerical: true,
    renderer: (record: ExperimentItem) => ({
      allowOverlay: false,
      data: record.numTrials,
      displayData: String(record.numTrials),
      kind: GridCellKind.Number,
    }),
    title: 'Trials',
    width: columnWidths.numTrials,
  },
  progress: {
    id: 'progress',
    renderer: (record: ExperimentItem) => {
      return (record.progress ?? 0) > 0
        ? {
            allowOverlay: false,
            copyData: String(record.progress ?? 0),
            data: {
              color: bodyStyles.getPropertyValue(getStateColorCssVar(record.state).slice(4, -1)),
              kind: 'range-cell',
              max: 1,
              min: 0,
              step: 1,
              value: record.progress ?? 0,
            },
            kind: GridCellKind.Custom,
          }
        : {
            allowOverlay: false,
            data: '',
            displayData: '',
            kind: GridCellKind.Text,
          };
    },
    title: 'Progress',
    width: columnWidths.progress,
  },
  resourcePool: {
    id: 'resourcePool',
    renderer: (record: ExperimentItem) => ({
      allowOverlay: false,
      data: String(record.resourcePool),
      displayData: String(record.resourcePool),
      kind: GridCellKind.Text,
    }),
    title: 'Resource Pool',
    width: columnWidths.resourcePool,
  },
  searcherMetricValue: {
    id: 'searcherMetricValue',
    isNumerical: true,
    renderer: (record: ExperimentItem) => ({
      allowOverlay: false,
      data: String(record.searcherMetricValue ?? ''),
      displayData: String(record.searcherMetricValue ?? ''),
      kind: GridCellKind.Text,
    }),
    title: 'Searcher Metric Values',
    width: columnWidths.searcherMetricValue,
  },
  searcherType: {
    id: 'searcherType',
    renderer: (record: ExperimentItem) => ({
      allowOverlay: false,
      data: String(record.searcherType),
      displayData: String(record.searcherType),
      kind: GridCellKind.Text,
    }),
    title: 'Searcher Type',
    width: columnWidths.searcherType,
  },
  selected: {
    icon: 'selected',
    id: 'selected',
    renderer: (record: ExperimentItem, idx) => ({
      allowOverlay: false,
      contentAlign: 'left',
      data: selectAll || rowSelection.hasIndex(idx),
      kind: GridCellKind.Boolean,
    }),
    themeOverride: { cellHorizontalPadding: 13, headerIconSize: 30 },
    title: '',
    width: columnWidths.selected,
  },
  startTime: {
    id: 'startTime',
    isNumerical: true,
    renderer: (record: ExperimentItem) => ({
      allowOverlay: false,
      data: getTimeInEnglish(new Date(record.startTime)),
      displayData: getTimeInEnglish(new Date(record.startTime)),
      kind: GridCellKind.Text,
    }),
    title: 'Start Time',
    width: columnWidths.startTime,
  },
  state: {
    id: 'state',
    renderer: (record: ExperimentItem) => ({
      allowAdd: false,
      allowOverlay: true,
      copyData: record.state.toLocaleLowerCase(),
      data: [],
      kind: GridCellKind.Image,
    }),
    title: 'State',
    width: columnWidths.state,
  },
  tags: {
    id: 'tags',
    renderer: (record: ExperimentItem) => ({
      allowOverlay: true,
      copyData: record['labels'].join(', '),
      data: {
        kind: 'tags-cell',
        possibleTags: [],
        readonly: true,
        tags: record['labels'],
      },
      kind: GridCellKind.Custom,
    }),
    title: 'Tags',
    width: columnWidths.tags,
  },
  user: {
    id: 'user',
    renderer: (record: ExperimentItem) => {
      const displayName = Loadable.match(users, {
        Loaded: (users) => getDisplayName(users?.find((u) => u.id === record.userId)),
        NotLoaded: () => undefined,
      });
      return {
        allowOverlay: true,
        copyData: String(record.userId),
        data: {
          image: undefined,
          initials: getInitials(displayName),
          kind: 'user-profile-cell',
          tint: getColor(displayName, darkLight),
        },
        kind: GridCellKind.Custom,
      };
    },

    title: 'User',
    width: columnWidths.user,
  },
});

export const defaultColumnWidths: Record<ExperimentColumn, number> = {
  archived: 80,
  checkpointCount: 74,
  checkpointSize: 74,
  description: 148,
  duration: 96,
  forkedFrom: 128,
  id: 50,
  name: 150,
  numTrials: 74,
  progress: 111,
  resourcePool: 140,
  searcherMetricValue: 74,
  searcherType: 140,
  selected: 45,
  startTime: 118,
  state: 106,
  tags: 106,
  user: 85,
};
