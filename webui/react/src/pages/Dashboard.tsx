import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Grid, { GridMode } from 'components/Grid';
import Message, { MessageType } from 'components/Message';
import OverviewStats from 'components/OverviewStats';
import Page from 'components/Page';
import Section from 'components/Section';
import TaskCard from 'components/TaskCard';
import TaskFilter from 'components/TaskFilter';
import { useStore } from 'contexts/Store';
import { ErrorType } from 'ErrorHandler';
import handleError from 'ErrorHandler';
import { useFetchUsers } from 'hooks/useFetch';
import usePolling from 'hooks/usePolling';
import useStorage from 'hooks/useStorage';
import {
  getCommands, getExperiments, getNotebooks, getShells, getTensorboards,
} from 'services/api';
import { Determinedexperimentv1State } from 'services/api-ts-sdk';
import { encodeExperimentState } from 'services/decoder';
import { validateDetApiEnumList } from 'services/utils';
import { ShirtSize } from 'themes';
import {
  ALL_VALUE, CommandTask, CommandType, ExperimentItem, RecentTask,
  ResourceType, RunState, TaskFilters, TaskType,
} from 'types';
import { isEqual } from 'utils/data';
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

const countActiveCommand = (commands: CommandTask[]): number => {
  return commands.filter(command => activeCommandStates.includes(command.state)).length;
};

const Dashboard: React.FC = () => {
  const { auth, cluster: overview, users } = useStore();
  const storage = useStorage(STORAGE_PATH);
  const initFilters = storage.getWithDefault(
    STORAGE_FILTERS_KEY,
    (!auth.user || auth.user?.isAdmin) ? defaultFilters : {
      ...defaultFilters,
      username: auth.user?.username,
    },
  );
  const [ filters, setFilters ] = useState<TaskFilters>(initFilters);
  const [ canceler ] = useState(new AbortController());
  const [ experiments, setExperiments ] = useState<ExperimentItem[]>();
  const [ tasks, setTasks ] = useState<CommandTask[]>();
  const [ activeTaskTally, setActiveTaskTally ] = useState({
    [CommandType.Command]: 0,
    [CommandType.Notebook]: 0,
    [CommandType.Shell]: 0,
    [CommandType.Tensorboard]: 0,
  });
  const [ activeExperimentCount, setActiveExperimentCount ] = useState<number>();

  const fetchUsers = useFetchUsers(canceler);

  const fetchExperiments = useCallback(async (): Promise<void> => {
    try {
      const states = (filters.states || []).map(state => encodeExperimentState(state as RunState));
      const users = filters.username ? [ filters.username ] : undefined;
      const response = await getExperiments(
        {
          archived: false,
          limit: 50,
          orderBy: 'ORDER_BY_DESC',
          sortBy: 'SORT_BY_START_TIME',
          states: validateDetApiEnumList(Determinedexperimentv1State, states),
          users,
        },
        { signal: canceler.signal },
      );
      setExperiments(prev => {
        if (isEqual(prev, response.experiments)) return prev;
        return response.experiments;
      });
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

  const fetchTasks = useCallback(async () => {
    try {
      const [ commands, notebooks, shells, tensorboards ] = await Promise.all([
        getCommands({ signal: canceler.signal }),
        getNotebooks({ signal: canceler.signal }),
        getShells({ signal: canceler.signal }),
        getTensorboards({ signal: canceler.signal }),
      ]);
      setActiveTaskTally(prev => {
        const newTally = {
          [CommandType.Command]: countActiveCommand(commands),
          [CommandType.Notebook]: countActiveCommand(notebooks),
          [CommandType.Shell]: countActiveCommand(shells),
          [CommandType.Tensorboard]: countActiveCommand(tensorboards),
        };
        if (!isEqual(prev, newTally)) return newTally;
        return prev;
      });
      setTasks(prev => {
        const newTasks = [ ...commands, ...notebooks, ...shells, ...tensorboards ];
        if (isEqual(prev, newTasks)) return prev;
        return newTasks;
      });
    } catch (e) {
      handleError({ message: 'Unable to fetch tasks.', silent: true, type: ErrorType.Api });
    }
  }, [ canceler ]);

  const fetchAll = useCallback(() => {
    fetchUsers();
    fetchExperiments();
    fetchActiveExperiments();
    fetchTasks();
  }, [ fetchUsers, fetchExperiments, fetchActiveExperiments, fetchTasks ]);

  usePolling(fetchAll);

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  /* Recent Tasks */

  const loadedTasks = useMemo(() => ([
    ...(experiments || []).map(experimentToTask),
    ...(tasks || []).map(commandToTask),
  ]), [ experiments, tasks ] );

  const sortedTasks = loadedTasks.sort(
    (a, b) => Date.parse(a.lastEvent.date) < Date.parse(b.lastEvent.date) ? 1 : -1,
  );

  const filteredTasks = filterTasks<TaskType, RecentTask>(sortedTasks, filters, users || [])
    .slice(0, filters.limit);

  const handleFilterChange = useCallback((filters: TaskFilters): void => {
    storage.set(STORAGE_FILTERS_KEY, filters);
    setFilters(filters);
    setExperiments(undefined);
  }, [ setExperiments, setFilters, storage ]);

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
          {activeExperimentCount ? <OverviewStats title="Active Experiments">
            {activeExperimentCount}
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
      <Section
        divider
        loading={!experiments || !tasks}
        options={<TaskFilter filters={filters} onChange={handleFilterChange} />}
        title="Recent Tasks">
        {filteredTasks.length !== 0
          ? (
            <Grid gap={ShirtSize.medium} mode={GridMode.AutoFill}>
              {filteredTasks.map((props: RecentTask) => <TaskCard key={props.id} {...props} />)}
            </Grid>
          ) : <Message
            title="No recent tasks matching the current filters"
            type={MessageType.Empty} />
        }
      </Section>
    </Page>
  );
};

export default Dashboard;
