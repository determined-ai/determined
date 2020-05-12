import React, { useCallback, useState } from 'react';

import Grid, { GridMode } from 'components/Grid';
import OverviewStats from 'components/OverviewStats';
import Section from 'components/Section';
import Spinner from 'components/Spinner';
import TaskCard from 'components/TaskCard';
import TaskFilter, { ALL_VALUE, filterTasks, TaskFilters } from 'components/TaskFilter';
import ActiveExperiments from 'contexts/ActiveExperiments';
import Auth from 'contexts/Auth';
import ClusterOverview from 'contexts/ClusterOverview';
import { Commands, Notebooks, Shells, Tensorboards } from 'contexts/Commands';
import Users from 'contexts/Users';
import usePolling from 'hooks/usePolling';
import { useRestApiSimple } from 'hooks/useRestApi';
import useStorage from 'hooks/useStorage';
import { ExperimentsParams, getExperiments } from 'services/api';
import { ShirtSize } from 'themes';
import {
  Command, CommandState, Experiment, RecentTask, ResourceType, RunState, TaskType,
} from 'types';
import { commandToTask, experimentToTask } from 'utils/types';

import css from './Dashboard.module.scss';

const defaultFilters: TaskFilters = {
  limit: 25,
  states: [ ALL_VALUE ],
  types: {
    [TaskType.Command]: false,
    [TaskType.Experiment]: false,
    [TaskType.Notebook]: false,
    [TaskType.Shell]: false,
    [TaskType.Tensorboard]: false,
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
  const commands = Commands.useStateContext();
  const notebooks = Notebooks.useStateContext();
  const shells = Shells.useStateContext();
  const tensorboards = Tensorboards.useStateContext();
  const [ experimentsResponse, requestExperiments ] =
    useRestApiSimple<ExperimentsParams, Experiment[]>(getExperiments, {});
  const storage = useStorage('dashboard/tasks');
  const initFilters = storage.getWithDefault('filters',
    { ...defaultFilters, username: (auth.user || {}).username });
  const [ filters, setFilters ] = useState<TaskFilters>(initFilters);

  const fetchExperiments = useCallback((): void => {
    requestExperiments({});
  }, [ requestExperiments ]);

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
    .reduce((acc, task) => ({ ...acc, [task.type]: acc[task.type] + 1 }), {
      [TaskType.Command]: 0,
      [TaskType.Experiment]: 0,
      [TaskType.Notebook]: 0,
      [TaskType.Shell]: 0,
      [TaskType.Tensorboard]: 0,
    });

  const emptyView = (
    <div className={css.emptyMessage}>
      No recent tasks matching the current filters.
    </div>
  );

  return (
    <div className={css.base}>
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
            {activeTally[TaskType.Experiment] ? <OverviewStats title="Active Experiments">
              {activeTaskTally[TaskType.Experiment]}
            </OverviewStats> : null}
            {activeTally[TaskType.Notebook] ? <OverviewStats title="Active Notebooks">
              {activeTally[TaskType.Notebook]}
            </OverviewStats> : null}
            {activeTally[TaskType.Tensorboard] ? <OverviewStats title="Active Tensorboards">
              {activeTally[TaskType.Tensorboard]}
            </OverviewStats> : null}
            {activeTally[TaskType.Shell] ? <OverviewStats title="Active Shells">
              {activeTally[TaskType.Shell]}
            </OverviewStats> : null}
            {activeTally[TaskType.Command] ? <OverviewStats title="Active Commands">
              {activeTally[TaskType.Command]}
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
    </div>
  );
};

export default Dashboard;
