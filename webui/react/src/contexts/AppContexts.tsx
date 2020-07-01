import React, { useCallback, useEffect } from 'react';

import ActiveExperiments from 'contexts/ActiveExperiments';
import Agents from 'contexts/Agents';
import ClusterOverview from 'contexts/ClusterOverview';
import { Commands, Notebooks, Shells, Tensorboards } from 'contexts/Commands';
import Experiments from 'contexts/Experiments';
import Users from 'contexts/Users';
import usePolling from 'hooks/usePolling';
import useRestApi, { useRestApiSimple } from 'hooks/useRestApi';
import { ioAgents, ioGenericCommands, ioUsers } from 'ioTypes';
import { getExperimentSummaries } from 'services/api';
import {
  jsonToAgents, jsonToCommands, jsonToNotebooks,
  jsonToShells, jsonToTensorboards, jsonToUsers,
} from 'services/decoder';
import { ExperimentsParams } from 'services/types';
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
    useRestApi<User[]>(ioUsers, { mappers: jsonToUsers });
  const [ agentsResponse, requestAgents ] =
    useRestApi<Agent[]>(ioAgents, { mappers: jsonToAgents });
  const [ commandsResponse, requestCommands ] =
    useRestApi<Command[]>(ioGenericCommands, { mappers: jsonToCommands });
  const [ activeExperimentsResponse, requestActiveExperiments ] =
    useRestApiSimple<ExperimentsParams, Experiment[]>(getExperimentSummaries, {});
  const [ notebooksResponse, requestNotebooks ] =
    useRestApi<Command[]>(ioGenericCommands, { mappers: jsonToNotebooks });
  const [ shellsResponse, requestShells ] =
    useRestApi<Command[]>(ioGenericCommands, { mappers: jsonToShells });
  const [ tensorboardsResponse, requestTensorboards ] =
    useRestApi<Command[]>(ioGenericCommands, { mappers: jsonToTensorboards });
  const [ experimentsResponse, requestExperiments ] =
    useRestApiSimple<ExperimentsParams, Experiment[]>(getExperimentSummaries, {});

  const fetchAll = useCallback((): void => {
    requestAgents({ url: '/agents' });
    requestCommands({ url: '/commands' });
    requestNotebooks({ url: '/notebooks' });
    requestShells({ url: '/shells' });
    requestTensorboards({ url: '/tensorboard' });
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
