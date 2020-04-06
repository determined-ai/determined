import React, { useEffect } from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';
import styled from 'styled-components';
import { theme } from 'styled-tools';

import Router from 'components/Router';
import SideBar from 'components/SideBar';
import ActiveExperiments from 'contexts/ActiveExperiments';
import Agents from 'contexts/Agents';
import ClusterOverview from 'contexts/ClusterOverview';
import { Commands, Notebooks, Shells, Tensorboards } from 'contexts/Commands';
import Users from 'contexts/Users';
import usePolling from 'hooks/usePolling';
import useRestApi from 'hooks/useRestApi';
import { ioAgents, ioExperiments, ioGenericCommands, ioUsers } from 'ioTypes';
import { detRoutes } from 'routes';
import {
  jsonToAgents, jsonToCommands, jsonToExperiments, jsonToNotebooks,
  jsonToShells, jsonToTensorboards, jsonToUsers,
} from 'services/decoder';
import { buildExperimentListGqlQuery } from 'services/graphql';
import { Agent, Command, Experiment, RunState, User } from 'types';

// querying active experiments only
const query = buildExperimentListGqlQuery({ states: [
  RunState.Active,
  RunState.StoppingCanceled,
  RunState.StoppingCompleted,
  RunState.StoppingError,
] });

const Determined: React.FC = () => {
  const setUsers = Users.useActionContext();
  const setAgents = Agents.useActionContext();
  const setCommands = Commands.useActionContext();
  const setExperiments = ActiveExperiments.useActionContext();
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
  const [ experimentsResponse, requestExperiments ] =
    useRestApi<Experiment[]>(ioExperiments, { mappers: jsonToExperiments });
  const [ notebooksResponse, requestNotebooks ] =
    useRestApi<Command[]>(ioGenericCommands, { mappers: jsonToNotebooks });
  const [ shellsResponse, requestShells ] =
    useRestApi<Command[]>(ioGenericCommands, { mappers: jsonToShells });
  const [ tensorboardsResponse, requestTensorboards ] =
    useRestApi<Command[]>(ioGenericCommands, { mappers: jsonToTensorboards });

  const fetchAll = (): void => {
    requestAgents({ url: '/agents' });
    requestCommands({ url: '/commands' });
    requestNotebooks({ url: '/notebooks' });
    requestShells({ url: '/shells' });
    requestTensorboards({ url: '/tensorboard' });
    requestExperiments({ body: query, method: 'POST', url: '/graphql' });
  };

  const fetchUsers = (): void => requestUsers({ url: '/users' });

  usePolling(fetchAll);
  usePolling(fetchUsers, { delay: 60000 });

  useEffect(() => {
    setUsers({ type: Users.ActionType.Set, value: usersResponse });
  }, [ usersResponse, setUsers ]);
  useEffect(() => {
    setAgents({ type: Agents.ActionType.Set, value: agentsResponse });
    if (!agentsResponse.data) return;
    setOverview({ type: ClusterOverview.ActionType.SetAgents, value: agentsResponse.data });
  }, [ agentsResponse.data, setOverview ]);
  useEffect(() => {
    setCommands({ type: Commands.ActionType.Set, value: commandsResponse });
  }, [ commandsResponse, setCommands ]);
  useEffect(() => {
    setExperiments({ type: ActiveExperiments.ActionType.Set, value: experimentsResponse });
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

  return (
    <Base>
      <SideBar />
      <Switch>
        <Route exact path="/det">
          <Redirect to="/det/dashboard" />
        </Route>
        <Router routes={detRoutes} />
      </Switch>
    </Base>
  );
};

const Base = styled.div`
  --navHeight: ${theme('sizes.navbar.height')};
  display: flex;
  height: calc(100% - var(--navHeight));
  &:first-child { flex-grow: 0; }
  &:last-child {
    background-color: white;
    flex-grow: 1;
  }
`;

export default Determined;
