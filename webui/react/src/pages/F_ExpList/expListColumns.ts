import { CellClickedEventArgs, GridCellKind, Theme as GTheme } from '@glideapps/glide-data-grid';
import { getColor, getInitials } from 'hew/Avatar';
import {
  ColumnDef,
  ColumnDefs,
  DEFAULT_COLUMN_WIDTH,
  getHeatmapColor,
  HeatmapProps,
  MIN_COLUMN_WIDTH,
} from 'hew/DataGrid/columns';
import {
  LINK_CELL,
  State,
  STATE_CELL,
  TAGS_CELL,
  TEXT_CELL,
  USER_AVATAR_CELL,
} from 'hew/DataGrid/custom-renderers/index';
import { Theme } from 'hew/Theme';
import { Loadable } from 'hew/utils/loadable';

import { handlePath, paths } from 'routes/utils';
import { CompoundRunState, DetailedUser, ExperimentWithTrial, JobState, RunState } from 'types';
import { getDurationInEnglish, getTimeInEnglish } from 'utils/datetime';
import { humanReadableNumber } from 'utils/number';
import { AnyMouseEvent } from 'utils/routes';
import { floatToPercent, humanReadableBytes } from 'utils/string';
import { getDisplayName } from 'utils/user';

// order used in ColumnPickerMenu
export const experimentColumns = [
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

function getCellStateFromExperimentState(expState: CompoundRunState) {
  switch (expState) {
    case JobState.SCHEDULED:
    case JobState.SCHEDULEDBACKFILLED:
    case JobState.QUEUED:
    case RunState.Queued: {
      return State.QUEUED;
    }
    case RunState.Starting:
    case RunState.Pulling: {
      return State.STARTING;
    }
    case RunState.Running: {
      return State.RUNNING;
    }
    case RunState.Paused: {
      return State.PAUSED;
    }
    case RunState.Completed: {
      return State.SUCCESS;
    }
    case RunState.Error:
    case RunState.Deleted:
    case RunState.Deleting:
    case RunState.DeleteFailed: {
      return State.ERROR;
    }
    case RunState.Active:
    case RunState.Unspecified:
    case JobState.UNSPECIFIED: {
      return State.ACTIVE;
    }
    default: {
      return State.STOPPED;
    }
  }
}

interface Params {
  appTheme: Theme;
  columnWidths: Record<string, number | null | undefined>;
  themeIsDark: boolean;
  users: Loadable<DetailedUser[]>;
}
export const getColumnDefs = ({
  columnWidths,
  themeIsDark,
  users,
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
    width: columnWidths.archived ?? defaultColumnWidths.archived ?? MIN_COLUMN_WIDTH,
  },
  checkpointCount: {
    id: 'checkpointCount',
    renderer: (record: ExperimentWithTrial) => ({
      allowOverlay: false,
      data: Number(record.experiment.checkpoints),
      displayData: String(record.experiment.checkpoints),
      kind: GridCellKind.Number,
    }),
    title: 'Checkpoints',
    tooltip: () => undefined,
    type: 'number',
    width: columnWidths.checkpointCount ?? defaultColumnWidths.checkpointCount ?? MIN_COLUMN_WIDTH,
  },
  checkpointSize: {
    id: 'checkpointSize',
    renderer: (record: ExperimentWithTrial) => ({
      allowOverlay: false,
      copyData: record.experiment.checkpointSize
        ? humanReadableBytes(record.experiment.checkpointSize)
        : '',
      data: { kind: TEXT_CELL },
      kind: GridCellKind.Custom,
    }),
    title: 'Checkpoint Size',
    tooltip: () => undefined,
    type: 'number',
    width: columnWidths.checkpointSize ?? defaultColumnWidths.checkpointSize ?? MIN_COLUMN_WIDTH,
  },
  description: {
    id: 'description',
    renderer: (record: ExperimentWithTrial) => ({
      allowOverlay: false,
      copyData: String(record.experiment.description),
      data: { kind: TEXT_CELL },
      kind: GridCellKind.Custom,
    }),
    title: 'Description',
    tooltip: () => undefined,
    width: columnWidths.description ?? defaultColumnWidths.description ?? MIN_COLUMN_WIDTH,
  },
  duration: {
    id: 'duration',
    renderer: (record: ExperimentWithTrial) => ({
      allowOverlay: false,
      copyData: getDurationInEnglish(record.experiment),
      data: { kind: TEXT_CELL },
      kind: GridCellKind.Custom,
    }),
    title: 'Duration',
    tooltip: () => undefined,
    type: 'number',
    width: columnWidths.duration ?? defaultColumnWidths.duration ?? MIN_COLUMN_WIDTH,
  },
  externalExperimentId: {
    id: 'externalExperimentId',
    renderer: (record: ExperimentWithTrial) => ({
      allowOverlay: false,
      copyData: record.experiment.externalExperimentId ?? '',
      data: { kind: TEXT_CELL },
      kind: GridCellKind.Custom,
    }),
    title: 'External Experiment ID',
    tooltip: () => undefined,
    width:
      columnWidths.externalExperimentId ??
      defaultColumnWidths.externalExperimentId ??
      MIN_COLUMN_WIDTH,
  },
  externalTrialId: {
    id: 'externalTrialId',
    renderer: (record: ExperimentWithTrial) => ({
      allowOverlay: false,
      copyData: record.experiment.externalTrialId ?? '',
      data: { kind: TEXT_CELL },
      kind: GridCellKind.Custom,
    }),
    title: 'External Trial ID',
    tooltip: () => undefined,
    width: columnWidths.externalTrialId ?? defaultColumnWidths.externalTrialId ?? MIN_COLUMN_WIDTH,
  },
  forkedFrom: {
    id: 'forkedFrom',
    renderer: (record: ExperimentWithTrial) => ({
      allowOverlay: false,
      copyData: String(record.experiment.forkedFrom ?? ''),
      cursor: record.experiment.forkedFrom ? 'pointer' : undefined,
      data: {
        kind: LINK_CELL,
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
      onClick: (e: CellClickedEventArgs) => {
        if (record.experiment.forkedFrom) {
          handlePath(e as unknown as AnyMouseEvent, {
            path: String(record.experiment.forkedFrom),
          });
        }
      },
      readonly: true,
    }),
    title: 'Forked From',
    tooltip: () => undefined,
    width: columnWidths.forkedFrom ?? defaultColumnWidths.forkedFrom ?? MIN_COLUMN_WIDTH,
  },
  id: {
    id: 'id',
    renderer: (record: ExperimentWithTrial) => ({
      allowOverlay: false,
      copyData: String(record.experiment.id),
      cursor: 'pointer',
      data: {
        kind: LINK_CELL,
        link: {
          href: paths.experimentDetails(record.experiment.id),
          title: String(record.experiment.id),
        },
        navigateOn: 'click',
        onClick: (e: CellClickedEventArgs) => {
          handlePath(e as unknown as AnyMouseEvent, {
            path: paths.experimentDetails(record.experiment.id),
          });
        },
        underlineOffset: 6,
      },
      kind: GridCellKind.Custom,
      readonly: true,
    }),
    title: 'ID',
    tooltip: () => undefined,
    width: columnWidths.id ?? defaultColumnWidths.id ?? MIN_COLUMN_WIDTH,
  },
  name: {
    id: 'name',
    renderer: (record: ExperimentWithTrial) => ({
      allowOverlay: false,
      copyData: String(record.experiment.name),
      cursor: 'pointer',
      data: {
        kind: LINK_CELL,
        link: {
          href: paths.experimentDetails(record.experiment.id),
          title: String(record.experiment.name),
          unmanaged: record.experiment.unmanaged,
        },
        navigateOn: 'click',
        onClick: (e: CellClickedEventArgs) => {
          handlePath(e as unknown as AnyMouseEvent, {
            path: paths.experimentDetails(record.experiment.id),
          });
        },
        underlineOffset: 6,
      },
      kind: GridCellKind.Custom,
      readonly: true,
    }),
    title: 'Name',
    tooltip: () => undefined,
    width: columnWidths.name ?? defaultColumnWidths.name ?? MIN_COLUMN_WIDTH,
  },
  numTrials: {
    id: 'numTrials',
    renderer: (record: ExperimentWithTrial) => ({
      allowOverlay: false,
      data: record.experiment.numTrials,
      displayData: String(record.experiment.numTrials),
      kind: GridCellKind.Number,
    }),
    title: 'Trials',
    tooltip: () => undefined,
    type: 'number',
    width: columnWidths.numTrials ?? defaultColumnWidths.numTrials ?? MIN_COLUMN_WIDTH,
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
    width: columnWidths.progress ?? defaultColumnWidths.progress ?? MIN_COLUMN_WIDTH,
  },
  resourcePool: {
    id: 'resourcePool',
    renderer: (record: ExperimentWithTrial) => ({
      allowOverlay: false,
      copyData: String(record.experiment.resourcePool),
      data: { kind: TEXT_CELL },
      kind: GridCellKind.Custom,
    }),
    title: 'Resource Pool',
    tooltip: () => undefined,
    width: columnWidths.resourcePool ?? defaultColumnWidths.resourcePool ?? MIN_COLUMN_WIDTH,
  },
  searcherMetric: {
    id: 'searcherMetric',
    renderer: (record: ExperimentWithTrial) => {
      const sMetric = record.experiment.searcherMetric ?? '';
      return {
        allowOverlay: false,
        copyData: sMetric,
        data: { kind: TEXT_CELL },
        kind: GridCellKind.Custom,
      };
    },
    title: 'Searcher Metric',
    tooltip: () => undefined,
    width: columnWidths.searcherMetric ?? defaultColumnWidths.searcherMetric ?? MIN_COLUMN_WIDTH,
  },
  searcherType: {
    id: 'searcherType',
    renderer: (record: ExperimentWithTrial) => ({
      allowOverlay: false,
      copyData: String(record.experiment.searcherType),
      data: { kind: TEXT_CELL },
      kind: GridCellKind.Custom,
    }),
    title: 'Searcher',
    tooltip: () => undefined,
    width: columnWidths.searcherType ?? defaultColumnWidths.searcherType ?? MIN_COLUMN_WIDTH,
  },
  startTime: {
    id: 'startTime',
    renderer: (record: ExperimentWithTrial) => ({
      allowOverlay: false,
      copyData: getTimeInEnglish(new Date(record.experiment.startTime)),
      data: { kind: TEXT_CELL },
      kind: GridCellKind.Custom,
    }),
    title: 'Start Time',
    tooltip: () => undefined,
    type: 'number',
    width: columnWidths.startTime ?? defaultColumnWidths.startTime ?? MIN_COLUMN_WIDTH,
  },
  state: {
    id: 'state',
    renderer: (record: ExperimentWithTrial) => ({
      allowAdd: false,
      allowOverlay: true,
      copyData: record.experiment.state.toLocaleLowerCase(),
      data: {
        appTheme,
        kind: STATE_CELL,
        state: getCellStateFromExperimentState(record.experiment.state),
      },
      kind: GridCellKind.Custom,
    }),
    themeOverride: { cellHorizontalPadding: 13 },
    title: 'State',
    tooltip: (record: ExperimentWithTrial) => record.experiment.state.toLocaleLowerCase(),
    width: columnWidths.state ?? defaultColumnWidths.state ?? MIN_COLUMN_WIDTH,
  },
  tags: {
    id: 'tags',
    renderer: (record: ExperimentWithTrial) => ({
      allowOverlay: true,
      copyData: record.experiment['labels'].join(', '),
      data: {
        kind: TAGS_CELL,
        possibleTags: [],
        readonly: true,
        tags: record.experiment['labels'],
      },
      kind: GridCellKind.Custom,
    }),
    title: 'Tags',
    tooltip: () => undefined,
    width: columnWidths.tags ?? defaultColumnWidths.tags ?? MIN_COLUMN_WIDTH,
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
          kind: USER_AVATAR_CELL,
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
    width: columnWidths.user ?? defaultColumnWidths.user ?? MIN_COLUMN_WIDTH,
  },
});

export const searcherMetricsValColumn = (
  columnWidth?: number,
  heatmapProps?: HeatmapProps,
): ColumnDef<ExperimentWithTrial> => {
  return {
    id: 'searcherMetricsVal',
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
        data: { kind: TEXT_CELL },
        kind: GridCellKind.Custom,
        themeOverride: theme,
      };
    },
    title: 'Searcher Metric Value',
    tooltip: () => undefined,
    type: 'number',
    width: columnWidth ?? DEFAULT_COLUMN_WIDTH,
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
  startTime: 118,
  state: 60,
  tags: 106,
  user: 50,
};
