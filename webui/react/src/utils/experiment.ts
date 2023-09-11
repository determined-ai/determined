import {
  cancellableRunStates,
  deletableRunStates,
  killableRunStates,
  pausableRunStates,
  terminalRunStates,
} from 'constants/states';
import {
  AnyTask,
  ExperimentAction,
  ExperimentBase,
  ExperimentItem,
  ExperimentPermissionsArgs,
  ExperimentSearcherName,
  Hyperparameters,
  HyperparameterType,
  Project,
  ProjectExperiment,
  RawJson,
  RunState,
  TrialDetails,
  TrialHyperparameters,
  WorkspacePermissionsArgs,
} from 'types';
import { deletePathList, getPathList, isNumber, setPathList, unflattenObject } from 'utils/data';

type ExperimentChecker = (experiment: ProjectExperiment, trial?: TrialDetails) => boolean;

type ExperimentPermissionSet = {
  canCreateExperiment: (arg0: WorkspacePermissionsArgs) => boolean;
  canDeleteExperiment: (arg0: ExperimentPermissionsArgs) => boolean;
  canModifyExperiment: (arg0: WorkspacePermissionsArgs) => boolean;
  canModifyExperimentMetadata: (arg0: WorkspacePermissionsArgs) => boolean;
  canMoveExperiment: (arg0: ExperimentPermissionsArgs) => boolean;
  canViewExperimentArtifacts: (arg0: WorkspacePermissionsArgs) => boolean;
};

// Differentiate Experiment from Task.
export const isExperiment = (obj: AnyTask | ExperimentItem): obj is ExperimentItem => {
  return 'config' in obj && 'archived' in obj;
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
  const hParams = Object.keys(trialHParams).reduce((acc, key) => {
    return {
      ...acc,
      [key]: {
        type: HyperparameterType.Constant,
        val: trialHParams[key] as number,
      },
    };
  }, {} as Record<keyof TrialHyperparameters, unknown>);
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

export const alwaysTrueExperimentChecker = (experiment: ProjectExperiment): boolean => true;

// Single trial experiment or trial of multi trial experiment can be continued.
export const canExperimentContinueTrial = (
  experiment: ProjectExperiment,
  trial?: TrialDetails,
): boolean =>
  !experiment.archived && !experiment.parentArchived && (!!trial || experiment?.numTrials === 1);

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

  [ExperimentAction.Kill]: (experiment) => killableRunStates.includes(experiment.state),

  [ExperimentAction.Move]: (experiment) => !experiment?.parentArchived && !experiment.archived,

  [ExperimentAction.Pause]: (experiment) => pausableRunStates.has(experiment.state),

  [ExperimentAction.OpenTensorBoard]: alwaysTrueExperimentChecker,

  [ExperimentAction.Unarchive]: (experiment) =>
    terminalRunStates.has(experiment.state) && experiment.archived,

  [ExperimentAction.ViewLogs]: alwaysTrueExperimentChecker,

  [ExperimentAction.SwitchPin]: alwaysTrueExperimentChecker,
};

export const canActionExperiment = (
  action: ExperimentAction,
  experiment: ProjectExperiment,
  trial?: TrialDetails,
): boolean => {
  return !!experiment && experimentCheckers[action](experiment, trial);
};

export const getActionsForExperiment = (
  experiment: ProjectExperiment,
  targets: ExperimentAction[],
  permissions: ExperimentPermissionSet,
): ExperimentAction[] => {
  if (!experiment) return []; // redundant, for clarity
  const workspace = { id: experiment.workspaceId };
  return targets
    .filter((action) => canActionExperiment(action, experiment))
    .filter((action) => {
      switch (action) {
        case ExperimentAction.ContinueTrial:
        case ExperimentAction.Fork:
        case ExperimentAction.HyperparameterSearch:
          return (
            permissions.canViewExperimentArtifacts({ workspace }) &&
            permissions.canCreateExperiment({ workspace })
          );

        case ExperimentAction.Delete:
          return permissions.canDeleteExperiment({ experiment });

        case ExperimentAction.DownloadCode:
        case ExperimentAction.OpenTensorBoard:
          return permissions.canViewExperimentArtifacts({ workspace });

        case ExperimentAction.Move:
          return permissions.canMoveExperiment({ experiment });

        case ExperimentAction.Edit:
          return permissions.canModifyExperimentMetadata({
            workspace: { id: experiment?.workspaceId },
          });

        case ExperimentAction.Activate:
        case ExperimentAction.Archive:
        case ExperimentAction.Cancel:
        case ExperimentAction.Kill:
        case ExperimentAction.Pause:
        case ExperimentAction.Unarchive:
          return permissions.canModifyExperiment({ workspace });

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
  experiment: ExperimentItem,
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
  } as ProjectExperiment);

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
