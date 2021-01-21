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

const defaultClusterOverview: ClusterOverview = {
  [ResourceType.CPU]: { available: 0, total: 0 },
  [ResourceType.GPU]: { available: 0, total: 0 },
  allocation: 0,
  totalResources: { available: 0, total: 0 },
};

export const agentsToOverview = (agents: Agent[]): ClusterOverview => {
  // Deep clone for render detection.
  const overview = clone(defaultClusterOverview);
  const tally = { available: 0, total: 0 };

  agents.forEach(agent => {
    agent.resources
      .filter(resource => resource.enabled)
      .forEach(resource => {
        const isResourceFree = resource.container == null;
        const availableResource = isResourceFree ? 1 : 0;
        overview[resource.type].available += availableResource;
        overview[resource.type].total++;
        tally.available += availableResource;
        tally.total++;
      });
  });

  overview.totalResources = tally;
  overview.allocation = tally.total ? percent((tally.total - tally.available) / tally.total) : 0;

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
