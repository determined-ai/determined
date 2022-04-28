import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Grid, { GridMode } from 'components/Grid';
import OverviewStats from 'components/OverviewStats';
import Page from 'components/Page';
import Section from 'components/Section';
import TaskCard from 'components/TaskCard';
import TaskFilter from 'components/TaskFilter';
import { activeCommandStates, activeRunStates } from 'constants/states';
import { useStore } from 'contexts/Store';
import { useFetchUsers } from 'hooks/useFetch';
import usePolling from 'hooks/usePolling';
import useStorage from 'hooks/useStorage';
import {
  getCommands, getExperiments, getJupyterLabs, getShells, getTensorBoards,
} from 'services/api';
import { Determinedexperimentv1State } from 'services/api-ts-sdk';
import { encodeExperimentState } from 'services/decoder';
import { validateDetApiEnumList } from 'services/utils';
import Message, { MessageType } from 'shared/components/message';
import { ShirtSize } from 'themes';
import {
  ALL_VALUE, CommandTask, CommandType, ExperimentItem, RecentTask,
  ResourceType, RunState, TaskFilters, TaskType,
} from 'types';
import { isEqual, validateEnumList } from 'utils/data';
import handleError, { ErrorType } from 'utils/error';
import { filterTasks, taskFromCommandTask, taskFromExperiment } from 'utils/task';

const defaultFilters: TaskFilters = {
  limit: 25,
  states: [ ALL_VALUE ],
  types: undefined,
  users: undefined,
};

const STORAGE_PATH = 'dashboard';
const STORAGE_FILTERS_KEY = 'filters';

const countActiveCommand = (commands: CommandTask[]): number => {
  return commands.filter(command => activeCommandStates.includes(command.state)).length;
};

const Dashboard: React.FC = () => {
  const { cluster: overview, users, auth: { user } } = useStore();
  const storage = useStorage(STORAGE_PATH);
  const initFilters = storage.getWithDefault(STORAGE_FILTERS_KEY, { ...defaultFilters });
  const [ filters, setFilters ] = useState<TaskFilters>(() => {
    return { ...initFilters, types: validateEnumList(TaskType, initFilters.types) };
  });
  const [ canceler ] = useState(new AbortController());
  const [ experiments, setExperiments ] = useState<ExperimentItem[]>();
  const [ tasks, setTasks ] = useState<CommandTask[]>();
  const [ activeTaskTally, setActiveTaskTally ] = useState({
    [CommandType.Command]: 0,
    [CommandType.JupyterLab]: 0,
    [CommandType.Shell]: 0,
    [CommandType.TensorBoard]: 0,
  });
  const [ activeExperimentCount, setActiveExperimentCount ] = useState<number>();

  const fetchUsers = useFetchUsers(canceler);

  const fetchExperiments = useCallback(async (): Promise<void> => {
    try {
      const states = (filters.states || []).map(state => encodeExperimentState(state as RunState));
      const response = await getExperiments(
        {
          archived: false,
          limit: 50,
          orderBy: 'ORDER_BY_DESC',
          sortBy: 'SORT_BY_START_TIME',
          states: validateDetApiEnumList(Determinedexperimentv1State, states),
          users: filters.users,
        },
        { signal: canceler.signal },
      );
      setExperiments(prev => {
        if (isEqual(prev, response.experiments)) return prev;
        return response.experiments;
      });
    } catch (e) {
      handleError(e, {
        publicSubject: 'Unable to fetch experiments.',
        silent: true,
        type: ErrorType.Api,
      });
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
      handleError(e, {
        publicSubject: 'Unable to fetch active experiments.',
        silent: true,
        type: ErrorType.Api,
      });
    }
  }, [ canceler, setActiveExperimentCount ]);

  const fetchTasks = useCallback(async () => {
    try {
      const [ commands, jupyterLabs, shells, tensorboards ] = await Promise.all([
        getCommands({ signal: canceler.signal }),
        getJupyterLabs({ signal: canceler.signal }),
        getShells({ signal: canceler.signal }),
        getTensorBoards({ signal: canceler.signal }),
      ]);
      setActiveTaskTally(prev => {
        const newTally = {
          [CommandType.Command]: countActiveCommand(commands),
          [CommandType.JupyterLab]: countActiveCommand(jupyterLabs),
          [CommandType.Shell]: countActiveCommand(shells),
          [CommandType.TensorBoard]: countActiveCommand(tensorboards),
        };
        if (!isEqual(prev, newTally)) return newTally;
        return prev;
      });
      setTasks(prev => {
        const newTasks = [ ...commands, ...jupyterLabs, ...shells, ...tensorboards ];
        if (isEqual(prev, newTasks)) return prev;
        return newTasks;
      });
    } catch (e) {
      handleError(e, {
        publicSubject: 'Unable to fetch tasks.',
        silent: true,
        type: ErrorType.Api,
      });
    }
  }, [ canceler ]);

  const fetchAll = useCallback(async () => {
    await Promise.allSettled([
      fetchUsers(),
      fetchExperiments(),
      fetchActiveExperiments(),
      fetchTasks(),
    ]);
  }, [ fetchUsers, fetchExperiments, fetchActiveExperiments, fetchTasks ]);

  usePolling(fetchAll);

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  /* Recent Tasks */

  const filteredTasks = useMemo(() => {
    const sorted = [
      ...(experiments || []).map(taskFromExperiment),
      ...(tasks || []).map(taskFromCommandTask),
    ].sort(
      (a, b) => Date.parse(a.lastEvent.date) < Date.parse(b.lastEvent.date) ? 1 : -1,
    );
    const filtered = filterTasks<TaskType, RecentTask>(sorted, filters, users || []);
    return filtered.slice(0, filters.limit);
  }, [ experiments, filters, tasks, users ]);

  const handleFilterChange = useCallback((filters: TaskFilters): void => {
    storage.set(STORAGE_FILTERS_KEY, filters);
    setFilters(filters);
  }, [ setFilters, storage ]);

  return (
    <Page docTitle="Overview" id="dashboard">
      <Section title="Overview">
        <Grid gap={ShirtSize.medium} minItemWidth={120} mode={GridMode.AutoFill}>
          <OverviewStats title="Cluster Allocation">
            {overview[ResourceType.ALL].allocation}<small>%</small>
          </OverviewStats>
          {overview[ResourceType.CUDA].total ? (
            <OverviewStats title="Available GPUs">
              {overview[ResourceType.CUDA].available}
              <small>/{overview[ResourceType.CUDA].total}</small>
            </OverviewStats>
          ) : null}
          {overview[ResourceType.ROCM].total ? (
            <OverviewStats title="Available ROCm GPUs">
              {overview[ResourceType.ROCM].available}
              <small>/{overview[ResourceType.ROCM].total}</small>
            </OverviewStats>
          ) : null}
          {overview[ResourceType.CPU].total ? (
            <OverviewStats title="Available CPUs">
              {overview[ResourceType.CPU].available}
              <small>/{overview[ResourceType.CPU].total}</small>
            </OverviewStats>
          ) : null}
          {activeExperimentCount ? (
            <OverviewStats title="Active Experiments">
              {activeExperimentCount}
            </OverviewStats>
          ) : null}
          {activeTaskTally[CommandType.JupyterLab] ? (
            <OverviewStats title="Active JupyterLabs">
              {activeTaskTally[CommandType.JupyterLab]}
            </OverviewStats>
          ) : null}
          {activeTaskTally[CommandType.TensorBoard] ? (
            <OverviewStats title="Active Tensorboards">
              {activeTaskTally[CommandType.TensorBoard]}
            </OverviewStats>
          ) : null}
          {activeTaskTally[CommandType.Shell] ? (
            <OverviewStats title="Active Shells">
              {activeTaskTally[CommandType.Shell]}
            </OverviewStats>
          ) : null}
          {activeTaskTally[CommandType.Command] ? (
            <OverviewStats title="Active Commands">
              {activeTaskTally[CommandType.Command]}
            </OverviewStats>
          ) : null}
        </Grid>
      </Section>
      <Section
        divider
        loading={!experiments || !tasks}
        options={<TaskFilter filters={filters} onChange={handleFilterChange} />}
        title="Recent Tasks">
        {filteredTasks.length !== 0 ? (
          <Grid gap={ShirtSize.medium} mode={GridMode.AutoFill}>
            {filteredTasks.map((props: RecentTask) => (
              <TaskCard
                curUser={user}
                key={props.id}
                {...props}
              />
            ))}
          </Grid>
        ) : (
          <Message
            title="No recent tasks matching the current filters"
            type={MessageType.Empty}
          />
        )}
      </Section>
    </Page>
  );
};

export default Dashboard;
