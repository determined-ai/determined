import { FilterFormSetWithoutId, Operator } from 'components/FilterForm/components/type';
import {
  cancellableRunStates,
  deletableRunStates,
  erroredRunStates,
  killableRunStates,
  pausableRunStates,
  terminalRunStates,
} from 'constants/states';
import {
  AnyTask,
  BulkExperimentItem,
  ContinuableNonSingleSearcherName,
  ExperimentAction,
  ExperimentBase,
  ExperimentPermissionsArgs,
  ExperimentSearcherName,
  FullExperimentItem,
  Hyperparameters,
  HyperparameterType,
  Project,
  ProjectExperiment,
  RawJson,
  RunState,
  SelectionType,
  TrialDetails,
  TrialHyperparameters,
  WorkspacePermissionsArgs,
} from 'types';
import { deletePathList, getPathList, isNumber, setPathList, unflattenObject } from 'utils/data';

type ExperimentChecker = (
  experiment: ProjectExperiment,
  trial?: TrialDetails,
  erroredTrialCount?: number,
) => boolean;

type ExperimentPermissionSet = {
  canCreateExperiment: (arg0: WorkspacePermissionsArgs) => boolean;
  canDeleteExperiment: (arg0: ExperimentPermissionsArgs) => boolean;
  canModifyExperiment: (arg0: WorkspacePermissionsArgs) => boolean;
  canModifyExperimentMetadata: (arg0: WorkspacePermissionsArgs) => boolean;
  canMoveExperiment: (arg0: ExperimentPermissionsArgs) => boolean;
  canViewExperimentArtifacts: (arg0: WorkspacePermissionsArgs) => boolean;
};

export const FULL_CONFIG_BUTTON_TEXT = 'Show Full Config';
export const SIMPLE_CONFIG_BUTTON_TEXT = 'Show Simple Config';

// Differentiate Experiment from Task.
export const isExperiment = <T extends BulkExperimentItem | FullExperimentItem>(
  obj: AnyTask | T,
): obj is T => {
  return 'hyperparameters' in obj && 'archived' in obj;
};

export const isSingleTrialExperiment = (experiment: ExperimentBase): boolean => {
  return (
    experiment?.config.searcher.name === ExperimentSearcherName.Single ||
    experiment?.config.searcher.max_trials === 1
  );
};

export const trialHParamsToExperimentHParams = (
  trialHParams: TrialHyperparameters,
): Hyperparameters => {
  const hParams = Object.keys(trialHParams).reduce(
    (acc, key) => {
      return {
        ...acc,
        [key]: {
          type: HyperparameterType.Constant,
          val: trialHParams[key] as number,
        },
      };
    },
    {} as Record<keyof TrialHyperparameters, unknown>,
  );
  return unflattenObject(hParams) as Hyperparameters;
};

/* Experiment Config */

const stepRemovalTranslations = [
  { oldName: 'min_checkpoint_period' },
  { oldName: 'min_validation_period' },
  { newName: 'searcher.budget', oldName: 'searcher.step_budget' },
  { newName: 'searcher.length_per_round', oldName: 'searcher.steps_per_round' },
  { newName: 'searcher.max_length', oldName: 'searcher.max_steps' },
  { newName: 'searcher.max_length', oldName: 'searcher.target_trial_steps' },
];

const getLengthFromStepCount = (config: RawJson, stepCount: number): [string, number] => {
  const DEFAULT_BATCHES_PER_STEP = 100;
  // provide backward compat for step count
  const batchesPerStep = config.batches_per_step || DEFAULT_BATCHES_PER_STEP;
  return ['batches', stepCount * batchesPerStep];
};

// Add opportunistic backward compatibility to old configs.
export const upgradeConfig = (config: RawJson): RawJson => {
  const newConfig = structuredClone(config);

  stepRemovalTranslations.forEach((translation) => {
    const oldPath = translation.oldName.split('.');
    const curValue = getPathList<undefined | null | number | unknown>(newConfig, oldPath);
    if (curValue === undefined) return;
    if (curValue === null) deletePathList(newConfig, oldPath);
    if (isNumber(curValue)) {
      const [key, count] = getLengthFromStepCount(newConfig, curValue);
      const newPath = (translation.newName || translation.oldName).split('.');
      setPathList(newConfig, newPath, { [key]: count });

      if (translation.newName) deletePathList(newConfig, oldPath);
    }
  });

  delete newConfig.batches_per_step;

  return newConfig;
};

/* eslint-disable @typescript-eslint/no-unused-vars */
export const isExperimentModifiable = (experiment: ProjectExperiment): boolean =>
  !experiment.archived && !experiment.parentArchived;

/* eslint-disable @typescript-eslint/no-unused-vars */
export const isExperimentForkable = (experiment: ProjectExperiment): boolean =>
  !experiment.parentArchived;

export const alwaysTrueExperimentChecker = (_experiment: ProjectExperiment): boolean => true;

const resumableSearcherTypes: ExperimentSearcherName[] = [
  ExperimentSearcherName.Grid,
  ExperimentSearcherName.Random,
];

// Single trial experiment or trial of multi trial experiment can be continued.
export const canExperimentContinueTrial = (
  experiment: ProjectExperiment,
  trial?: TrialDetails,
): boolean => {
  if (experiment.archived || experiment.parentArchived) return false;
  if ((!!trial || experiment?.numTrials === 1) && terminalRunStates.has(experiment.state))
    return true;
  // multi trial experiment can continue if it's terminated but not completed, and the searcher type is grid or random.
  if (
    ContinuableNonSingleSearcherName.has(experiment.config?.searcher.name || 'custom') &&
    erroredRunStates.has(experiment.state)
  )
    return true;
  return false;
};

const experimentCheckers: Record<ExperimentAction, ExperimentChecker> = {
  /**
   * for internal use: the typing ensures that checkers
   * are defined for every ExperimentAction
   * we expose the functions below as convenient wrappers
   */
  [ExperimentAction.Activate]: (experiment) => experiment.state === RunState.Paused,

  [ExperimentAction.Archive]: (experiment) =>
    !experiment.parentArchived && !experiment.archived && terminalRunStates.has(experiment.state),

  [ExperimentAction.Cancel]: (experiment) => cancellableRunStates.has(experiment.state),

  [ExperimentAction.CompareTrials]: alwaysTrueExperimentChecker,

  [ExperimentAction.ContinueTrial]: canExperimentContinueTrial,

  [ExperimentAction.Delete]: (experiment) => deletableRunStates.has(experiment.state),

  [ExperimentAction.DownloadCode]: (experiment) => experiment.modelDefinitionSize !== 0,

  [ExperimentAction.Edit]: (experiment) => !experiment?.parentArchived && !experiment?.archived,

  [ExperimentAction.HyperparameterSearch]: alwaysTrueExperimentChecker,

  [ExperimentAction.Fork]: isExperimentForkable,

  [ExperimentAction.Retry]: (experiment, _, erroredTrialCount) =>
    (erroredRunStates.has(experiment.state) ||
      (experiment.state === RunState.Completed && (erroredTrialCount ?? 0) > 0)) &&
    resumableSearcherTypes.includes(experiment.searcherType as ExperimentSearcherName),

  [ExperimentAction.Kill]: (experiment) => killableRunStates.includes(experiment.state),

  [ExperimentAction.Move]: (experiment) => !experiment?.parentArchived && !experiment.archived,

  [ExperimentAction.Pause]: (experiment) => pausableRunStates.has(experiment.state),

  [ExperimentAction.OpenTensorBoard]: alwaysTrueExperimentChecker,

  [ExperimentAction.Unarchive]: (experiment) =>
    terminalRunStates.has(experiment.state) && experiment.archived,

  [ExperimentAction.ViewLogs]: alwaysTrueExperimentChecker,

  [ExperimentAction.SwitchPin]: alwaysTrueExperimentChecker,

  [ExperimentAction.RetainLogs]: alwaysTrueExperimentChecker,
};

export const canActionExperiment = (
  action: ExperimentAction,
  experiment: ProjectExperiment,
  trial?: TrialDetails,
  erroredTrialCount?: number,
): boolean => {
  return !!experiment && experimentCheckers[action](experiment, trial, erroredTrialCount);
};

export const getActionsForExperiment = (
  experiment: ProjectExperiment,
  targets: ExperimentAction[],
  permissions: ExperimentPermissionSet,
  erroredTrialCount?: number,
): ExperimentAction[] => {
  if (!experiment) return []; // redundant, for clarity
  const workspace = { id: experiment.workspaceId };
  return targets
    .filter((action) => canActionExperiment(action, experiment, undefined, erroredTrialCount))
    .filter((action) => {
      switch (action) {
        case ExperimentAction.ContinueTrial:
        case ExperimentAction.Fork:
        case ExperimentAction.HyperparameterSearch:
        case ExperimentAction.Retry:
          return (
            permissions.canViewExperimentArtifacts({ workspace }) &&
            permissions.canCreateExperiment({ workspace }) &&
            !experiment.unmanaged
          );

        case ExperimentAction.Delete:
          return permissions.canDeleteExperiment({ experiment });

        case ExperimentAction.DownloadCode:
          return permissions.canViewExperimentArtifacts({ workspace });

        case ExperimentAction.OpenTensorBoard:
          return permissions.canViewExperimentArtifacts({ workspace }) && !experiment.unmanaged;

        case ExperimentAction.Move:
          return permissions.canMoveExperiment({ experiment });

        case ExperimentAction.Edit:
          return permissions.canModifyExperimentMetadata({
            workspace: { id: experiment?.workspaceId },
          });

        case ExperimentAction.RetainLogs:
        case ExperimentAction.Archive:
        case ExperimentAction.Unarchive:
          return permissions.canModifyExperiment({ workspace });

        case ExperimentAction.Activate:
        case ExperimentAction.Cancel:
        case ExperimentAction.Kill:
        case ExperimentAction.Pause:
          return permissions.canModifyExperiment({ workspace }) && !experiment.unmanaged;

        default:
          return true;
      }
    });
};

export const getActionsForExperimentsUnion = (
  experiments: ProjectExperiment[],
  targets: ExperimentAction[],
  permissions: ExperimentPermissionSet,
): ExperimentAction[] => {
  if (!experiments.length) return []; // redundant, for clarity
  const actionsForExperiments = experiments.map((e) =>
    getActionsForExperiment(e, targets, permissions),
  );
  return targets.filter((action) =>
    actionsForExperiments.some((experimentActions) => experimentActions.includes(action)),
  );
};

export const getProjectExperimentForExperimentItem = (
  experiment: BulkExperimentItem,
  project?: Project,
): ProjectExperiment =>
  ({
    ...experiment,
    parentArchived: !!project?.archived,
    projectId: project?.id ?? 0,
    projectName: project?.name,
    projectOwnerId: project?.userId ?? 0,
    workspaceId: project?.workspaceId ?? 0,
    workspaceName: project?.workspaceName,
  }) as ProjectExperiment;

const runStateSortOrder: RunState[] = [
  RunState.Active,
  RunState.Running,
  RunState.Paused,
  RunState.Starting,
  RunState.Pulling,
  RunState.Queued,
  RunState.StoppingError,
  RunState.Error,
  RunState.StoppingCompleted,
  RunState.Completed,
  RunState.StoppingCanceled,
  RunState.Canceled,
  RunState.DeleteFailed,
  RunState.Deleting,
  RunState.Deleted,
  RunState.Unspecified,
];

export const runStateSortValues: Map<RunState, number> = new Map(
  runStateSortOrder.map((state, idx) => [state, idx]),
);

export const runStateSorter = (a: RunState, b: RunState): number => {
  return (runStateSortValues.get(a) || 0) - (runStateSortValues.get(b) || 0);
};

export const getExperimentName = (config: RawJson): string => {
  return config.name || '';
};

// For unitless searchers, this will return undefined.
export const getMaxLengthType = (config: RawJson): string | undefined => {
  return (Object.keys(config.searcher?.max_length || {}) || [])[0];
};

export const getMaxLengthValue = (config: RawJson): number => {
  const value = (Object.keys(config.searcher?.max_length || {}) || [])[0];
  return value
    ? parseInt(config.searcher?.max_length[value])
    : parseInt(config.searcher?.max_length);
};

export const trialContinueConfig = (
  experimentConfig: RawJson,
  trialHparams: TrialHyperparameters,
  trialId: number,
  workspaceName: string,
  projectName: string,
): RawJson => {
  const newConfig = structuredClone(experimentConfig);
  return {
    ...newConfig,
    hyperparameters: trialHParamsToExperimentHParams(trialHparams),
    project: projectName,
    searcher: {
      max_length: experimentConfig.searcher.max_length,
      metric: experimentConfig.searcher.metric,
      name: 'single',
      smaller_is_better: experimentConfig.searcher.smaller_is_better,
      source_trial_id: trialId,
    },
    workspace: workspaceName,
  };
};

const idToFilter = (operator: Operator, id: number) =>
  ({
    columnName: 'id',
    kind: 'field',
    location: 'LOCATION_TYPE_EXPERIMENT',
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
          location: 'LOCATION_TYPE_EXPERIMENT',
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
