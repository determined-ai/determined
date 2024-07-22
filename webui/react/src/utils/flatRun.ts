import { FilterFormSetWithoutId, Operator } from 'components/FilterForm/components/type';
import {
  deletableRunStates,
  killableRunStates,
  pausableRunStates,
  terminalRunStates,
} from 'constants/states';
import { PermissionsHook } from 'hooks/usePermissions';
import { FlatRun, FlatRunAction, RunState, SelectionType } from 'types';

type FlatRunChecker = (flatRun: Readonly<FlatRun>) => boolean;

type FlatRunPermissionSet = Pick<
  PermissionsHook,
  'canCreateFlatRun' | 'canDeleteFlatRun' | 'canModifyFlatRun' | 'canMoveFlatRun'
>;

const flatRunCheckers: Record<FlatRunAction, FlatRunChecker> = {
  [FlatRunAction.Archive]: (flatRun) =>
    !flatRun.parentArchived && !flatRun.archived && terminalRunStates.has(flatRun.state),

  [FlatRunAction.Delete]: (flatRun) => deletableRunStates.has(flatRun.state),

  [FlatRunAction.Kill]: (flatRun) => killableRunStates.includes(flatRun.state),

  [FlatRunAction.Move]: (flatRun) => !flatRun.parentArchived && !flatRun.archived,

  [FlatRunAction.Pause]: (run) => pausableRunStates.has(run.state) && !run.experiment?.isMultitrial,

  [FlatRunAction.Resume]: (run) => run.state === RunState.Paused,

  [FlatRunAction.Unarchive]: (flatRun) => terminalRunStates.has(flatRun.state) && flatRun.archived,
};

export const canActionFlatRun = (action: FlatRunAction, flatRun: Readonly<FlatRun>): boolean => {
  return flatRunCheckers[action](flatRun);
};

export const getActionsForFlatRun = (
  flatRun: Readonly<FlatRun>,
  targets: ReadonlyArray<FlatRunAction>,
  permissions: Readonly<FlatRunPermissionSet>,
): ReadonlyArray<FlatRunAction> => {
  if (!flatRun) return []; // redundant, for clarity
  const workspace = { id: flatRun.workspaceId };
  return targets
    .filter((action) => canActionFlatRun(action, flatRun))
    .filter((action) => {
      switch (action) {
        case FlatRunAction.Delete:
          return permissions.canDeleteFlatRun({ flatRun });
        case FlatRunAction.Move:
          return permissions.canMoveFlatRun({ flatRun });
        case FlatRunAction.Archive:
        case FlatRunAction.Unarchive:
          return permissions.canModifyFlatRun({ workspace });
        case FlatRunAction.Pause:
        case FlatRunAction.Resume:
        case FlatRunAction.Kill:
          return permissions.canModifyFlatRun({ workspace }) && !flatRun.experiment?.unmanaged;
        default:
          return true;
      }
    });
};

export const getActionsForFlatRunsUnion = (
  flatRun: ReadonlyArray<Readonly<FlatRun>>,
  targets: ReadonlyArray<FlatRunAction>,
  permissions: Readonly<FlatRunPermissionSet>,
): Readonly<FlatRunAction[]> => {
  if (!flatRun.length) return [];
  const actionsForRuns = flatRun.map((run) => getActionsForFlatRun(run, targets, permissions));
  return targets.filter((action) =>
    actionsForRuns.some((runActions) => runActions.includes(action)),
  );
};

const idToFilter = (operator: Operator, id: number) =>
  ({
    columnName: 'id',
    kind: 'field',
    location: 'LOCATION_TYPE_RUN',
    operator,
    type: 'COLUMN_TYPE_NUMBER',
    value: id,
  }) as const;

export const getIdsFilter = (
  filterFormSet: FilterFormSetWithoutId,
  selection: SelectionType,
): FilterFormSetWithoutId | undefined => {
  const filterGroup: FilterFormSetWithoutId['filterGroup'] =
    selection.type === 'ALL_EXCEPT'
      ? {
          children: [
            filterFormSet.filterGroup,
            {
              children: selection.exclusions.map(idToFilter.bind(this, '!=')),
              conjunction: 'and',
              kind: 'group',
            },
          ],
          conjunction: 'and',
          kind: 'group',
        }
      : {
          children: selection.selections.map(idToFilter.bind(this, '=')),
          conjunction: 'or',
          kind: 'group',
        };

  const filter: FilterFormSetWithoutId = {
    ...filterFormSet,
    filterGroup: {
      children: [
        filterGroup,
        {
          columnName: 'searcherType',
          kind: 'field',
          location: 'LOCATION_TYPE_RUN',
          operator: '!=',
          type: 'COLUMN_TYPE_TEXT',
          value: 'single',
        } as const,
      ],
      conjunction: 'and',
      kind: 'group',
    },
  };
  return filter;
};
