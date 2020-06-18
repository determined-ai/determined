import React, { useCallback, useState } from 'react';

import Grid, { GridMode } from 'components/Grid';
import Message from 'components/Message';
import OverviewStats from 'components/OverviewStats';
import Page from 'components/Page';
import Section from 'components/Section';
import Spinner from 'components/Spinner';
import TaskCard from 'components/TaskCard';
import TaskFilter, { ALL_VALUE, filterTasks, TaskFilters } from 'components/TaskFilter';
import ActiveExperiments from 'contexts/ActiveExperiments';
import Auth from 'contexts/Auth';
import ClusterOverview from 'contexts/ClusterOverview';
import { Commands, Notebooks, Shells, Tensorboards } from 'contexts/Commands';
import Experiments from 'contexts/Experiments';
import Users from 'contexts/Users';
import useStorage from 'hooks/useStorage';
import { ShirtSize } from 'themes';
import {
  Command, CommandState, CommandType, RecentTask, ResourceType, RunState, TaskType,
} from 'types';
import { isExperimentTask } from 'utils/task';
import { commandToTask, experimentToTask } from 'utils/types';

import css from './Dashboard.module.scss';

const defaultFilters: TaskFilters = {
  limit: 25,
  states: [ ALL_VALUE ],
  types: {
    [CommandType.Command]: false,
    Experiment: false,
    [CommandType.Notebook]: false,
    [CommandType.Shell]: false,
    [CommandType.Tensorboard]: false,
  },
  username: undefined,
};

const activeStates = [
  RunState.Active,
  RunState.StoppingCanceled,
  RunState.StoppingCompleted,
  RunState.StoppingError,
  CommandState.Assigned,
  CommandState.Pending,
  CommandState.Pulling,
  CommandState.Running,
  CommandState.Starting,
  CommandState.Terminating,
];

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
  const initFilters = storage.getWithDefault('filters',
    { ...defaultFilters, username: (auth.user || {}).username });
  const [ filters, setFilters ] = useState<TaskFilters>(initFilters);

  /* Overview */

  const countActiveCommand = (commands: Command[]): number => {
    return commands.filter(command => command.state !== CommandState.Terminated).length;
  };

  const activeTaskTally = {
    [CommandType.Command]: countActiveCommand(commands.data || []),
    Experiment: (activeExperiments.data || []).length,
    [CommandType.Notebook]: countActiveCommand(notebooks.data || []),
    [CommandType.Shell]: countActiveCommand(shells.data || []),
    [CommandType.Tensorboard]: countActiveCommand(tensorboards.data || []),
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
    (a, b) => Date.parse(a.lastEvent.date) < Date.parse(b.lastEvent.date) ? 1 : -1);

  const filteredTasks = filterTasks(sortedTasks, filters, users.data || []);

  const tasks = filteredTasks.map((props: RecentTask) => {
    return <TaskCard key={props.id} {...props} />;
  });

  const handleFilterChange = useCallback((filters: TaskFilters): void => {
    storage.set('filters', filters);
    setFilters(filters);
  }, [ setFilters, storage ]);

  const taskFilter = <TaskFilter
    authUser={auth.user}
    filters={filters}
    users={users.data || []}
    onChange={handleFilterChange} />;

  /* Overview */

  const activeTally = loadedTasks
    .filter(task => activeStates.includes(task.state))
    .reduce((acc, task) => {
      const attr: TaskType = isExperimentTask(task) ? 'Experiment' : task.type;
      return { ...acc, attr: acc[attr] + 1 };
    }, {
      [CommandType.Command]: 0,
      Experiment: 0,
      [CommandType.Notebook]: 0,
      [CommandType.Shell]: 0,
      [CommandType.Tensorboard]: 0,
    });

  const emptyView = (
    <Message>
      No recent tasks matching the current filters.
    </Message>
  );

  return (
    <Page className={css.base} hideTitle title="Dashboard">
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
            {activeTally.Experiment ? <OverviewStats title="Active Experiments">
              {activeTaskTally.Experiment}
            </OverviewStats> : null}
            {activeTally[CommandType.Notebook] ? <OverviewStats title="Active Notebooks">
              {activeTally[CommandType.Notebook]}
            </OverviewStats> : null}
            {activeTally[CommandType.Tensorboard] ? <OverviewStats title="Active Tensorboards">
              {activeTally[CommandType.Tensorboard]}
            </OverviewStats> : null}
            {activeTally[CommandType.Shell] ? <OverviewStats title="Active Shells">
              {activeTally[CommandType.Shell]}
            </OverviewStats> : null}
            {activeTally[CommandType.Command] ? <OverviewStats title="Active Commands">
              {activeTally[CommandType.Command]}
            </OverviewStats> : null}
          </Grid>
        </div>
      </Section>
      <Section divider={true} options={taskFilter} title="Recent Tasks">
        {showTasksSpinner
          ? <Spinner />
          : tasks.length !== 0
            ? <Grid gap={ShirtSize.medium} mode={GridMode.AutoFill}>{tasks}</Grid>
            : emptyView
        }
      </Section>
    </Page>
  );
};

export default Dashboard;
