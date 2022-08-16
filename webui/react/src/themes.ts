/* eslint-disable sort-keys-fix/sort-keys-fix */
import {
  DarkLight,
  getCssVar,
  themeDarkDetermined,
  themeDarkHpe,
  themeLightDetermined,
  themeLightHpe,
} from 'shared/themes';
import { BrandingType, CheckpointState, CommandState, JobState, ResourceState, RunState,
  SlotState, WorkspaceState } from 'types';

/*
 * Where did we get our sizes from?
 * https://www.quora.com/What-is-the-difference-among-big-large-huge-enormous-and-giant
 */
export enum ShirtSize {
  small = 'small',
  medium = 'medium',
  large = 'large',
}

const stateColorMapping = {
  [RunState.Active]: 'active',
  [RunState.Canceled]: 'inactive',
  [RunState.Completed]: 'success',
  [RunState.Deleted]: 'critical',
  [RunState.Deleting]: 'critical',
  [RunState.DeleteFailed]: 'critical',
  [RunState.Errored]: 'critical',
  [RunState.Paused]: 'warning',
  [RunState.StoppingCanceled]: 'inactive',
  [RunState.StoppingCompleted]: 'success',
  [RunState.StoppingError]: 'critical',
  [RunState.Unspecified]: 'inactive',
  [CommandState.Pending]: 'warning',
  [CommandState.Assigned]: 'warning',
  [CommandState.Pulling]: 'active',
  [CommandState.Starting]: 'active',
  [CommandState.Running]: 'active',
  [CommandState.Terminating]: 'inactive',
  [CommandState.Terminated]: 'inactive',
  [ResourceState.Unspecified]: 'inactive',
  [ResourceState.Running]: 'active',
  [ResourceState.Assigned]: 'pending',
  [ResourceState.Pulling]: 'pending',
  [ResourceState.Starting]: 'pending',
  [ResourceState.Warm]: 'free',
  [SlotState.Free]: 'free',
  [SlotState.Pending]: 'pending',
  [SlotState.Running]: 'active',
  [SlotState.Potential]: 'potential',
  [JobState.SCHEDULED]: 'active',
  [JobState.SCHEDULEDBACKFILLED]: 'active',
  [JobState.QUEUED]: 'warning',
};

export type StateOfUnion = RunState | CommandState | ResourceState | CheckpointState |
SlotState | JobState | WorkspaceState

export const getStateColorCssVar = (
  state: StateOfUnion | undefined,
  options: { isOn?: boolean, strongWeak?: 'strong' | 'weak' } = {},
): string => {
  const name = state ? stateColorMapping[state] : 'active';
  const on = options.isOn ? '-on' : '';
  const strongWeak = options.strongWeak ? `-${options.strongWeak}` : '';
  return `var(--theme-status-${name}${on}${strongWeak})`;
};

export const getStateColor = (state: StateOfUnion | undefined): string => {
  return getCssVar(getStateColorCssVar(state));
};

export default {
  [BrandingType.Determined]: {
    [DarkLight.Dark]: themeDarkDetermined,
    [DarkLight.Light]: themeLightDetermined,
  },
  [BrandingType.HPE]: {
    [DarkLight.Dark]: themeDarkHpe,
    [DarkLight.Light]: themeLightHpe,
  },
};
