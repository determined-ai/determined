import { useCallback } from 'react';

import { generateContext } from 'contexts';
import { RestApiState } from 'hooks/useRestApi';
import { getAgents } from 'services/api';
import { Agent } from 'types';

import ClusterOverview from './ClusterOverview';

const initialState = {
  errorCount: 0,
  hasLoaded: false,
  isLoading: false,
};

export const Agents = generateContext<RestApiState<Agent[]>>({
  initialState: initialState,
  name: 'Agents',
});

export const useFetchAgents = (canceler: AbortController): () => Promise<void> => {
  const setAgents = Agents.useActionContext();
  const setOverview = ClusterOverview.useActionContext();

  return useCallback(async (): Promise<void> => {
    try {
      const agentsResponse = await getAgents({ signal: canceler.signal });
      setAgents({
        type: Agents.ActionType.Set,
        value: {
          data: agentsResponse,
          errorCount: 0,
          hasLoaded: true,
          isLoading: false,
        },
      });
      setOverview({ type: ClusterOverview.ActionType.SetAgents, value: agentsResponse });
    } catch (e) {}
  }, [ canceler, setAgents, setOverview ]);
};

export default Agents;
