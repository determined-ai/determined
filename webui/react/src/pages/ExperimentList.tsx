import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Button, Input, Modal, Table } from 'antd';
import { SelectValue } from 'antd/lib/select';
import { ColumnType } from 'antd/lib/table/interface';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Icon from 'components/Icon';
import { makeClickHandler } from 'components/Link';
import linkCss from 'components/Link.module.scss';
import Page from 'components/Page';
import StateSelectFilter from 'components/StateSelectFilter';
import TableBatch from 'components/TableBatch';
import TagList from 'components/TagList';
import Toggle from 'components/Toggle';
import UserSelectFilter from 'components/UserSelectFilter';
import Auth from 'contexts/Auth';
import Users from 'contexts/Users';
import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import usePolling from 'hooks/usePolling';
import useRestApi from 'hooks/useRestApi';
import useStorage from 'hooks/useStorage';
import { setupUrlForDev } from 'routes';
import {
  archiveExperiment, getExperimentSummaries, killExperiment, launchTensorboard, setExperimentState,
} from 'services/api';
import { patchExperiment } from 'services/api';
import { ExperimentsParams } from 'services/types';
import {
  ALL_VALUE, Command, Experiment, ExperimentFilters, ExperimentItem, RunState, TBSourceType,
} from 'types';
import { alphanumericSorter } from 'utils/data';
import { openBlank } from 'utils/routes';
import { filterExperiments, processExperiments } from 'utils/task';
import { cancellableRunStates, isTaskKillable, terminalRunStates, waitPageUrl } from 'utils/types';

import css from './ExperimentList.module.scss';
import { columns as experimentColumns } from './ExperimentList.table';

enum Action {
  Activate = 'Activate',
  Archive = 'Archive',
  Cancel = 'Cancel',
  Kill = 'Kill',
  Pause = 'Pause',
  OpenTensorBoard = 'OpenTensorboard',
  Unarchive = 'Unarchive',
}

const defaultFilters: ExperimentFilters = {
  limit: 25,
  showArchived: false,
  states: [ ALL_VALUE ],
  username: undefined,
};

const columns = [ ...experimentColumns ];

const ExperimentList: React.FC = () => {
  const auth = Auth.useStateContext();
  const users = Users.useStateContext();
  const [ experiments, setExperiments ] = useState<ExperimentItem[]>([]);
  const [ experimentsResponse, triggerExperimentsRequest ] =
    useRestApi<ExperimentsParams, Experiment[]>(getExperimentSummaries, {});
  const storage = useStorage('experiment-list');
  const initFilters = storage.getWithDefault(
    'filters',
    { ...defaultFilters, username: (auth.user || {}).username },
  );
  const [ filters, setFilters ] = useState<ExperimentFilters>(initFilters);
  const [ search, setSearch ] = useState('');
  const [ selectedRowKeys, setSelectedRowKeys ] = useState<string[]>([]);

  const filteredExperiments = useMemo(() => {
    return filterExperiments(experiments, filters, users.data || [], search);
  }, [ experiments, filters, search, users.data ]);

  const showBatch = selectedRowKeys.length !== 0;

  const experimentMap = useMemo(() => {
    return experiments.reduce((acc, task) => {
      acc[task.id] = task;
      return acc;
    }, {} as Record<string, ExperimentItem>);
  }, [ experiments ]);

  const selectedExperiments = useMemo(() => {
    return selectedRowKeys.map(key => experimentMap[key]);
  }, [ experimentMap, selectedRowKeys ]);

  const {
    hasActivatable,
    hasArchivable,
    hasCancelable,
    hasKillable,
    hasPausable,
    hasUnarchivable,
  } = useMemo(() => {
    const tracker = {
      hasActivatable: false,
      hasArchivable: false,
      hasCancelable: false,
      hasKillable: false,
      hasPausable: false,
      hasUnarchivable: false,
    };
    for (let i = 0; i < selectedExperiments.length; i++) {
      const experiment = selectedExperiments[i];
      const isArchivable = !experiment.archived && terminalRunStates.has(experiment.state);
      const isCancelable = cancellableRunStates.includes(experiment.state);
      const isKillable = isTaskKillable(experiment);
      const isActivatable = experiment.state === RunState.Paused;
      const isPausable = experiment.state === RunState.Active;
      if (!tracker.hasArchivable && isArchivable) tracker.hasArchivable = true;
      if (!tracker.hasUnarchivable && experiment.archived) tracker.hasUnarchivable = true;
      if (!tracker.hasCancelable && isCancelable) tracker.hasCancelable = true;
      if (!tracker.hasKillable && isKillable) tracker.hasKillable = true;
      if (!tracker.hasActivatable && isActivatable) tracker.hasActivatable = true;
      if (!tracker.hasPausable && isPausable) tracker.hasPausable = true;
    }
    return tracker;
  }, [ selectedExperiments ]);

  const fetchExperiments = useCallback((): void => {
    triggerExperimentsRequest({});
  }, [ triggerExperimentsRequest ]);

  usePolling(fetchExperiments);

  const setLabels = useCallback((id) => {
    return (labels: string[]) => {
      patchExperiment({
        body: {
          labels: labels.reduce((a, c) => ({ ...a, [c]: true }), {}),
        },
        experimentId: id })
        .then(fetchExperiments);
    };

  }, [ fetchExperiments ]);

  useEffect(() => {
    const nameColumn: ColumnType<ExperimentItem> = {
      dataIndex: 'name',
      render: function nameRenderer(_, record) {
        return (
          <div className={css.nameColumn}>
            {record.name || ''}
            <TagList className={css.tagList}
              setTags={setLabels(record.id)} tags={record.config.labels || []} />
          </div>
        );
      },
      sorter: (a: ExperimentItem, b: ExperimentItem): number => alphanumericSorter(a.name, b.name),
      title: 'Name',
    };

    const existingCol = columns.find(col => col.dataIndex === nameColumn.dataIndex);
    if (!existingCol) columns.splice(1, 0, nameColumn);
  }, [ setLabels ]);

  useEffect(() => {
    const experiments = processExperiments(experimentsResponse.data || [], users.data || []);
    setExperiments(experiments);
  }, [ experimentsResponse, setExperiments, users ]);

  const handleSearchChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setSearch(e.target.value || '');
  }, []);

  const handleFilterChange = useCallback((filters: ExperimentFilters): void => {
    storage.set('filters', filters);
    setFilters(filters);
  }, [ setFilters, storage ]);

  const handleArchiveChange = useCallback((value: boolean): void => {
    handleFilterChange({ ...filters, showArchived: value });
  }, [ filters, handleFilterChange ]);

  const handleStateChange = useCallback((value: SelectValue): void => {
    if (typeof value !== 'string') return;
    handleFilterChange({ ...filters, states: [ value ] });
  }, [ filters, handleFilterChange ]);

  const handleUserChange = useCallback((value: SelectValue) => {
    const username = value === ALL_VALUE ? undefined : value as string;
    handleFilterChange({ ...filters, username });
  }, [ filters, handleFilterChange ]);

  const sendBatchActions = useCallback((action: Action): Promise<void[] | Command> => {
    if (action === Action.OpenTensorBoard) {
      return launchTensorboard({
        ids: selectedExperiments.map(experiment => experiment.id),
        type: TBSourceType.Experiment,
      });
    }
    return Promise.all(selectedExperiments
      .map(experiment => {
        switch (action) {
          case Action.Activate:
            return setExperimentState({ experimentId: experiment.id, state: RunState.Active });
          case Action.Archive:
            return archiveExperiment(experiment.id, true);
          case Action.Cancel:
            return setExperimentState({ experimentId: experiment.id, state: RunState.Canceled });
          case Action.Kill:
            return killExperiment({ experimentId: experiment.id });
          case Action.Pause:
            return setExperimentState({ experimentId: experiment.id, state: RunState.Paused });
          case Action.Unarchive:
            return archiveExperiment(experiment.id, false);
          default:
            return Promise.resolve();
        }
      }));
  }, [ selectedExperiments ]);

  const handleBatchAction = useCallback(async (action: Action) => {
    try {
      const result = await sendBatchActions(action);
      if (action === Action.OpenTensorBoard) {
        const url = waitPageUrl(result as Command);
        if (url) openBlank(setupUrlForDev(url));
      }

      // Refetch experiment list to get updates based on batch action.
      await fetchExperiments();
    } catch (e) {
      const publicSubject = action === Action.OpenTensorBoard ?
        'Unable to Open TensorBoard for Selected Experiments' :
        `Unable to ${action} Selected Experiments`;
      handleError({
        error: e,
        level: ErrorLevel.Error,
        message: e.message,
        publicMessage: 'Please try again later.',
        publicSubject,
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [ fetchExperiments, sendBatchActions ]);

  const handleConfirmation = useCallback((action: Action) => {
    Modal.confirm({
      content: `
        Are you sure you want to ${action.toLocaleLowerCase()}
        all the eligible selected experiments?
      `,
      icon: <ExclamationCircleOutlined />,
      okText: action,
      onOk: () => handleBatchAction(action),
      title: 'Confirm Batch Action',
    });
  }, [ handleBatchAction ]);

  const handleTableRowSelect = useCallback(rowKeys => setSelectedRowKeys(rowKeys), []);

  const handleTableRow = useCallback((record: ExperimentItem) => ({
    onClick: makeClickHandler(record.url as string),
  }), []);

  return (
    <Page title="Experiments">
      <div className={css.base}>
        <div className={css.header}>
          <Input
            allowClear
            className={css.search}
            placeholder="name"
            prefix={<Icon name="search" size="small" />}
            onChange={handleSearchChange} />
          <div className={css.filters}>
            <Toggle
              checked={filters.showArchived}
              prefixLabel="Show Archived"
              onChange={handleArchiveChange} />
            <StateSelectFilter
              showCommandStates={false}
              value={filters.states}
              onChange={handleStateChange} />
            <UserSelectFilter value={filters.username} onChange={handleUserChange} />
          </div>
        </div>
        <TableBatch message="Apply batch operations to multiple experiments." show={showBatch}>
          <Button
            type="primary"
            onClick={(): Promise<void> => handleBatchAction(Action.OpenTensorBoard)}>
              Open TensorBoard
          </Button>
          <Button
            disabled={!hasActivatable}
            onClick={(): void => handleConfirmation(Action.Activate)}>Activate</Button>
          <Button
            disabled={!hasPausable}
            onClick={(): void => handleConfirmation(Action.Pause)}>Pause</Button>
          <Button
            disabled={!hasArchivable}
            onClick={(): void => handleConfirmation(Action.Archive)}>Archive</Button>
          <Button
            disabled={!hasUnarchivable}
            onClick={(): void => handleConfirmation(Action.Unarchive)}>Unarchive</Button>
          <Button
            disabled={!hasCancelable}
            onClick={(): void => handleConfirmation(Action.Cancel)}>Cancel</Button>
          <Button
            danger
            disabled={!hasKillable}
            type="primary"
            onClick={(): void => handleConfirmation(Action.Kill)}>Kill</Button>
        </TableBatch>
        <Table
          columns={columns}
          dataSource={filteredExperiments}
          loading={!experimentsResponse.hasLoaded}
          rowClassName={(): string => linkCss.base}
          rowKey="id"
          rowSelection={{ onChange: handleTableRowSelect, selectedRowKeys }}
          size="small"
          onRow={handleTableRow}
        />
      </div>
    </Page>
  );
};

export default ExperimentList;
