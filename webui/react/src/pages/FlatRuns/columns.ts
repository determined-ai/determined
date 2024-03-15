import {
  CompactSelection,
  DataEditorProps,
  GridCell,
  GridCellKind,
  Theme as GTheme,
  SizedGridColumn,
} from '@glideapps/glide-data-grid';
import { getColor, getInitials } from 'hew/Avatar';
import { Theme } from 'hew/Theme';
import { Loadable } from 'hew/utils/loadable';

import { getTimeInEnglish } from 'pages/F_ExpList/glide-table/utils';
import { paths } from 'routes/utils';
import { DetailedUser, FlatRun, ProjectColumn } from 'types';
import { getPath, isString } from 'utils/data';
import { DURATION_UNIT_MEASURES, durationInEnglish, formatDatetime } from 'utils/datetime';
import { humanReadableNumber } from 'utils/number';
import { floatToPercent, humanReadableBytes } from 'utils/string';
import { getDisplayName } from 'utils/user';

export const MIN_COLUMN_WIDTH = 40;
export const NO_PINS_WIDTH = 200;

export const MULTISELECT = 'selected';

// order used in ColumnPickerMenu
export const runColumns = [
  MULTISELECT,
  'id',
  'name',
  'state',
  'startTime',
  'user',
  'searcherType',
  'searcherMetric',
  'searcherMetricsVal',
  'tags',
  'forkedFrom',
  'progress',
  'duration',
  'resourcePool',
  'checkpointCount',
  'checkpointSize',
  'externalExperimentId',
  'externalTrialId',
  'experimentDescription',
  'parentArchived',
] as const;

export type RunColumn = (typeof runColumns)[number];

export const defaultExperimentColumns: RunColumn[] = [
  'id',
  'name',
  'state',
  'startTime',
  'user',
  'searcherType',
  'searcherMetric',
  'searcherMetricsVal',
  'tags',
  'progress',
  'duration',
  'resourcePool',
  'checkpointCount',
  'checkpointSize',
];

export type ColumnDef = SizedGridColumn & {
  id: string;
  isNumerical?: boolean;
  renderer: (record: FlatRun, idx: number) => GridCell;
  tooltip: (record: FlatRun) => string | undefined;
};

export type ColumnDefs = Record<string, ColumnDef>;

interface HeatmapProps {
  min: number;
  max: number;
}

interface Params {
  appTheme: Theme;
  columnWidths: Record<string, number>;
  rowSelection: CompactSelection;
  themeIsDark: boolean;
  users: Loadable<DetailedUser[]>;
  selectAll: boolean;
}
export const getColumnDefs = ({
  columnWidths,
  rowSelection,
  themeIsDark,
  users,
  selectAll,
  appTheme,
}: Params): ColumnDefs => ({
  archived: {
    id: 'parentArchived',
    renderer: (record: FlatRun) => ({
      allowOverlay: false,
      data: String(record.parentArchived),
      displayData: record.parentArchived ? '📦' : '',
      kind: GridCellKind.Text,
    }),
    title: 'Archived',
    tooltip: () => undefined,
    width: columnWidths.archived,
  },
  checkpointCount: {
    id: 'checkpointCount',
    isNumerical: true,
    renderer: (record: FlatRun) => ({
      allowOverlay: false,
      data: Number(record.checkpointCount),
      displayData: String(record.checkpointCount),
      kind: GridCellKind.Number,
    }),
    title: 'Checkpoints',
    tooltip: () => undefined,
    width: columnWidths.checkpointCount,
  },
  checkpointSize: {
    id: 'checkpointSize',
    isNumerical: true,
    renderer: (record: FlatRun) => ({
      allowOverlay: false,
      copyData: record.checkpointSize ? humanReadableBytes(record.checkpointSize) : '',
      data: { kind: 'text-cell' },
      kind: GridCellKind.Custom,
    }),
    title: 'Checkpoint Size',
    tooltip: () => undefined,
    width: columnWidths.checkpointSize,
  },
  duration: {
    id: 'duration',
    isNumerical: true,
    renderer: (record: FlatRun) => ({
      allowOverlay: false,
      copyData: record.duration
        ? durationInEnglish(record.duration, {
            conjunction: ' ',
            delimiter: ' ',
            largest: 2,
            serialComma: false,
            unitMeasures: { ...DURATION_UNIT_MEASURES, ms: 1000 },
          })
        : '',
      data: { kind: 'text-cell' },
      kind: GridCellKind.Custom,
    }),
    title: 'Duration',
    tooltip: () => undefined,
    width: columnWidths.duration,
  },
  externalExperimentId: {
    id: 'externalExperimentId',
    renderer: (record: FlatRun) => ({
      allowOverlay: false,
      copyData: record.externalExperimentId ?? '',
      data: { kind: 'text-cell' },
      kind: GridCellKind.Custom,
    }),
    title: 'External Experiment ID',
    tooltip: () => undefined,
    width: columnWidths.externalExperimentId,
  },
  externalTrialId: {
    id: 'externalTrialId',
    renderer: (record: FlatRun) => ({
      allowOverlay: false,
      copyData: String(record.externalRunId ?? ''),
      data: { kind: 'text-cell' },
      kind: GridCellKind.Custom,
    }),
    title: 'External Trial ID',
    tooltip: () => undefined,
    width: columnWidths.externalTrialId,
  },
  forkedFrom: {
    id: 'forkedFrom',
    renderer: (record: FlatRun) => ({
      allowOverlay: false,
      copyData: String(record.forkedFrom ?? ''),
      cursor: record.forkedFrom ? 'pointer' : undefined,
      data: {
        kind: 'link-cell',
        link:
          record.forkedFrom !== undefined
            ? {
                href: record.forkedFrom ? paths.experimentDetails(record.forkedFrom) : undefined,
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
    renderer: (record: FlatRun) => ({
      allowOverlay: false,
      copyData: String(record.id),
      cursor: 'pointer',
      data: {
        kind: 'link-cell',
        link: {
          href: paths.experimentDetails(record.id),
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
    id: 'searcherName',
    renderer: (record: FlatRun) => ({
      allowOverlay: false,
      copyData: String(record.experimentName),
      cursor: 'pointer',
      data: {
        kind: 'link-cell',
        link: record.experimentId
          ? {
              href: paths.experimentDetails(record.experimentId),
              title: String(record.experimentName),
              unmanaged: record.unmanaged,
            }
          : undefined,
        navigateOn: 'click',
        underlineOffset: 6,
      },
      kind: GridCellKind.Custom,
      readonly: true,
    }),
    title: 'Name',
    tooltip: () => undefined,
    width: columnWidths.name,
  },
  progress: {
    id: 'experimentProgress',
    renderer: (record: FlatRun) => {
      const percentage = floatToPercent(record.experimentProgress ?? 0, 0);

      return {
        allowOverlay: false,
        data: percentage,
        displayData: percentage,
        kind: GridCellKind.Text,
      };
    },
    title: 'Searcher Progress',
    tooltip: () => undefined,
    width: columnWidths.progress,
  },
  resourcePool: {
    id: 'resourcePool',
    renderer: (record: FlatRun) => ({
      allowOverlay: false,
      copyData: String(record.resourcePool),
      data: { kind: 'text-cell' },
      kind: GridCellKind.Custom,
    }),
    title: 'Resource Pool',
    tooltip: () => undefined,
    width: columnWidths.resourcePool,
  },
  searcherMetric: {
    id: 'searcherMetric',
    isNumerical: false,
    renderer: (record: FlatRun) => {
      const sMetric = record.searcherMetric ?? '';
      return {
        allowOverlay: false,
        copyData: sMetric,
        data: { kind: 'text-cell' },
        kind: GridCellKind.Custom,
      };
    },
    title: 'Searcher Metric',
    tooltip: () => undefined,
    width: columnWidths.searcherMetric,
  },
  searcherType: {
    id: 'searcherType',
    renderer: (record: FlatRun) => ({
      allowOverlay: false,
      copyData: String(record.searcherType),
      data: { kind: 'text-cell' },
      kind: GridCellKind.Custom,
    }),
    title: 'Searcher',
    tooltip: () => undefined,
    width: columnWidths.searcherType,
  },
  selected: {
    icon: selectAll ? 'allSelected' : rowSelection.length ? 'someSelected' : 'noneSelected',
    id: MULTISELECT,
    renderer: (_: FlatRun, idx) => ({
      allowOverlay: false,
      contentAlign: 'left',
      copyData: String(rowSelection.hasIndex(idx)),
      data: {
        checked: rowSelection.hasIndex(idx),
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
    renderer: (record: FlatRun) => ({
      allowOverlay: false,
      copyData: getTimeInEnglish(new Date(record.startTime)),
      data: { kind: 'text-cell' },
      kind: GridCellKind.Custom,
    }),
    title: 'Start Time',
    tooltip: () => undefined,
    width: columnWidths.startTime,
  },
  state: {
    id: 'state',
    renderer: (record: FlatRun) => ({
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
    tooltip: (record: FlatRun) => record.state.toLocaleLowerCase(),
    width: columnWidths.state,
  },
  tags: {
    id: 'tags',
    renderer: (record: FlatRun) => ({
      allowOverlay: true,
      copyData: record.labels?.join(', ') ?? '',
      data: {
        kind: 'tags-cell',
        possibleTags: [],
        readonly: true,
        tags: record.labels,
      },
      kind: GridCellKind.Custom,
    }),
    title: 'Tags',
    tooltip: () => undefined,
    width: columnWidths.tags,
  },
  user: {
    id: 'user',
    renderer: (record: FlatRun) => {
      const displayName = Loadable.match(users, {
        _: () => undefined,
        Loaded: (users) => getDisplayName(users?.find((u) => u.id === record.userId)),
      });
      return {
        allowOverlay: true,
        copyData: String(record.userId),
        data: {
          image: undefined,
          initials: getInitials(displayName),
          kind: 'user-profile-cell',
          tint: getColor(displayName, themeIsDark),
        },
        kind: GridCellKind.Custom,
      };
    },
    title: 'User',
    tooltip: (record: FlatRun) => {
      return Loadable.match(users, {
        _: () => undefined,
        Loaded: (users) => getDisplayName(users?.find((u) => u.id === record.userId)),
      });
    },
    width: columnWidths.user,
  },
});

export const searcherMetricsValColumn = (
  columnWidth?: number,
  heatmapProps?: HeatmapProps,
): ColumnDef => {
  return {
    id: 'searcherMetricsVal',
    isNumerical: true,
    renderer: (record: FlatRun) => {
      const sMetricValue = record.searcherMetricValue;

      let theme: Partial<GTheme> = {};
      if (heatmapProps && sMetricValue) {
        const { min, max } = heatmapProps;
        theme = {
          accentLight: getHeatmapColor(min, max, sMetricValue),
          bgCell: getHeatmapColor(min, max, sMetricValue),
          textDark: 'white',
        };
      }
      return {
        allowOverlay: false,
        copyData: sMetricValue
          ? typeof sMetricValue === 'number'
            ? humanReadableNumber(sMetricValue)
            : sMetricValue
          : '',
        data: { kind: 'text-cell' },
        kind: GridCellKind.Custom,
        themeOverride: theme,
      };
    },
    title: 'Searcher Metric Value',
    tooltip: () => undefined,
    width: columnWidth ?? columnWidthsFallback,
  };
};

export const defaultTextColumn = (
  column: ProjectColumn,
  columnWidth?: number,
  dataPath?: string,
): ColumnDef => {
  return {
    id: column.column,
    renderer: (record: FlatRun) => {
      const data = isString(dataPath) ? getPath<string>(record, dataPath) : undefined;
      return {
        allowOverlay: false,
        copyData: String(data ?? ''),
        data: { kind: 'text-cell' },
        kind: GridCellKind.Custom,
      };
    },
    title: column.displayName || column.column,
    tooltip: () => undefined,
    width: columnWidth ?? columnWidthsFallback,
  };
};

const getHeatmapPercentage = (min: number, max: number, value: number): number => {
  if (min >= max || value >= max) return 1;
  if (value <= min) return 0;
  return (value - min) / (max - min);
};

const getHeatmapColor = (min: number, max: number, value: number): string => {
  const p = getHeatmapPercentage(min, max, value);
  const red = [44, 222];
  const green = [119, 66];
  const blue = [176, 91];
  return `rgb(${red[0] + (red[1] - red[0]) * p}, ${green[0] + (green[1] - green[0]) * p}, ${
    blue[0] + (blue[1] - blue[0]) * p
  })`;
};

export const defaultNumberColumn = (
  column: ProjectColumn,
  columnWidth?: number,
  dataPath?: string,
  heatmapProps?: HeatmapProps,
): ColumnDef => {
  return {
    id: column.column,
    renderer: (record: FlatRun) => {
      const data = isString(dataPath) ? getPath<number>(record, dataPath) : undefined;
      let theme: Partial<GTheme> = {};
      if (heatmapProps && data !== undefined) {
        const { min, max } = heatmapProps;
        theme = {
          accentLight: getHeatmapColor(min, max, data),
          bgCell: getHeatmapColor(min, max, data),
          textDark: 'white',
        };
      }
      return {
        allowOverlay: false,
        copyData: data !== undefined ? String(data) : '',
        data: { kind: 'text-cell' },
        kind: GridCellKind.Custom,
        themeOverride: theme,
      };
    },
    title: column.displayName || column.column,
    tooltip: () => undefined,
    width: columnWidth ?? columnWidthsFallback,
  };
};

export const defaultDateColumn = (
  column: ProjectColumn,
  columnWidth?: number,
  dataPath?: string,
): ColumnDef => {
  return {
    id: column.column,
    renderer: (record: FlatRun) => {
      const data = isString(dataPath) ? getPath<string>(record, dataPath) : undefined;
      return {
        allowOverlay: false,
        copyData: formatDatetime(String(data), { outputUTC: false }),
        data: { kind: 'text-cell' },
        kind: GridCellKind.Custom,
      };
    },
    title: column.displayName || column.column,
    tooltip: () => undefined,
    width: columnWidth ?? columnWidthsFallback,
  };
};

export const columnWidthsFallback = 140;

export const defaultColumnWidths: Record<RunColumn, number> = {
  checkpointCount: 120,
  checkpointSize: 110,
  duration: 86,
  experimentDescription: 148,
  externalExperimentId: 160,
  externalTrialId: 130,
  forkedFrom: 86,
  id: 50,
  name: 290,
  parentArchived: 80,
  progress: 65,
  resourcePool: 140,
  searcherMetric: 120,
  searcherMetricsVal: 120,
  searcherType: 120,
  selected: 40,
  startTime: 118,
  state: 60,
  tags: 106,
  user: 50,
};

// TODO: use theme here
// eslint-disable-next-line @typescript-eslint/no-unused-vars
export const getHeaderIcons = (_appTheme: Theme): DataEditorProps['headerIcons'] => ({
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
