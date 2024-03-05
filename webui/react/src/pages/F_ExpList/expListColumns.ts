import { CompactSelection, GridCellKind, Theme as GTheme } from '@glideapps/glide-data-grid';
import { getColor, getInitials } from 'hew/Avatar';
import { Theme } from 'hew/Theme';
import { Loadable } from 'hew/utils/loadable';

import { paths } from 'routes/utils';
import { DetailedUser, ExperimentWithTrial } from 'types';
import { humanReadableNumber } from 'utils/number';
import { floatToPercent, humanReadableBytes } from 'utils/string';
import { getDisplayName } from 'utils/user';

import {
  ColumnDef,
  ColumnDefs,
  columnWidthsFallback,
  getHeatmapColor,
  HeatmapProps,
  MULTISELECT,
} from './glide-table/columns';
import { getDurationInEnglish, getTimeInEnglish } from './utils';

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
  'externalExperimentId',
  'externalTrialId',
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
}: Params): ColumnDefs<ExperimentWithTrial> => ({
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
      copyData: record.experiment.checkpointSize
        ? humanReadableBytes(record.experiment.checkpointSize)
        : '',
      data: { kind: 'text-cell' },
      kind: GridCellKind.Custom,
    }),
    title: 'Checkpoint Size',
    tooltip: () => undefined,
    width: columnWidths.checkpointSize,
  },
  description: {
    id: 'description',
    renderer: (record: ExperimentWithTrial) => ({
      allowOverlay: false,
      copyData: String(record.experiment.description),
      data: { kind: 'text-cell' },
      kind: GridCellKind.Custom,
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
      copyData: getDurationInEnglish(record.experiment),
      data: { kind: 'text-cell' },
      kind: GridCellKind.Custom,
    }),
    title: 'Duration',
    tooltip: () => undefined,
    width: columnWidths.duration,
  },
  externalExperimentId: {
    id: 'externalExperimentId',
    renderer: (record: ExperimentWithTrial) => ({
      allowOverlay: false,
      copyData: record.experiment.externalExperimentId ?? '',
      data: { kind: 'text-cell' },
      kind: GridCellKind.Custom,
    }),
    title: 'External Experiment ID',
    tooltip: () => undefined,
    width: columnWidths.externalExperimentId,
  },
  externalTrialId: {
    id: 'externalTrialId',
    renderer: (record: ExperimentWithTrial) => ({
      allowOverlay: false,
      copyData: record.experiment.externalTrialId ?? '',
      data: { kind: 'text-cell' },
      kind: GridCellKind.Custom,
    }),
    title: 'External Trial ID',
    tooltip: () => undefined,
    width: columnWidths.externalTrialId,
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
          unmanaged: record.experiment.unmanaged,
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
      const percentage = floatToPercent(record.experiment.progress ?? 0, 0);

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
      copyData: String(record.experiment.resourcePool),
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
    renderer: (record: ExperimentWithTrial) => {
      const sMetric = record.experiment.searcherMetric ?? '';
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
    renderer: (record: ExperimentWithTrial) => ({
      allowOverlay: false,
      copyData: String(record.experiment.searcherType),
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
      copyData: getTimeInEnglish(new Date(record.experiment.startTime)),
      data: { kind: 'text-cell' },
      kind: GridCellKind.Custom,
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
        _: () => undefined,
        Loaded: (users) => getDisplayName(users?.find((u) => u.id === record.experiment.userId)),
      });
      return {
        allowOverlay: true,
        copyData: String(record.experiment.userId),
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
    tooltip: (record: ExperimentWithTrial) => {
      return Loadable.match(users, {
        _: () => undefined,
        Loaded: (users) => getDisplayName(users?.find((u) => u.id === record.experiment.userId)),
      });
    },
    width: columnWidths.user,
  },
});

export const searcherMetricsValColumn = (
  columnWidth?: number,
  heatmapProps?: HeatmapProps,
): ColumnDef<ExperimentWithTrial> => {
  return {
    id: 'searcherMetricsVal',
    isNumerical: true,
    renderer: (record: ExperimentWithTrial) => {
      const sMetricValue = record.bestTrial?.searcherMetricsVal;

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

export const defaultColumnWidths: Record<ExperimentColumn, number> = {
  archived: 80,
  checkpointCount: 120,
  checkpointSize: 110,
  description: 148,
  duration: 86,
  externalExperimentId: 160,
  externalTrialId: 130,
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
