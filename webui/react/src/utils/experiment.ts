import {
  cancellableRunStates,
  deletableRunStates,
  killableRunStates,
  pausableRunStates,
  terminalRunStates,
} from 'constants/states';
import {
  AnyTask,
  DetailedUser,
  ExperimentAction,
  ExperimentBase,
  ExperimentItem,
  ExperimentSearcherName,
  Hyperparameters,
  HyperparameterType,
  Project,
  ProjectExperiment,
  RawJson,
  RunState,
  TrialHyperparameters,
} from 'types';

import { clone, deletePathList, getPathList, isNumber, setPathList, unflattenObject } from './data';

type ExperimentChecker = (experiment: ProjectExperiment, user?: DetailedUser) => boolean

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
  return [ 'batches', stepCount * batchesPerStep ];
};

// Add opportunistic backward compatibility to old configs.
export const upgradeConfig = (config: RawJson): RawJson => {
  const newConfig = clone(config);

  stepRemovalTranslations.forEach((translation) => {
    const oldPath = translation.oldName.split('.');
    const curValue = getPathList<undefined | null | number | unknown>(newConfig, oldPath);
    if (curValue === undefined) return;
    if (curValue === null) deletePathList(newConfig, oldPath);
    if (isNumber(curValue)) {
      const [ key, count ] = getLengthFromStepCount(newConfig, curValue);
      const newPath = (translation.newName || translation.oldName).split('.');
      setPathList(newConfig, newPath, { [key]: count });

      if (translation.newName) deletePathList(newConfig, oldPath);
    }
  });

  delete newConfig.batches_per_step;

  return newConfig;
};

/* eslint-disable @typescript-eslint/no-unused-vars */
export const isExperimentModifiable = (
  experiment: ProjectExperiment,
  user?: DetailedUser,
): boolean => !experiment.archived && !experiment.parentArchived;

export const alwaysTrueExperimentChecker = (
  experiment: ProjectExperiment,
  user?: DetailedUser,
): boolean => true;

export const alwaysFalseExperimentChecker = (
  experiment: ProjectExperiment,
  user?: DetailedUser,
): boolean => false;

const experimentCheckers: Record<ExperimentAction, ExperimentChecker> = {
  /**
   * for internal use: the typing ensures that checkers
   * are defined for every ExperimentAction
   * we expose the functions below as convenient wrappers
   */
  [ExperimentAction.Activate]: (experiment, user) => experiment.state === RunState.Paused,

  [ExperimentAction.Archive]: (experiment, user) =>
    !experiment.parentArchived && !experiment.archived && terminalRunStates.has(experiment.state),

  [ExperimentAction.Cancel]: (experiment, user) =>
    cancellableRunStates.has(experiment.state),

  [ExperimentAction.CompareTrials]: alwaysTrueExperimentChecker,

  [ExperimentAction.ContinueTrial]: isExperimentModifiable,

  [ExperimentAction.Delete]: (experiment, user) =>
    !!user && (user.isAdmin || user.username === experiment.username)
      ? deletableRunStates.has(experiment.state)
      : false,

  [ExperimentAction.DownloadCode]: alwaysTrueExperimentChecker,

  [ExperimentAction.Fork]: isExperimentModifiable,

  [ExperimentAction.Kill]: (experiment, user) =>
    killableRunStates.includes(experiment.state),

  [ExperimentAction.Move]: (experiment, user) =>
    !!user &&
    (user.isAdmin || user.username === experiment.username) &&
    !experiment?.parentArchived &&
    !experiment.archived,

  [ExperimentAction.Pause]: (experiment, user) => pausableRunStates.has(experiment.state),

  [ExperimentAction.OpenTensorBoard]: alwaysTrueExperimentChecker,

  [ExperimentAction.Unarchive]: (experiment, user) =>
    terminalRunStates.has(experiment.state) && experiment.archived,

  [ExperimentAction.ViewLogs]: alwaysTrueExperimentChecker,
};

export const canUserActionExperiment = (
  user: DetailedUser | undefined,
  action: ExperimentAction,
  experiment: ProjectExperiment,
): boolean => !!experiment && experimentCheckers[action](experiment, user);

export const getActionsForExperiment = (
  experiment: ProjectExperiment,
  targets: ExperimentAction[],
  user?: DetailedUser,
): ExperimentAction[] => {
  if (!experiment) return []; // redundant, for clarity
  return targets.filter(action => canUserActionExperiment(user, action, experiment));
};

export const getActionsForExperimentsUnion = (
  experiments: ProjectExperiment[],
  targets: ExperimentAction[],
  user?: DetailedUser,
): ExperimentAction[] => {
  if (!experiments.length) return []; // redundant, for clarity
  const actionsForExperiments = experiments.map(e => getActionsForExperiment(e, targets, user));
  return targets.filter((action) =>
    actionsForExperiments.some(experimentActions => experimentActions.includes(action)));
};

export const getActionsForExperimentsIntersection = (
  experiments: ProjectExperiment[],
  targets: ExperimentAction[],
  user?: DetailedUser,
): ExperimentAction[] => {
  if (!experiments.length) [];
  const actionsForExperiments = experiments.map(e => getActionsForExperiment(e, targets, user));
  return targets.filter((action) =>
    actionsForExperiments.every(experimentActions => experimentActions.includes(action)));
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
    workspaceId: project?.workspaceId ?? 0,
    workspaceName: project?.workspaceName,
  } as ProjectExperiment);
