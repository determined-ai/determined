import React, { useCallback, useEffect, useState } from 'react';

import ActiveExperiments from 'contexts/ActiveExperiments';
import Agents from 'contexts/Agents';
import ClusterOverview from 'contexts/ClusterOverview';
import { Commands, Notebooks, Shells, Tensorboards } from 'contexts/Commands';
import Users from 'contexts/Users';
import usePolling from 'hooks/usePolling';
import useRestApi from 'hooks/useRestApi';
import {
  getAgents, getCommands, getExperimentSummaries, getNotebooks,
  getShells, getTensorboards, getUsers,
} from 'services/api';
import { EmptyParams, ExperimentsParams } from 'services/types';
import { Command, DetailedUser, ExperimentBase } from 'types';
import { activeRunStates } from 'utils/types';

const AppContexts: React.FC = () => {
  const [ canceler ] = useState(new AbortController());
  const setUsers = Users.useActionContext();
  const setAgents = Agents.useActionContext();
  const setCommands = Commands.useActionContext();
  const setActiveExperiments = ActiveExperiments.useActionContext();
  const setNotebooks = Notebooks.useActionContext();
  const setShells = Shells.useActionContext();
  const setTensorboards = Tensorboards.useActionContext();
  const setOverview = ClusterOverview.useActionContext();
  const [ usersResponse, triggerUsersRequest ] =
    useRestApi<EmptyParams, DetailedUser[]>(getUsers, {});
  const [ commandsResponse, triggerCommandsRequest ] =
    useRestApi<EmptyParams, Command[]>(getCommands, {});
  const [ notebooksResponse, triggerNotebooksRequest ] =
    useRestApi<EmptyParams, Command[]>(getNotebooks, {});
  const [ shellsResponse, triggerShellsRequest ] =
    useRestApi<EmptyParams, Command[]>(getShells, {});
  const [ tensorboardsResponse, triggerTensorboardsRequest ] =
    useRestApi<EmptyParams, Command[]>(getTensorboards, {});
  const [ activeExperimentsResponse, triggerActiveExperimentsRequest ] =
    useRestApi<ExperimentsParams, ExperimentBase[]>(getExperimentSummaries, {});

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

  const fetchAll = useCallback((): void => {
    triggerCommandsRequest({});
    triggerNotebooksRequest({});
    triggerShellsRequest({});
    triggerTensorboardsRequest({});
    triggerActiveExperimentsRequest({ states: activeRunStates });
  }, [
    triggerCommandsRequest,
    triggerNotebooksRequest,
    triggerShellsRequest,
    triggerTensorboardsRequest,
    triggerActiveExperimentsRequest,
  ]);

  const fetchUsers = useCallback((): void => {
    triggerUsersRequest({ url: '/users' });
  }, [ triggerUsersRequest ]);

  usePolling(fetchAgents);
  usePolling(fetchAll);
  usePolling(fetchUsers, { delay: 60000 });

  useEffect(() => {
    setUsers({ type: Users.ActionType.Set, value: usersResponse });
  }, [ usersResponse, setUsers ]);
  useEffect(() => {
    setCommands({ type: Commands.ActionType.Set, value: commandsResponse });
  }, [ commandsResponse, setCommands ]);
  useEffect(() => {
    setActiveExperiments({
      type: ActiveExperiments.ActionType.Set,
      value: activeExperimentsResponse,
    });
  }, [ activeExperimentsResponse, setActiveExperiments ]);
  useEffect(() => {
    setNotebooks({ type: Commands.ActionType.Set, value: notebooksResponse });
  }, [ notebooksResponse, setNotebooks ]);
  useEffect(() => {
    setShells({ type: Commands.ActionType.Set, value: shellsResponse });
  }, [ shellsResponse, setShells ]);
  useEffect(() => {
    setTensorboards({ type: Commands.ActionType.Set, value: tensorboardsResponse });
  }, [ tensorboardsResponse, setTensorboards ]);

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  return <React.Fragment />;
};

export default AppContexts;
