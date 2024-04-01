import { CellClickedEventArgs, GridCellKind } from '@glideapps/glide-data-grid';
import { getColor, getInitials } from 'hew/Avatar';
import { ColumnDefs } from 'hew/DataGrid/columns';
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

import { getTimeInEnglish } from 'pages/F_ExpList/utils';
import { handlePath, paths } from 'routes/utils';
import { DetailedUser, FlatRun, RunState } from 'types';
import { DURATION_UNIT_MEASURES, durationInEnglish } from 'utils/datetime';
import { humanReadableNumber } from 'utils/number';
import { AnyMouseEvent } from 'utils/routes';
import { capitalize, floatToPercent, humanReadableBytes } from 'utils/string';
import { getDisplayName } from 'utils/user';

// order used in ColumnPickerMenu
export const runColumns = [
  'id',
  'state',
  'startTime',
  'endTime',
  'user',
  'searcherType',
  'searcherMetric',
  'searcherMetricsVal',
  'tags',
  'forkedFrom',
  'duration',
  'experimentProgress',
  'experimentId',
  'name',
  'experimentDescription',
  'resourcePool',
  'checkpointCount',
  'checkpointSize',
  'externalExperimentId',
  'externalRunId',
  'experimentDescription',
  'isExpMultitrial',
  'parentArchived',
] as const;

export type RunColumn = (typeof runColumns)[number];

export const defaultRunColumns: RunColumn[] = [...runColumns];

function getCellStateFromExperimentState(expState: RunState) {
  switch (expState) {
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
    case RunState.Unspecified: {
      return State.ACTIVE;
    }
    default: {
      return State.STOPPED;
    }
  }
}

interface Params {
  appTheme: Theme;
  columnWidths: Record<string, number>;
  themeIsDark: boolean;
  users: Loadable<DetailedUser[]>;
}
export const getColumnDefs = ({
  columnWidths,
  themeIsDark,
  users,
  appTheme,
}: Params): ColumnDefs<FlatRun> => ({
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
      data: { kind: TEXT_CELL },
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
        ? durationInEnglish(
            (record.endTime ? new Date(record.endTime) : new Date()).getTime() -
              new Date(record.startTime).getTime(),
            {
              conjunction: ' ',
              delimiter: ' ',
              largest: 2,
              serialComma: false,
              unitMeasures: { ...DURATION_UNIT_MEASURES, ms: 1000 },
            },
          )
        : '',
      data: { kind: TEXT_CELL },
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
      copyData: record.experiment?.externalExperimentId ?? '',
      data: { kind: TEXT_CELL },
      kind: GridCellKind.Custom,
    }),
    title: 'External Experiment ID',
    tooltip: () => undefined,
    width: columnWidths.externalExperimentId,
  },
  externalRunId: {
    id: 'externalRunId',
    renderer: (record: FlatRun) => ({
      allowOverlay: false,
      copyData: String(record.externalRunId ?? ''),
      data: { kind: TEXT_CELL },
      kind: GridCellKind.Custom,
    }),
    title: 'External Run ID',
    tooltip: () => undefined,
    width: columnWidths.externalRunId,
  },
  forkedFrom: {
    id: 'forkedFrom',
    renderer: (record: FlatRun) => ({
      allowOverlay: false,
      copyData: String(record.experiment?.forkedFrom ?? ''),
      cursor: record.experiment?.forkedFrom ? 'pointer' : undefined,
      data: {
        kind: LINK_CELL,
        link:
          record.experiment?.forkedFrom !== undefined
            ? {
                href: record.experiment?.forkedFrom
                  ? paths.experimentDetails(record.experiment?.forkedFrom)
                  : undefined,
                title: String(record.experiment?.forkedFrom ?? ''),
              }
            : undefined,
        navigateOn: 'click',
        onClick: (e: CellClickedEventArgs) => {
          if (record.experiment?.forkedFrom) {
            handlePath(e as unknown as AnyMouseEvent, {
              path: String(record.experiment.forkedFrom),
            });
          }
        },
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
        kind: LINK_CELL,
        link: record.experiment?.id
          ? {
              href: paths.trialDetails(record.id, record.experiment.id),
              title: String(record.id),
              unmanaged: record.experiment.unmanaged,
            }
          : undefined,
        navigateOn: 'click',
        onClick: (e: CellClickedEventArgs) => {
          if (record.experiment) {
            handlePath(e as unknown as AnyMouseEvent, {
              path: paths.trialDetails(record.id, record.experiment.id),
            });
          }
        },
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
    renderer: (record: FlatRun) => ({
      allowOverlay: false,
      copyData: String(record.experiment?.name),
      cursor: 'pointer',
      data: {
        kind: LINK_CELL,
        link: record.experiment?.id
          ? {
              href: paths.experimentDetails(record.experiment?.id),
              title: String(record.experiment?.name),
              unmanaged: record.experiment.unmanaged,
            }
          : undefined,
        navigateOn: 'click',
        onClick: (e: CellClickedEventArgs) => {
          if (record.experiment) {
            handlePath(e as unknown as AnyMouseEvent, {
              path: paths.experimentDetails(record.experiment.id),
            });
          }
        },
        underlineOffset: 6,
      },
      kind: GridCellKind.Custom,

      readonly: true,
    }),
    title: 'Searcher Name',
    tooltip: () => undefined,
    width: columnWidths.name,
  },
  progress: {
    id: 'experimentProgress',
    renderer: (record: FlatRun) => {
      const percentage = floatToPercent(record.experiment?.progress ?? 0, 0);

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
      copyData: String(record.experiment?.resourcePool),
      data: { kind: TEXT_CELL },
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
      const sMetric = record.experiment?.searcherMetric ?? '';
      return {
        allowOverlay: false,
        copyData: sMetric,
        data: { kind: TEXT_CELL },
        kind: GridCellKind.Custom,
      };
    },
    title: 'Searcher Metric',
    tooltip: () => undefined,
    width: columnWidths.searcherMetric,
  },
  searcherMetricsVal: {
    id: 'searcherMetricsVal',
    isNumerical: true,
    renderer: (record: FlatRun) => {
      const sMetricValue = record.searcherMetricValue;

      return {
        allowOverlay: false,
        copyData: sMetricValue
          ? typeof sMetricValue === 'number'
            ? humanReadableNumber(sMetricValue)
            : sMetricValue
          : '',
        data: { kind: TEXT_CELL },
        kind: GridCellKind.Custom,
      };
    },
    title: 'Searcher Metric Value',
    tooltip: () => undefined,
    width: columnWidths.searcherMetricsVal,
  },
  searcherType: {
    id: 'searcherType',
    renderer: (record: FlatRun) => ({
      allowOverlay: false,
      copyData: String(record.experiment?.searcherType),
      data: { kind: TEXT_CELL },
      kind: GridCellKind.Custom,
    }),
    title: 'Searcher Type',
    tooltip: () => undefined,
    width: columnWidths.searcherType,
  },
  startTime: {
    id: 'startTime',
    isNumerical: true,
    renderer: (record: FlatRun) => ({
      allowOverlay: false,
      copyData: getTimeInEnglish(new Date(record.startTime)),
      data: { kind: TEXT_CELL },
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
      copyData: capitalize(record.state),
      data: {
        appTheme,
        kind: STATE_CELL,
        state: getCellStateFromExperimentState(record.state),
      },
      kind: GridCellKind.Custom,
    }),
    themeOverride: { cellHorizontalPadding: 13 },
    title: 'State',
    tooltip: (record: FlatRun) => capitalize(record.state),
    width: columnWidths.state,
  },
  tags: {
    id: 'tags',
    renderer: (record: FlatRun) => ({
      allowOverlay: true,
      copyData: record.labels?.join(', ') ?? '',
      data: {
        kind: TAGS_CELL,
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
          kind: USER_AVATAR_CELL,
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

export const defaultColumnWidths: Partial<Record<RunColumn, number>> = {
  checkpointCount: 120,
  checkpointSize: 110,
  duration: 86,
  experimentDescription: 148,
  experimentProgress: 65,
  externalExperimentId: 160,
  externalRunId: 130,
  forkedFrom: 86,
  id: 50,
  name: 290,
  parentArchived: 80,
  resourcePool: 140,
  searcherMetric: 120,
  searcherMetricsVal: 120,
  searcherType: 120,
  startTime: 118,
  state: 60,
  tags: 106,
  user: 50,
};
