import { useCallback } from 'react';

import { generateContext } from 'contexts';
import { getAgents } from 'services/api';
import { Agent } from 'types';

import ClusterOverview from './ClusterOverview';

const Agents = generateContext<Agent[] | undefined>({
  initialState: undefined,
  name: 'Agents',
});

export const useFetchAgents = (canceler: AbortController): () => Promise<void> => {
  const setAgents = Agents.useActionContext();
  const setOverview = ClusterOverview.useActionContext();

  return useCallback(async (): Promise<void> => {
    try {
      const agentsResponse = await getAgents({ signal: canceler.signal });
      setAgents({ type: Agents.ActionType.Set, value: agentsResponse });
      setOverview({ type: ClusterOverview.ActionType.SetAgents, value: agentsResponse });
    } catch (e) {}
  }, [ canceler, setAgents, setOverview ]);
};

export default Agents;
