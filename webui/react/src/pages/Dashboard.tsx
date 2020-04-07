import React, { useCallback, useState } from 'react';
import styled from 'styled-components';
import { theme } from 'styled-tools';

import Grid, { GridMode } from 'components/Grid';
import OverviewStats from 'components/OverviewStats';
import Section from 'components/Section';
import Spinner from 'components/Spinner';
import TaskCard from 'components/TaskCard';
import TaskFilter, { filterTasks, getTaskCounts, TaskFilters } from 'components/TaskFilter';
import ActiveExperiments from 'contexts/ActiveExperiments';
import Auth from 'contexts/Auth';
import ClusterOverview from 'contexts/ClusterOverview';
import { Commands, Notebooks, Shells, Tensorboards } from 'contexts/Commands';
import Users from 'contexts/Users';
import usePolling from 'hooks/usePolling';
import useRestApi from 'hooks/useRestApi';
import useStorage from 'hooks/useStorage';
import { ioExperiments } from 'ioTypes';
import { jsonToExperiments } from 'services/decoder';
import { buildExperimentListGqlQuery } from 'services/graphql';
import {
  Command, CommandState, Experiment, RecentTask, ResourceType, TaskType,
} from 'types';
import { commandToTask, experimentToTask } from 'utils/types';

const defaultFilters: TaskFilters = {
  limit: 25,
  states: [ 'ALL' ],
  types: {
    [TaskType.Command]: true,
    [TaskType.Experiment]: true,
    [TaskType.Notebook]: true,
    [TaskType.Tensorboard]: true,
    [TaskType.Shell]: true,
  },
  userId: undefined,
};

const Dashboard: React.FC = () => {
  const auth = Auth.useStateContext();
  const users = Users.useStateContext();
  const overview = ClusterOverview.useStateContext();
  const activeExperiments = ActiveExperiments.useStateContext();
  const commands = Commands.useStateContext();
  const notebooks = Notebooks.useStateContext();
  const shells = Shells.useStateContext();
  const tensorboards = Tensorboards.useStateContext();
  const [ experimentsResponse, requestExperiments ] =
    useRestApi<Experiment[]>(ioExperiments, { mappers: jsonToExperiments });
  const storage = useStorage('dashboard/tasks');
  const [ filters, setFilters ] = useState<TaskFilters>(
    storage.getWithDefault('filters', { ...defaultFilters, userId: (auth.user || {}).id }),
  );

  const fetchExperiments = (): void => {
    requestExperiments({
      body: buildExperimentListGqlQuery({ limit: 100 }),
      method: 'POST',
      url: '/graphql',
    });
  };

  usePolling(fetchExperiments);

  /* Overview */

  const countActiveCommand = (commands: Command[]): number => {
    return commands.filter(command => command.state !== CommandState.Terminated).length;
  };

  const activeTaskTally = {
    [TaskType.Command]: countActiveCommand(commands.data || []),
    [TaskType.Experiment]: (activeExperiments.data || []).length,
    [TaskType.Notebook]: countActiveCommand(notebooks.data || []),
    [TaskType.Shell]: countActiveCommand(shells.data || []),
    [TaskType.Tensorboard]: countActiveCommand(tensorboards.data || []),
  };

  /* Recent Tasks */

  const showTasksSpinner = (
    !experimentsResponse.hasLoaded ||
    !commands.hasLoaded ||
    !notebooks.hasLoaded ||
    !shells.hasLoaded ||
    !tensorboards.hasLoaded
  );

  const genericCommands = [
    ...(commands.data || []),
    ...(notebooks.data || []),
    ...(shells.data || []),
    ...(tensorboards.data || []),
  ];

  const loadedTasks = [
    ...(experimentsResponse.data || []).map(experimentToTask),
    ...genericCommands.map(commandToTask),
  ];

  const sortedTasks = loadedTasks.sort(
    (a, b) => Date.parse(a.lastEvent.date) < Date.parse(b.lastEvent.date) ? 1 : -1);

  const filteredTasks = filterTasks(sortedTasks, filters);

  const tasks = filteredTasks.map((props: RecentTask) => {
    return <TaskCard key={props.id} {...props} />;
  });

  const taskCounts = getTaskCounts(filteredTasks);

  const handleFilterChange = useCallback((filters: TaskFilters): void => {
    storage.set('filters', filters);
    setFilters(filters);
  }, [ setFilters, storage ]);

  const taskFilter = <TaskFilter
    counts={taskCounts}
    filters={filters}
    users={users.data || []}
    onChange={handleFilterChange} />;

  return (
    <Base>
      <Section title="Overview">
        <StyledGrid minItemWidth={12}>
          <OverviewStats title="Cluster Allocation">
            {overview.allocation}<small>%</small>
          </OverviewStats>
          <OverviewStats title="Available GPUs">
            {overview[ResourceType.GPU].available}
            <small>/{overview[ResourceType.GPU].total}</small>
          </OverviewStats>
          <OverviewStats title="Available CPUs">
            {overview[ResourceType.CPU].available}
            <small>/{overview[ResourceType.CPU].total}</small>
          </OverviewStats>
          <OverviewStats title="Active Experiments">
            {activeTaskTally[TaskType.Experiment]}
          </OverviewStats>
          <OverviewStats title="Active Notebooks">
            {activeTaskTally[TaskType.Notebook]}
          </OverviewStats>
          <OverviewStats title="Active Tensorboards">
            {activeTaskTally[TaskType.Tensorboard]}
          </OverviewStats>
        </StyledGrid>
      </Section>
      <Section divider={true} options={taskFilter} title="Recent Tasks">
        {showTasksSpinner ? <Spinner />
          : tasks.length !== 0 ? <Grid gap={2} mode={GridMode.AutoFill}>{tasks}</Grid>
            : <EmptyMessage>No recent tasks matching the current filters.</EmptyMessage>}
      </Section>
    </Base>
  );
};

const StyledGrid = styled(Grid)`
  background-color: ${theme('colors.monochrome.14')};
  padding: ${theme('sizes.layout.medium')};
`;

const Base = styled.div`
  background-color: transparent;
  overflow: auto;
  padding: ${theme('sizes.layout.giant')};
  width: 100%;
`;

const EmptyMessage = styled.div`
  align-items: center;
  background-color: ${theme('colors.monochrome.16')};
  color: ${theme('colors.monochrome.9')};
  display: flex;
  font-size: ${theme('sizes.font.big')};
  font-style: italic;
  justify-content: center;
  padding: ${theme('sizes.layout.giant')};
`;

export default Dashboard;
