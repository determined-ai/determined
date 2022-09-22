/* eslint-disable sort-keys-fix/sort-keys-fix */
import {
  DarkLight,
  getCssVar,
  themeDarkDetermined,
  themeDarkHpe,
  themeLightDetermined,
  themeLightHpe,
} from 'shared/themes';
import {
  BrandingType,
  CheckpointState,
  CommandState,
  JobState,
  ResourceState,
  RunState,
  RunStateValue,
  SlotState,
  WorkspaceState,
} from 'types';

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
  [RunState.ACTIVE]: 'active',
  [RunState.CANCELED]: 'inactive',
  [RunState.COMPLETED]: 'success',
  [RunState.DELETED]: 'critical',
  [RunState.DELETING]: 'critical',
  [RunState.DELETE_FAILED]: 'critical',
  [RunState.ERROR]: 'critical',
  [RunState.PAUSED]: 'warning',
  [RunState.STOPPING_CANCELED]: 'inactive',
  [RunState.STOPPING_COMPLETED]: 'success',
  [RunState.STOPPING_ERROR]: 'critical',
  [RunState.STOPPING_KILLED]: 'killed',
  [RunState.UNSPECIFIED]: 'inactive',
  [RunState.QUEUED]: 'warning',
  [RunState.PULLING]: 'pending',
  [RunState.STARTING]: 'pending',
  [RunState.RUNNING]: 'active',
  [CommandState.Waiting]: 'inactive',
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

export type StateOfUnion =
  | RunStateValue
  | CommandState
  | ResourceState
  | CheckpointState
  | SlotState
  | JobState
  | WorkspaceState;

export const getStateColorCssVar = (
  state: StateOfUnion | undefined,
  options: { isOn?: boolean; strongWeak?: 'strong' | 'weak' } = {},
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
