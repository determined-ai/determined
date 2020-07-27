import React, { useCallback, useEffect } from 'react';

import ActiveExperiments from 'contexts/ActiveExperiments';
import Agents from 'contexts/Agents';
import ClusterOverview from 'contexts/ClusterOverview';
import { Commands, Notebooks, Shells, Tensorboards } from 'contexts/Commands';
import Experiments from 'contexts/Experiments';
import Users from 'contexts/Users';
import usePolling from 'hooks/usePolling';
import useRestApi from 'hooks/useRestApi';
import {
  getAgents, getCommands, getExperimentSummaries, getNotebooks, getShells,
  getTensorboards, getUsers,
} from 'services/api';
import { EmptyParams, ExperimentsParams } from 'services/types';
import { Agent, Command, Experiment, User } from 'types';
import { activeRunStates } from 'utils/types';

const AppContexts: React.FC = () => {
  const setUsers = Users.useActionContext();
  const setAgents = Agents.useActionContext();
  const setCommands = Commands.useActionContext();
  const setActiveExperiments = ActiveExperiments.useActionContext();
  const setExperiments = Experiments.useActionContext();
  const setNotebooks = Notebooks.useActionContext();
  const setShells = Shells.useActionContext();
  const setTensorboards = Tensorboards.useActionContext();
  const setOverview = ClusterOverview.useActionContext();
  const [ usersResponse, requestUsers ] =
    useRestApi<EmptyParams, User[]>(getUsers, {});
  const [ agentsResponse, requestAgents ] =
    useRestApi<EmptyParams, Agent[]>(getAgents, {});
  const [ commandsResponse, requestCommands ] =
    useRestApi<EmptyParams, Command[]>(getCommands, {});
  const [ notebooksResponse, requestNotebooks ] =
    useRestApi<EmptyParams, Command[]>(getNotebooks, {});
  const [ shellsResponse, requestShells ] =
    useRestApi<EmptyParams, Command[]>(getShells, {});
  const [ tensorboardsResponse, requestTensorboards ] =
    useRestApi<EmptyParams, Command[]>(getTensorboards, {});
  const [ experimentsResponse, requestExperiments ] =
    useRestApi<ExperimentsParams, Experiment[]>(getExperimentSummaries, {});
  const [ activeExperimentsResponse, requestActiveExperiments ] =
    useRestApi<ExperimentsParams, Experiment[]>(getExperimentSummaries, {});

  const fetchAll = useCallback((): void => {
    requestAgents({});
    requestCommands({});
    requestNotebooks({});
    requestShells({});
    requestTensorboards({});
    requestActiveExperiments({ states: activeRunStates });
    requestExperiments({});
  }, [
    requestAgents,
    requestCommands,
    requestNotebooks,
    requestShells,
    requestTensorboards,
    requestExperiments,
    requestActiveExperiments,
  ]);

  const fetchUsers = useCallback((): void => requestUsers({ url: '/users' }), [ requestUsers ]);

  usePolling(fetchAll);
  usePolling(fetchUsers, { delay: 60000 });

  useEffect(() => {
    setUsers({ type: Users.ActionType.Set, value: usersResponse });
  }, [ usersResponse, setUsers ]);
  useEffect(() => {
    setAgents({ type: Agents.ActionType.Set, value: agentsResponse });
    if (!agentsResponse.data) return;
    setOverview({ type: ClusterOverview.ActionType.SetAgents, value: agentsResponse.data });
  }, [ agentsResponse.data, setOverview, agentsResponse, setAgents ]);
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
    setExperiments({ type: Experiments.ActionType.Set, value: experimentsResponse });
  }, [ experimentsResponse, setExperiments ]);
  useEffect(() => {
    setNotebooks({ type: Commands.ActionType.Set, value: notebooksResponse });
  }, [ notebooksResponse, setNotebooks ]);
  useEffect(() => {
    setShells({ type: Commands.ActionType.Set, value: shellsResponse });
  }, [ shellsResponse, setShells ]);
  useEffect(() => {
    setTensorboards({ type: Commands.ActionType.Set, value: tensorboardsResponse });
  }, [ tensorboardsResponse, setTensorboards ]);

  return <React.Fragment />;
};

export default AppContexts;
