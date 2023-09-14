/* eslint-disable sort-keys-fix/sort-keys-fix */
import {
  BrandingType,
  CheckpointState,
  CommandState,
  JobState,
  ResourceState,
  RunState,
  SlotState,
  ValueOf,
  WorkspaceState,
} from 'components/kit/internal/types';

import {
  DarkLight,
  getCssVar,
  themeDarkDetermined,
  themeDarkHpe,
  themeLightDetermined,
  themeLightHpe,
} from './themeUtils';

/*
 * Where did we get our sizes from?
 * https://www.quora.com/What-is-the-difference-among-big-large-huge-enormous-and-giant
 */
export const ShirtSize = {
  Small: 'small',
  Medium: 'medium',
  Large: 'large',
} as const;

export type ShirtSize = ValueOf<typeof ShirtSize>;

const stateColorMapping = {
  [RunState.Active]: 'active',
  [RunState.Canceled]: 'inactive',
  [RunState.Completed]: 'success',
  [RunState.Deleted]: 'critical',
  [RunState.Deleting]: 'critical',
  [RunState.DeleteFailed]: 'critical',
  [RunState.Error]: 'critical',
  [RunState.Paused]: 'warning',
  [RunState.StoppingCanceled]: 'inactive',
  [RunState.StoppingCompleted]: 'success',
  [RunState.StoppingError]: 'critical',
  [RunState.StoppingKilled]: 'killed',
  [CheckpointState.PartiallyDeleted]: 'warning',
  [RunState.Unspecified]: 'inactive',
  [RunState.Queued]: 'warning',
  [RunState.Pulling]: 'pending',
  [RunState.Starting]: 'pending',
  [RunState.Running]: 'active',
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
  [JobState.UNSPECIFIED]: 'inactive',
};

export type StateOfUnion =
  | RunState
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

export const themes = {
  [BrandingType.Determined]: {
    [DarkLight.Dark]: themeDarkDetermined,
    [DarkLight.Light]: themeLightDetermined,
  },
  [BrandingType.HPE]: {
    [DarkLight.Dark]: themeDarkHpe,
    [DarkLight.Light]: themeLightHpe,
  },
};
