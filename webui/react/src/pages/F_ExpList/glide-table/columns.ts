import {
  CompactSelection,
  DataEditorProps,
  GridCell,
  GridCellKind,
  SizedGridColumn,
} from '@glideapps/glide-data-grid';
import { NavigateFunction } from 'react-router-dom';

import { terminalRunStates } from 'constants/states';
import { paths } from 'routes/utils';
import { getColor, getInitials } from 'shared/components/Avatar';
import { DarkLight, Theme } from 'shared/themes';
import { humanReadableNumber } from 'shared/utils/number';
import { humanReadableBytes } from 'shared/utils/string';
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
  tooltip: (record: ExperimentItem) => string | undefined;
};

export type ColumnDefs = Record<ExperimentColumn, ColumnDef>;
interface Params {
  appTheme: Theme;
  columnWidths: Record<ExperimentColumn, number>;
  navigate: NavigateFunction;
  rowSelection: CompactSelection;
  darkLight: DarkLight;
  users: Loadable<DetailedUser[]>;
  selectAll: boolean;
}
export const getColumnDefs = ({
  columnWidths,
  navigate,
  rowSelection,
  darkLight,
  users,
  selectAll,
  appTheme,
}: Params): ColumnDefs => ({
  archived: {
    id: 'archived',
    renderer: (record: ExperimentItem) => ({
      allowOverlay: false,
      data: String(record.archived),
      displayData: record.archived ? '📦' : '',
      kind: GridCellKind.Text,
    }),
    title: 'Archived',
    tooltip: () => undefined,
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
    tooltip: () => undefined,
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
    tooltip: () => undefined,
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
    tooltip: () => undefined,
    width: columnWidths.description,
  },
  duration: {
    id: 'duration',
    isNumerical: true,
    renderer: (record: ExperimentItem) => ({
      allowOverlay: false,
      data: getDurationInEnglish(record),
      displayData: getDurationInEnglish(record),
      kind: GridCellKind.Text,
    }),
    title: 'Duration',
    tooltip: () => undefined,
    width: columnWidths.duration,
  },
  forkedFrom: {
    id: 'forkedFrom',
    renderer: (record: ExperimentItem) => ({
      allowOverlay: false,
      copyData: String(record.forkedFrom ?? ''),
      cursor: record.forkedFrom ? 'pointer' : undefined,
      data: {
        kind: 'link-cell',
        link:
          record.forkedFrom !== undefined
            ? {
                onClick: () =>
                  record.forkedFrom && navigate(paths.experimentDetails(record.forkedFrom)),
                title: String(record.forkedFrom ?? ''),
              }
            : undefined,
        navigateOn: 'click',
        underlineOffset: 6,
      },
      kind: GridCellKind.Custom,
      readonly: true,
    }),
    title: 'Forked From',
    tooltip: () => undefined,
    width: columnWidths.forkedFrom,
  },
  id: {
    id: 'id',
    renderer: (record: ExperimentItem) => ({
      allowOverlay: false,
      copyData: String(record.id),
      cursor: 'pointer',
      data: {
        kind: 'link-cell',
        link: {
          onClick: () => navigate(paths.experimentDetails(record.id)),
          title: String(record.id),
        },

        navigateOn: 'click',
        underlineOffset: 6,
      },
      kind: GridCellKind.Custom,
      readonly: true,
    }),
    title: 'ID',
    tooltip: () => undefined,
    width: columnWidths.id,
  },
  name: {
    id: 'name',
    renderer: (record: ExperimentItem) => ({
      allowOverlay: false,
      copyData: String(record.name),
      cursor: 'pointer',
      data: {
        kind: 'link-cell',
        link: {
          onClick: () => navigate(paths.experimentDetails(record.id)),
          title: String(record.name),
        },
        navigateOn: 'click',
        underlineOffset: 6,
      },
      kind: GridCellKind.Custom,
      readonly: true,
    }),
    themeOverride: { horizontalBorderColor: '#225588' },
    title: 'Name',
    tooltip: () => undefined,
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
    tooltip: () => undefined,
    width: columnWidths.numTrials,
  },
  progress: {
    id: 'progress',
    renderer: (record: ExperimentItem) => {
      const progress = [...terminalRunStates.keys()].includes(record.state)
        ? 1
        : record.progress ?? 0;
      const percentage = `${(progress * 100).toFixed()}%`;

      return {
        allowOverlay: false,
        data: percentage,
        displayData: percentage,
        kind: GridCellKind.Text,
      };
    },
    title: 'Progress',
    tooltip: () => undefined,
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
    tooltip: () => undefined,
    width: columnWidths.resourcePool,
  },
  searcherMetricValue: {
    id: 'searcherMetricValue',
    isNumerical: true,
    renderer: (record: ExperimentItem) => ({
      allowOverlay: false,
      data:
        record.searcherMetricValue !== undefined
          ? humanReadableNumber(record.searcherMetricValue)
          : '',
      displayData: String(record.searcherMetricValue ?? ''),
      kind: GridCellKind.Text,
    }),
    title: 'Searcher Metric Values',
    tooltip: () => undefined,
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
    tooltip: () => undefined,
    width: columnWidths.searcherType,
  },
  selected: {
    icon: selectAll ? 'allSelected' : rowSelection.length ? 'someSelected' : 'noneSelected',
    id: 'selected',
    renderer: (_: ExperimentItem, idx) => ({
      allowOverlay: false,
      contentAlign: 'left',
      copyData: String(rowSelection.hasIndex(idx)),
      data: {
        checked: selectAll || rowSelection.hasIndex(idx),
        kind: 'checkbox-cell',
      },
      kind: GridCellKind.Custom,
    }),
    themeOverride: { cellHorizontalPadding: 10 },
    title: '',
    tooltip: () => undefined,
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
    tooltip: () => undefined,
    width: columnWidths.startTime,
  },
  state: {
    id: 'state',
    renderer: (record: ExperimentItem) => ({
      allowAdd: false,
      allowOverlay: true,
      copyData: record.state.toLocaleLowerCase(),
      data: {
        appTheme,
        kind: 'experiment-state-cell',
        state: record.state,
      },
      kind: GridCellKind.Custom,
    }),
    themeOverride: { cellHorizontalPadding: 13 },
    title: 'State',
    tooltip: (record: ExperimentItem) => record.state.toLocaleLowerCase(),
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
    tooltip: () => undefined,
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
    tooltip: (record: ExperimentItem) => {
      const displayName = Loadable.match(users, {
        Loaded: (users) => getDisplayName(users?.find((u) => u.id === record.userId)),
        NotLoaded: () => undefined,
      });
      return displayName;
    },
    width: columnWidths.user,
  },
});

export const defaultColumnWidths: Record<ExperimentColumn, number> = {
  archived: 80,
  checkpointCount: 140,
  checkpointSize: 140,
  description: 148,
  duration: 96,
  forkedFrom: 128,
  id: 50,
  name: 290,
  numTrials: 74,
  progress: 111,
  resourcePool: 140,
  searcherMetricValue: 74,
  searcherType: 140,
  selected: 40,
  startTime: 118,
  state: 106,
  tags: 106,
  user: 85,
};

// TODO: use theme here
// eslint-disable-next-line @typescript-eslint/no-unused-vars
export const getHeaderIcons = (appTheme: Theme): DataEditorProps['headerIcons'] => ({
  allSelected: () => `
    <svg width="16" height="16" viewBox="-1 -1 15 15" fill="none" xmlns="http://www.w3.org/2000/svg">
      <rect x="0.5" y="0.5" width="13" height="13" rx="3" fill="#D9D9D9" fill-opacity="0.05" stroke="#454545"/>
      <line x1="5.25" y1="6.5" x2="6.75" y2="8" stroke="#454545" stroke-width="1.5" stroke-linecap="round"/>
      <line x1="6.75" y1="8" x2="9.25" y2="5.5" stroke="#454545" stroke-width="1.5" stroke-linecap="round"/>
    </svg>
  `,
  noneSelected: () => `
    <svg width="16" height="16" viewBox="-1 -1 15 15" fill="none" xmlns="http://www.w3.org/2000/svg">
      <rect x="0.5" y="0.5" width="13" height="13" rx="3" fill="#D9D9D9" fill-opacity="0.05" stroke="#454545"/>
    </svg>
  `,
  someSelected: () => `
    <svg width="16" height="16" viewBox="-1 -1 15 15" fill="none" xmlns="http://www.w3.org/2000/svg">
      <rect x="0.5" y="0.5" width="13" height="13" rx="3" fill="#D9D9D9" fill-opacity="0.05" stroke="#454545"/>
      <line x1="3" y1="7" x2="11" y2="7" stroke="#929292" stroke-width="2"/>
    </svg>
  `,
});
