import { CellClickedEventArgs, GridCellKind } from '@glideapps/glide-data-grid';
import { getColor, getInitials } from 'hew/Avatar';
import { ColumnDef, ColumnDefs, DEFAULT_COLUMN_WIDTH } from 'hew/DataGrid/columns';
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
import { handleEmptyCell } from 'utils/table';
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
  columnWidths: Record<string, number>;
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
    width: columnWidths.archived,
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
    width: columnWidths.checkpointCount,
  },
  checkpointSize: {
    id: 'checkpointSize',
    renderer: (record: ExperimentWithTrial) =>
      handleEmptyCell(record.experiment.checkpointSize, (data) => ({
        allowOverlay: false,
        copyData: humanReadableBytes(data),
        data: { kind: TEXT_CELL },
        kind: GridCellKind.Custom,
      })),
    title: 'Checkpoint Size',
    tooltip: () => undefined,
    width: columnWidths.checkpointSize,
  },
  description: {
    id: 'description',
    renderer: (record: ExperimentWithTrial) =>
      handleEmptyCell(
        record.experiment.description,
        (data) => ({
          allowOverlay: false,
          copyData: String(data),
          data: { kind: TEXT_CELL },
          kind: GridCellKind.Custom,
        }),
        false,
      ),
    title: 'Description',
    tooltip: () => undefined,
    width: columnWidths.description,
  },
  duration: {
    id: 'duration',
    renderer: (record: ExperimentWithTrial) =>
      handleEmptyCell(record.experiment.duration, () => ({
        allowOverlay: false,
        copyData: getDurationInEnglish(record.experiment),
        data: { kind: TEXT_CELL },
        kind: GridCellKind.Custom,
      })),
    title: 'Duration',
    tooltip: () => undefined,
    width: columnWidths.duration,
  },
  externalExperimentId: {
    id: 'externalExperimentId',
    renderer: (record: ExperimentWithTrial) =>
      handleEmptyCell(record.experiment.externalExperimentId, (data) => ({
        allowOverlay: false,
        copyData: data,
        data: { kind: TEXT_CELL },
        kind: GridCellKind.Custom,
      })),
    title: 'External Search ID',
    tooltip: () => undefined,
    width: columnWidths.externalExperimentId,
  },
  externalTrialId: {
    id: 'externalTrialId',
    renderer: (record: ExperimentWithTrial) =>
      handleEmptyCell(record.experiment.externalTrialId, (data) => ({
        allowOverlay: false,
        copyData: data,
        data: { kind: TEXT_CELL },
        kind: GridCellKind.Custom,
      })),
    title: 'External Trial ID',
    tooltip: () => undefined,
    width: columnWidths.externalTrialId,
  },
  forkedFrom: {
    id: 'forkedFrom',
    renderer: (record: ExperimentWithTrial) =>
      handleEmptyCell(record.experiment.forkedFrom, (data) => ({
        allowOverlay: false,
        copyData: String(record.experiment.forkedFrom),
        cursor: 'pointer',
        data: {
          kind: LINK_CELL,
          link: {
            href: paths.experimentDetails(data),
            title: String(record.experiment.forkedFrom),
          },
          navigateOn: 'click',
          underlineOffset: 6,
        },
        kind: GridCellKind.Custom,
        onClick: (e: CellClickedEventArgs) => {
          handlePath(e as unknown as AnyMouseEvent, {
            path: String(record.experiment.forkedFrom),
          });
        },
        readonly: true,
      })),
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
        kind: LINK_CELL,
        link: {
          href: paths.searchDetails(record.experiment.id),
          title: String(record.experiment.id),
        },
        navigateOn: 'click',
        onClick: (e: CellClickedEventArgs) => {
          handlePath(e as unknown as AnyMouseEvent, {
            path: paths.searchDetails(record.experiment.id),
          });
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
    renderer: (record: ExperimentWithTrial) => ({
      allowOverlay: false,
      copyData: String(record.experiment.name),
      cursor: 'pointer',
      data: {
        kind: LINK_CELL,
        link: {
          href: paths.searchDetails(record.experiment.id),
          title: String(record.experiment.name),
          unmanaged: record.experiment.unmanaged,
        },
        navigateOn: 'click',
        onClick: (e: CellClickedEventArgs) => {
          handlePath(e as unknown as AnyMouseEvent, {
            path: paths.searchDetails(record.experiment.id),
          });
        },
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
    renderer: (record: ExperimentWithTrial) => ({
      allowOverlay: false,
      data: record.experiment.numTrials,
      displayData: String(record.experiment.numTrials),
      kind: GridCellKind.Number,
    }),
    title: 'Runs',
    tooltip: () => undefined,
    width: columnWidths.numTrials,
  },
  progress: {
    id: 'progress',
    renderer: (record: ExperimentWithTrial) =>
      handleEmptyCell(record.experiment.progress, (data) => ({
        allowOverlay: false,
        data: floatToPercent(data, 0),
        displayData: floatToPercent(data, 0),
        kind: GridCellKind.Text,
      })),
    title: 'Progress',
    tooltip: () => undefined,
    width: columnWidths.progress,
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
    width: columnWidths.resourcePool,
  },
  searcherMetric: {
    id: 'searcherMetric',
    renderer: (record: ExperimentWithTrial) =>
      handleEmptyCell(record.experiment.searcherMetric, (data) => ({
        allowOverlay: false,
        copyData: data,
        data: { kind: TEXT_CELL },
        kind: GridCellKind.Custom,
      })),
    title: 'Searcher Metric',
    tooltip: () => undefined,
    width: columnWidths.searcherMetric,
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
    width: columnWidths.searcherType,
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
        kind: STATE_CELL,
        state: getCellStateFromExperimentState(record.experiment.state),
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
        kind: TAGS_CELL,
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
    width: columnWidths.user,
  },
});

export const searcherMetricsValColumn = (columnWidth?: number): ColumnDef<ExperimentWithTrial> => {
  return {
    id: 'searcherMetricsVal',
    renderer: (record: ExperimentWithTrial) =>
      handleEmptyCell(record.bestTrial?.searcherMetricsVal, (data) => ({
        allowOverlay: false,
        copyData: humanReadableNumber(data),
        data: { kind: TEXT_CELL },
        kind: GridCellKind.Custom,
      })),
    title: 'Searcher Metric Value',
    tooltip: () => undefined,
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
