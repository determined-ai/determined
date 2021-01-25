import { generateContext } from 'contexts';
import { Agent, ClusterOverview, ResourceType } from 'types';
import { clone } from 'utils/data';
import { percent } from 'utils/number';

enum ActionType {
  Reset,
  Set,
  SetAgents,
}

type State = ClusterOverview;

type Action =
  | { type: ActionType.Reset }
  | { type: ActionType.Set; value: State }
  | { type: ActionType.SetAgents; value: Agent[] }

const defaultResourceTally = { allocation:0, available: 0, total: 0 };

const defaultClusterOverview: ClusterOverview = {
  [ResourceType.CPU]: clone(defaultResourceTally),
  [ResourceType.GPU]: clone(defaultResourceTally),
  [ResourceType.ALL]: clone(defaultResourceTally),
  [ResourceType.UNSPECIFIED]: clone(defaultResourceTally),
};

export const agentsToOverview = (agents: Agent[]): ClusterOverview => {
  // Deep clone for render detection.
  const overview = clone(defaultClusterOverview) as ClusterOverview;

  agents.forEach(agent => {
    agent.resources
      .filter(resource => resource.enabled)
      .forEach(resource => {
        const isResourceFree = resource.container == null;
        const availableResource = isResourceFree ? 1 : 0;
        overview[resource.type].available += availableResource;
        overview[resource.type].total++;
        overview[ResourceType.ALL].available += availableResource;
        overview[ResourceType.ALL].total++;
      });
  });

  for (const key in overview) {
    const rt = key as ResourceType;
    overview[rt].allocation = overview[rt].total !== 0 ?
      percent((overview[rt].total - overview[rt].available) / overview[rt].total) : 0;
  }

  return overview;
};

const reducer = (state: State, action: Action): State => {
  switch (action.type) {
    case ActionType.Reset:
      return defaultClusterOverview;
    case ActionType.Set:
      return action.value;
    case ActionType.SetAgents:
      return agentsToOverview(action.value);
    default:
      return state;
  }
};

const contextProvider = generateContext<ClusterOverview, Action>({
  initialState: defaultClusterOverview,
  name: 'ClusterOverview',
  reducer,
});

export default { ...contextProvider, ActionType };
