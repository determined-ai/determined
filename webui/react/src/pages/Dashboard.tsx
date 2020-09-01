import React, { useCallback, useState } from 'react';

import Grid, { GridMode } from 'components/Grid';
import Message, { MessageType } from 'components/Message';
import OverviewStats from 'components/OverviewStats';
import Page from 'components/Page';
import Section from 'components/Section';
import Spinner from 'components/Spinner';
import TaskCard from 'components/TaskCard';
import TaskFilter from 'components/TaskFilter';
import ActiveExperiments from 'contexts/ActiveExperiments';
import Auth from 'contexts/Auth';
import ClusterOverview from 'contexts/ClusterOverview';
import { Commands, Notebooks, Shells, Tensorboards } from 'contexts/Commands';
import Experiments from 'contexts/Experiments';
import Users from 'contexts/Users';
import useStorage from 'hooks/useStorage';
import { ShirtSize } from 'themes';
import {
  ALL_VALUE, Command, CommandType, RecentTask, ResourceType,
  TaskFilters, TaskType,
} from 'types';
import { getPath } from 'utils/data';
import { filterTasks } from 'utils/task';
import { activeCommandStates, commandToTask, experimentToTask } from 'utils/types';

import css from './Dashboard.module.scss';

const defaultFilters: TaskFilters = {
  limit: 25,
  states: [ ALL_VALUE ],
  types: {
    [CommandType.Command]: false,
    [CommandType.Notebook]: false,
    [CommandType.Shell]: false,
    [CommandType.Tensorboard]: false,
    Experiment: false,
  },
  username: undefined,
};

const Dashboard: React.FC = () => {
  const auth = Auth.useStateContext();
  const users = Users.useStateContext();
  const overview = ClusterOverview.useStateContext();
  const activeExperiments = ActiveExperiments.useStateContext();
  const experiments = Experiments.useStateContext();
  const commands = Commands.useStateContext();
  const notebooks = Notebooks.useStateContext();
  const shells = Shells.useStateContext();
  const tensorboards = Tensorboards.useStateContext();

  const storage = useStorage('dashboard/tasks');
  const initFilters = storage.getWithDefault('filters', {
    ...defaultFilters,
    username: getPath<string>(auth, 'user.username'),
  });
  const [ filters, setFilters ] = useState<TaskFilters>(initFilters);

  /* Overview */

  const countActiveCommand = (commands: Command[]): number => {
    return commands.filter(command => activeCommandStates.includes(command.state)).length;
  };

  const activeTaskTally = {
    [CommandType.Command]: countActiveCommand(commands.data || []),
    [CommandType.Notebook]: countActiveCommand(notebooks.data || []),
    [CommandType.Shell]: countActiveCommand(shells.data || []),
    [CommandType.Tensorboard]: countActiveCommand(tensorboards.data || []),
    Experiment: (activeExperiments.data || []).length,
  };

  /* Recent Tasks */

  const showTasksSpinner = (
    !experiments.hasLoaded ||
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
    ...(experiments.data || []).map(experimentToTask),
    ...genericCommands.map(commandToTask),
  ];

  const sortedTasks = loadedTasks.sort(
    (a, b) => Date.parse(a.lastEvent.date) < Date.parse(b.lastEvent.date) ? 1 : -1,
  );

  const filteredTasks = filterTasks<TaskType, RecentTask>(sortedTasks, filters, users.data || [])
    .slice(0, filters.limit);

  const tasks = filteredTasks.map((props: RecentTask) => {
    return <TaskCard key={props.id} {...props} />;
  });

  const handleFilterChange = useCallback((filters: TaskFilters): void => {
    storage.set('filters', filters);
    setFilters(filters);
  }, [ setFilters, storage ]);

  const taskFilter = <TaskFilter filters={filters} onChange={handleFilterChange} />;

  return (
    <Page className={css.base} id="dashboard">
      <Section title="Overview">
        <div className={css.overview}>
          <Grid gap={ShirtSize.medium} minItemWidth={12} mode={GridMode.AutoFill}>
            <OverviewStats title="Cluster Allocation">
              {overview.allocation}<small>%</small>
            </OverviewStats>
            {overview[ResourceType.GPU].total ? <OverviewStats title="Available GPUs">
              {overview[ResourceType.GPU].available}
              <small>/{overview[ResourceType.GPU].total}</small>
            </OverviewStats> : null}
            {overview[ResourceType.CPU].total ? <OverviewStats title="Available CPUs">
              {overview[ResourceType.CPU].available}
              <small>/{overview[ResourceType.CPU].total}</small>
            </OverviewStats> : null}
            {activeTaskTally.Experiment ? <OverviewStats title="Active Experiments">
              {activeTaskTally.Experiment}
            </OverviewStats> : null}
            {activeTaskTally[CommandType.Notebook] ? <OverviewStats title="Active Notebooks">
              {activeTaskTally[CommandType.Notebook]}
            </OverviewStats> : null}
            {activeTaskTally[CommandType.Tensorboard] ? <OverviewStats title="Active Tensorboards">
              {activeTaskTally[CommandType.Tensorboard]}
            </OverviewStats> : null}
            {activeTaskTally[CommandType.Shell] ? <OverviewStats title="Active Shells">
              {activeTaskTally[CommandType.Shell]}
            </OverviewStats> : null}
            {activeTaskTally[CommandType.Command] ? <OverviewStats title="Active Commands">
              {activeTaskTally[CommandType.Command]}
            </OverviewStats> : null}
          </Grid>
        </div>
      </Section>
      <Section divider={true} options={taskFilter} title="Recent Tasks">
        {showTasksSpinner
          ? <Spinner />
          : tasks.length !== 0
            ? <Grid gap={ShirtSize.medium} mode={GridMode.AutoFill}>{tasks}</Grid>
            : <Message
              title="No recent tasks matching the current filters"
              type={MessageType.Empty} />
        }
      </Section>
    </Page>
  );
};

export default Dashboard;
