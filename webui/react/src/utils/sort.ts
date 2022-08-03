import { alphaNumericSorter } from '../shared/utils/sort';
import { CommandState, HpImportance, MetricName, MetricType, RunState, State } from '../types';

const runStateSortValues: Record<RunState, number> = {
    [RunState.Active]: 0,
    [RunState.Paused]: 1,
    [RunState.StoppingError]: 2,
    [RunState.Errored]: 3,
    [RunState.StoppingCompleted]: 4,
    [RunState.Completed]: 5,
    [RunState.StoppingCanceled]: 6,
    [RunState.Canceled]: 7,
    [RunState.Deleted]: 7,
    [RunState.Deleting]: 7,
    [RunState.DeleteFailed]: 7,
    [RunState.Unspecified]: 8,
};
const commandStateSortValues: Record<CommandState, number> = {
    [CommandState.Pending]: 0,
    [CommandState.Assigned]: 1,
    [CommandState.Pulling]: 2,
    [CommandState.Starting]: 3,
    [CommandState.Running]: 4,
    [CommandState.Terminating]: 5,
    [CommandState.Terminated]: 6,
};
export const commandStateSorter = (a: CommandState, b: CommandState): number => {
    return commandStateSortValues[a] - commandStateSortValues[b];
};
export const hpImportanceSorter = (a: string, b: string, hpImportance: HpImportance): number => {
    const aValue = hpImportance[a];
    const bValue = hpImportance[b];
    if (aValue < bValue) return 1;
    if (aValue > bValue) return -1;
    return 0;
};
/*
 * Sort the metric names by having the validation metrics come first followed by training metrics.
 * Within each type of metric, sort in the order they appear in the `MetricNames` array.
 * Within the respective type of metrics, `MetricNames` is currently sorted alphanumerically.
 */
export const metricNameSorter = (a: MetricName, b: MetricName): number => {
    const isAValidation = a.type === MetricType.Validation;
    const isBValidation = b.type === MetricType.Validation;
    if (isAValidation && !isBValidation) return -1;
    if (isBValidation && !isAValidation) return 1;
    return alphaNumericSorter(a.name, b.name);
};
export const runStateSorter = (a: RunState, b: RunState): number => {
    return runStateSortValues[a] - runStateSortValues[b];
};
export const taskStateSorter = (a: State, b: State): number => {
    // FIXME this is O(n) we can do it in constant time.
    // What is the right typescript way of doing it?
    const aValue = Object.values(RunState).includes(a as RunState) ?
        runStateSortValues[a as RunState] : commandStateSortValues[a as CommandState];
    const bValue = Object.values(RunState).includes(b as RunState) ?
        runStateSortValues[b as RunState] : commandStateSortValues[b as CommandState];
    return aValue - bValue;
};
