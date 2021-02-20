import React, { useCallback, useEffect, useState } from 'react';

import Agents from 'contexts/Agents';
import ClusterOverview from 'contexts/ClusterOverview';
import Users from 'contexts/Users';
import usePolling from 'hooks/usePolling';
import useRestApi from 'hooks/useRestApi';
import { getAgents, getUsers } from 'services/api';
import { EmptyParams } from 'services/types';
import { DetailedUser } from 'types';

const AppContexts: React.FC = () => {
  const [ canceler ] = useState(new AbortController());
  const setUsers = Users.useActionContext();
  const setAgents = Agents.useActionContext();
  const setOverview = ClusterOverview.useActionContext();
  const [ usersResponse, triggerUsersRequest ] =
    useRestApi<EmptyParams, DetailedUser[]>(getUsers, {});

  const fetchAgents = useCallback(async (): Promise<void> => {
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

  const fetchUsers = useCallback((): void => {
    triggerUsersRequest({ url: '/users' });
  }, [ triggerUsersRequest ]);

  const fetchAll = useCallback((): void => {
    fetchAgents();
  }, [
    fetchAgents,
  ]);

  usePolling(fetchAll);
  usePolling(fetchUsers, { delay: 60000 });

  useEffect(() => {
    setUsers({ type: Users.ActionType.Set, value: usersResponse });
  }, [ usersResponse, setUsers ]);

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  return <React.Fragment />;
};

export default AppContexts;
