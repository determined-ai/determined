import { useCallback } from 'react';

import { generateContext } from 'contexts';
import { getAgents } from 'services/api';
import { Agent } from 'types';
import { isEqual } from 'utils/data';

import ClusterOverview from './ClusterOverview';

const Agents = generateContext<Agent[] | undefined>({
  initialState: undefined,
  name: 'Agents',
});

export const useFetchAgents = (canceler: AbortController): () => Promise<void> => {
  const agents = Agents.useStateContext();
  const setAgents = Agents.useActionContext();
  const setOverview = ClusterOverview.useActionContext();

  return useCallback(async (): Promise<void> => {
    try {
      const agentsResponse = await getAgents({ signal: canceler.signal });

      // Checking for changes before making an update call.
      if (!isEqual(agents, agentsResponse)) {
        setAgents({ type: Agents.ActionType.Set, value: agentsResponse });
        setOverview({ type: ClusterOverview.ActionType.SetAgents, value: agentsResponse });
      }
    } catch (e) {}
  }, [ agents, canceler, setAgents, setOverview ]);
};

export default Agents;
