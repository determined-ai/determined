import React, { useCallback, useEffect, useState } from 'react';

import ActiveExperiments from 'contexts/ActiveExperiments';
import Agents from 'contexts/Agents';
import ClusterOverview from 'contexts/ClusterOverview';
import {
  useFetchCommands, useFetchNotebooks, useFetchShells, useFetchTensorboards,
} from 'contexts/Commands';
import Users from 'contexts/Users';
import usePolling from 'hooks/usePolling';
import useRestApi from 'hooks/useRestApi';
import { getAgents, getExperiments, getUsers } from 'services/api';
import { EmptyParams, GetExperimentsParams } from 'services/types';
import { DetailedUser, ExperimentPagination } from 'types';
import { activeRunStates } from 'utils/types';

const AppContexts: React.FC = () => {
  const [ canceler ] = useState(new AbortController());
  const setUsers = Users.useActionContext();
  const setAgents = Agents.useActionContext();
  const setActiveExperiments = ActiveExperiments.useActionContext();
  const setOverview = ClusterOverview.useActionContext();
  const [ usersResponse, triggerUsersRequest ] =
    useRestApi<EmptyParams, DetailedUser[]>(getUsers, {});
  const [ activeExperimentsResponse, triggerActiveExperimentsRequest ] =
    useRestApi<GetExperimentsParams, ExperimentPagination>(getExperiments, {});

  const fetchActiveExperiments = useCallback((): void => {
    triggerActiveExperimentsRequest({ states: activeRunStates });
  }, [ triggerActiveExperimentsRequest ]);

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

  const fetchCommands = useFetchCommands(canceler);
  const fetchNotebooks = useFetchNotebooks(canceler);
  const fetchShells = useFetchShells(canceler);
  const fetchTensorboards = useFetchTensorboards(canceler);

  const fetchUsers = useCallback((): void => {
    triggerUsersRequest({ url: '/users' });
  }, [ triggerUsersRequest ]);

  const fetchAll = useCallback((): void => {
    fetchActiveExperiments();
    fetchAgents();
    fetchCommands();
    fetchNotebooks();
    fetchShells();
    fetchTensorboards();
  }, [
    fetchActiveExperiments,
    fetchAgents,
    fetchCommands,
    fetchNotebooks,
    fetchShells,
    fetchTensorboards,
  ]);

  usePolling(fetchAll);
  usePolling(fetchUsers, { delay: 60000 });

  useEffect(() => {
    setUsers({ type: Users.ActionType.Set, value: usersResponse });
  }, [ usersResponse, setUsers ]);
  useEffect(() => {
    setActiveExperiments({
      type: ActiveExperiments.ActionType.Set,
      value: activeExperimentsResponse,
    });
  }, [ activeExperimentsResponse, setActiveExperiments ]);

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  return <React.Fragment />;
};

export default AppContexts;
