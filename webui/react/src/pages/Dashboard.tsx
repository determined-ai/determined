import React, { useCallback, useEffect, useState } from 'react';

import Grid, { GridMode } from 'components/Grid';
import Message, { MessageType } from 'components/Message';
import OverviewStats from 'components/OverviewStats';
import Page from 'components/Page';
import Section from 'components/Section';
import Spinner from 'components/Spinner';
import TaskCard from 'components/TaskCard';
import TaskFilter from 'components/TaskFilter';
import Auth from 'contexts/Auth';
import ClusterOverview from 'contexts/ClusterOverview';
import {
  Commands, Notebooks, Shells, Tensorboards,
  useFetchCommands, useFetchNotebooks, useFetchShells, useFetchTensorboards,
} from 'contexts/Commands';
import Users, { useFetchUsers } from 'contexts/Users';
import { ErrorType } from 'ErrorHandler';
import handleError from 'ErrorHandler';
import usePolling from 'hooks/usePolling';
import useStorage from 'hooks/useStorage';
import { getExperiments } from 'services/api';
import { encodeExperimentState } from 'services/decoder';
import { ShirtSize } from 'themes';
import {
  ALL_VALUE, CommandTask, CommandType, ExperimentItem, RecentTask,
  ResourceType, RunState, TaskFilters, TaskType,
} from 'types';
import { filterTasks } from 'utils/task';
import { activeCommandStates, activeRunStates, commandToTask, experimentToTask } from 'utils/types';

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

const STORAGE_PATH = 'dashboard';
const STORAGE_FILTERS_KEY = 'filters';

const Dashboard: React.FC = () => {
  const auth = Auth.useStateContext();
  const users = Users.useStateContext();
  const storage = useStorage(STORAGE_PATH);
  const initFilters = storage.getWithDefault(
    STORAGE_FILTERS_KEY,
    (!auth.user || auth.user?.isAdmin) ? defaultFilters : {
      ...defaultFilters,
      username: auth.user?.username,
    },
  );
  const [ filters, setFilters ] = useState<TaskFilters>(initFilters);
  const overview = ClusterOverview.useStateContext();
  const commands = Commands.useStateContext();
  const notebooks = Notebooks.useStateContext();
  const shells = Shells.useStateContext();
  const tensorboards = Tensorboards.useStateContext();
  const [ canceler ] = useState(new AbortController());
  const [ experiments, setExperiments ] = useState<ExperimentItem[]>();
  const [ activeExperimentCount, setActiveExperimentCount ] = useState<number>();

  const fetchUsers = useFetchUsers(canceler);
  const fetchCommands = useFetchCommands(canceler);
  const fetchNotebooks = useFetchNotebooks(canceler);
  const fetchShells = useFetchShells(canceler);
  const fetchTensorboards = useFetchTensorboards(canceler);

  const fetchExperiments = useCallback(async (): Promise<void> => {
    try {
      const states = filters.states.includes(ALL_VALUE) ? undefined : filters.states.map(state => {
        /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
        return encodeExperimentState(state as RunState) as any;
      });
      const users = filters.username ? [ filters.username ] : undefined;
      const response = await getExperiments(
        {
          archived: false,
          limit: 50,
          orderBy: 'ORDER_BY_DESC',
          sortBy: 'SORT_BY_START_TIME',
          states,
          users,
        },
        { signal: canceler.signal },
      );
      setExperiments(response.experiments);
    } catch (e) {
      handleError({ message: 'Unable to fetch experiments.', silent: true, type: ErrorType.Api });
    }
  }, [ canceler, filters, setExperiments ]);

  const fetchActiveExperiments = useCallback(async () => {
    try {
      const response = await getExperiments(
        { limit: -2, states: activeRunStates },
        { signal: canceler.signal },
      );
      setActiveExperimentCount(response.pagination.total);
    } catch (e) {
      handleError({
        message: 'Unable to fetch active experiments.',
        silent: true,
        type: ErrorType.Api,
      });
    }
  }, [ canceler, setActiveExperimentCount ]);

  const fetchAll = useCallback(() => {
    fetchUsers();
    fetchExperiments();
    fetchActiveExperiments();
    fetchCommands();
    fetchNotebooks();
    fetchShells();
    fetchTensorboards();
  }, [
    fetchUsers,
    fetchExperiments,
    fetchActiveExperiments,
    fetchCommands,
    fetchNotebooks,
    fetchShells,
    fetchTensorboards,
  ]);

  usePolling(fetchAll);

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  /* Overview */

  const countActiveCommand = (commands: CommandTask[]): number => {
    return commands.filter(command => activeCommandStates.includes(command.state)).length;
  };

  const activeTaskTally = {
    [CommandType.Command]: countActiveCommand(commands || []),
    [CommandType.Notebook]: countActiveCommand(notebooks || []),
    [CommandType.Shell]: countActiveCommand(shells || []),
    [CommandType.Tensorboard]: countActiveCommand(tensorboards || []),
    Experiment: activeExperimentCount,
  };

  /* Recent Tasks */

  const showTasksSpinner = (
    !experiments ||
    !commands ||
    !notebooks ||
    !shells ||
    !tensorboards
  );

  const genericCommands = [
    ...(commands || []),
    ...(notebooks || []),
    ...(shells || []),
    ...(tensorboards || []),
  ];

  const loadedTasks = [
    ...(experiments || []).map(experimentToTask),
    ...genericCommands.map(commandToTask),
  ];

  const sortedTasks = loadedTasks.sort(
    (a, b) => Date.parse(a.lastEvent.date) < Date.parse(b.lastEvent.date) ? 1 : -1,
  );

  const filteredTasks = filterTasks<TaskType, RecentTask>(sortedTasks, filters, users || [])
    .slice(0, filters.limit);

  const tasks = filteredTasks.map((props: RecentTask) => {
    return <TaskCard key={props.id} {...props} />;
  });

  const handleFilterChange = useCallback((filters: TaskFilters): void => {
    storage.set(STORAGE_FILTERS_KEY, filters);
    setFilters(filters);
    setExperiments(undefined);
  }, [ setExperiments, setFilters, storage ]);

  const taskFilter = <TaskFilter filters={filters} onChange={handleFilterChange} />;

  return (
    <Page docTitle="Overview" id="dashboard">
      <Section title="Overview">
        <Grid gap={ShirtSize.medium} minItemWidth={12} mode={GridMode.AutoFill}>
          <OverviewStats title="Cluster Allocation">
            {overview[ResourceType.ALL].allocation}<small>%</small>
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
      </Section>
      <Section divider={true} loading={showTasksSpinner} options={taskFilter} title="Recent Tasks">
        {tasks.length !== 0
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
