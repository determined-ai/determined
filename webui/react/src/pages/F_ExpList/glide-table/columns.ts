import {
  CompactSelection,
  DataEditorProps,
  GridCell,
  GridCellKind,
  Theme as GTheme,
  SizedGridColumn,
} from '@hpe.com/glide-data-grid';

import { getColor, getInitials } from 'components/Avatar';
import { terminalRunStates } from 'constants/states';
import { paths } from 'routes/utils';
import { DetailedUser, ExperimentWithTrial, ProjectColumn } from 'types';
import { getPath, isString } from 'utils/data';
import { formatDatetime } from 'utils/datetime';
import { Loadable } from 'utils/loadable';
import { humanReadableNumber } from 'utils/number';
import { humanReadableBytes } from 'utils/string';
import { DarkLight, Theme } from 'utils/themes';
import { getDisplayName } from 'utils/user';

import { getDurationInEnglish, getTimeInEnglish } from './utils';

export const MIN_COLUMN_WIDTH = 40;
export const NO_PINS_WIDTH = 200;

export const MULTISELECT = 'selected';

// order used in ColumnPickerMenu
export const experimentColumns = [
  MULTISELECT,
  'id',
  'name',
  'state',
  'startTime',
  'user',
  'numTrials',
  'searcherType',
  'searcherMetric',
  'searcherMetricsVal',
  'description',
  'tags',
  'forkedFrom',
  'progress',
  'duration',
  'resourcePool',
  'checkpointCount',
  'checkpointSize',
  'archived',
] as const;

export type ExperimentColumn = (typeof experimentColumns)[number];

export const defaultExperimentColumns: ExperimentColumn[] = [
  'id',
  'name',
  'state',
  'startTime',
  'user',
  'numTrials',
  'searcherType',
  'searcherMetric',
  'searcherMetricsVal',
  'description',
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
  renderer: (record: ExperimentWithTrial, idx: number) => GridCell;
  tooltip: (record: ExperimentWithTrial) => string | undefined;
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
  darkLight: DarkLight;
  users: Loadable<DetailedUser[]>;
  selectAll: boolean;
}
export const getColumnDefs = ({
  columnWidths,
  rowSelection,
  darkLight,
  users,
  selectAll,
  appTheme,
}: Params): ColumnDefs => ({
  archived: {
    id: 'archived',
    renderer: (record: ExperimentWithTrial) => ({
      allowOverlay: false,
      data: String(record.experiment.archived),
      displayData: record.experiment.archived ? '📦' : '',
      kind: GridCellKind.Text,
    }),
    title: 'Archived',
    tooltip: () => undefined,
    width: columnWidths.archived,
  },
  checkpointCount: {
    id: 'checkpointCount',
    isNumerical: true,
    renderer: (record: ExperimentWithTrial) => ({
      allowOverlay: false,
      data: Number(record.experiment.checkpoints),
      displayData: String(record.experiment.checkpoints),
      kind: GridCellKind.Number,
    }),
    title: 'Checkpoints',
    tooltip: () => undefined,
    width: columnWidths.checkpointCount,
  },
  checkpointSize: {
    id: 'checkpointSize',
    isNumerical: true,
    renderer: (record: ExperimentWithTrial) => ({
      allowOverlay: false,
      data: record.experiment.checkpointSize
        ? humanReadableBytes(record.experiment.checkpointSize)
        : '',
      displayData: record.experiment.checkpointSize
        ? humanReadableBytes(record.experiment.checkpointSize)
        : '',
      kind: GridCellKind.Text,
    }),
    title: 'Checkpoint Size',
    tooltip: () => undefined,
    width: columnWidths.checkpointSize,
  },
  description: {
    id: 'description',
    renderer: (record: ExperimentWithTrial) => ({
      allowOverlay: false,
      data: String(record.experiment.description),
      displayData: String(record.experiment.description),
      kind: GridCellKind.Text,
    }),
    title: 'Description',
    tooltip: () => undefined,
    width: columnWidths.description,
  },
  duration: {
    id: 'duration',
    isNumerical: true,
    renderer: (record: ExperimentWithTrial) => ({
      allowOverlay: false,
      data: getDurationInEnglish(record.experiment),
      displayData: getDurationInEnglish(record.experiment),
      kind: GridCellKind.Text,
    }),
    title: 'Duration',
    tooltip: () => undefined,
    width: columnWidths.duration,
  },
  forkedFrom: {
    id: 'forkedFrom',
    renderer: (record: ExperimentWithTrial) => ({
      allowOverlay: false,
      copyData: String(record.experiment.forkedFrom ?? ''),
      cursor: record.experiment.forkedFrom ? 'pointer' : undefined,
      data: {
        kind: 'link-cell',
        link:
          record.experiment.forkedFrom !== undefined
            ? {
                href: record.experiment.forkedFrom
                  ? paths.experimentDetails(record.experiment.forkedFrom)
                  : undefined,
                title: String(record.experiment.forkedFrom ?? ''),
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
    renderer: (record: ExperimentWithTrial) => ({
      allowOverlay: false,
      copyData: String(record.experiment.id),
      cursor: 'pointer',
      data: {
        kind: 'link-cell',
        link: {
          href: paths.experimentDetails(record.experiment.id),
          title: String(record.experiment.id),
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
    renderer: (record: ExperimentWithTrial) => ({
      allowOverlay: false,
      copyData: String(record.experiment.name),
      cursor: 'pointer',
      data: {
        kind: 'link-cell',
        link: {
          href: paths.experimentDetails(record.experiment.id),
          title: String(record.experiment.name),
        },
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
  numTrials: {
    id: 'numTrials',
    isNumerical: true,
    renderer: (record: ExperimentWithTrial) => ({
      allowOverlay: false,
      data: record.experiment.numTrials,
      displayData: String(record.experiment.numTrials),
      kind: GridCellKind.Number,
    }),
    title: 'Trials',
    tooltip: () => undefined,
    width: columnWidths.numTrials,
  },
  progress: {
    id: 'progress',
    renderer: (record: ExperimentWithTrial) => {
      const progress = [...terminalRunStates.keys()].includes(record.experiment.state)
        ? 1
        : record.experiment.progress ?? 0;
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
    renderer: (record: ExperimentWithTrial) => ({
      allowOverlay: false,
      data: String(record.experiment.resourcePool),
      displayData: String(record.experiment.resourcePool),
      kind: GridCellKind.Text,
    }),
    title: 'Resource Pool',
    tooltip: () => undefined,
    width: columnWidths.resourcePool,
  },
  searcherMetric: {
    id: 'searcherMetric',
    isNumerical: false,
    renderer: (record: ExperimentWithTrial) => {
      const sMetric = record.experiment.config.searcher.metric;
      return {
        allowOverlay: false,
        data: sMetric,
        displayData: sMetric,
        kind: GridCellKind.Text,
      };
    },
    title: 'Searcher Metric',
    tooltip: () => undefined,
    width: columnWidths.searcherMetric,
  },
  searcherType: {
    id: 'searcherType',
    renderer: (record: ExperimentWithTrial) => ({
      allowOverlay: false,
      data: String(record.experiment.searcherType),
      displayData: String(record.experiment.searcherType),
      kind: GridCellKind.Text,
    }),
    title: 'Searcher',
    tooltip: () => undefined,
    width: columnWidths.searcherType,
  },
  selected: {
    icon: selectAll ? 'allSelected' : rowSelection.length ? 'someSelected' : 'noneSelected',
    id: MULTISELECT,
    renderer: (_: ExperimentWithTrial, idx) => ({
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
    renderer: (record: ExperimentWithTrial) => ({
      allowOverlay: false,
      data: getTimeInEnglish(new Date(record.experiment.startTime)),
      displayData: getTimeInEnglish(new Date(record.experiment.startTime)),
      kind: GridCellKind.Text,
    }),
    title: 'Start Time',
    tooltip: () => undefined,
    width: columnWidths.startTime,
  },
  state: {
    id: 'state',
    renderer: (record: ExperimentWithTrial) => ({
      allowAdd: false,
      allowOverlay: true,
      copyData: record.experiment.state.toLocaleLowerCase(),
      data: {
        appTheme,
        kind: 'experiment-state-cell',
        state: record.experiment.state,
      },
      kind: GridCellKind.Custom,
    }),
    themeOverride: { cellHorizontalPadding: 13 },
    title: 'State',
    tooltip: (record: ExperimentWithTrial) => record.experiment.state.toLocaleLowerCase(),
    width: columnWidths.state,
  },
  tags: {
    id: 'tags',
    renderer: (record: ExperimentWithTrial) => ({
      allowOverlay: true,
      copyData: record.experiment['labels'].join(', '),
      data: {
        kind: 'tags-cell',
        possibleTags: [],
        readonly: true,
        tags: record.experiment['labels'],
      },
      kind: GridCellKind.Custom,
    }),
    title: 'Tags',
    tooltip: () => undefined,
    width: columnWidths.tags,
  },
  user: {
    id: 'user',
    renderer: (record: ExperimentWithTrial) => {
      const displayName = Loadable.match(users, {
        Loaded: (users) => getDisplayName(users?.find((u) => u.id === record.experiment.userId)),
        NotLoaded: () => undefined,
      });
      return {
        allowOverlay: true,
        copyData: String(record.experiment.userId),
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
    tooltip: (record: ExperimentWithTrial) => {
      return Loadable.match(users, {
        Loaded: (users) => getDisplayName(users?.find((u) => u.id === record.experiment.userId)),
        NotLoaded: () => undefined,
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
    renderer: (record: ExperimentWithTrial) => {
      const sMetric = record.experiment.config.searcher.metric;
      const sMetricValue = record.bestTrial?.summaryValidationMetrics?.metrics?.[sMetric];

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
        data: sMetricValue?.toString() || '',
        displayData: sMetricValue
          ? typeof sMetricValue === 'number'
            ? humanReadableNumber(sMetricValue)
            : sMetricValue
          : '',
        kind: GridCellKind.Text,
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
    renderer: (record: ExperimentWithTrial) => {
      const data = isString(dataPath) ? getPath<string>(record, dataPath) : undefined;
      return {
        allowOverlay: false,
        data: String(data),
        displayData: String(data ?? ''),
        kind: GridCellKind.Text,
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
    renderer: (record: ExperimentWithTrial) => {
      const data = isString(dataPath) ? getPath<number>(record, dataPath) : undefined;
      let theme: Partial<GTheme> = {};
      if (heatmapProps && data) {
        const { min, max } = heatmapProps;
        theme = {
          accentLight: getHeatmapColor(min, max, data),
          bgCell: getHeatmapColor(min, max, data),
          textDark: 'white',
        };
      }
      return {
        allowOverlay: false,
        data: Number(data),
        displayData: data !== undefined ? String(data) : '',
        kind: GridCellKind.Number,
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
    renderer: (record: ExperimentWithTrial) => {
      const data = isString(dataPath) ? getPath<string>(record, dataPath) : undefined;
      return {
        allowOverlay: false,
        data: String(data),
        displayData: formatDatetime(String(data), { outputUTC: false }),
        kind: GridCellKind.Text,
      };
    },
    title: column.displayName || column.column,
    tooltip: () => undefined,
    width: columnWidth ?? columnWidthsFallback,
  };
};

export const columnWidthsFallback = 140;

export const defaultColumnWidths: Record<ExperimentColumn, number> = {
  archived: 80,
  checkpointCount: 120,
  checkpointSize: 110,
  description: 148,
  duration: 86,
  forkedFrom: 86,
  id: 50,
  name: 290,
  numTrials: 50,
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
