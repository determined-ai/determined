import { Input, Table } from 'antd';
import { SelectValue } from 'antd/lib/select';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Icon from 'components/Icon';
import { makeClickHandler } from 'components/Link';
import linkCss from 'components/Link.module.scss';
import Page from 'components/Page';
import StateSelectFilter from 'components/StateSelectFilter';
import Toggle from 'components/Toggle';
import UserSelectFilter from 'components/UserSelectFilter';
import Auth from 'contexts/Auth';
import Users from 'contexts/Users';
import usePolling from 'hooks/usePolling';
import { useRestApiSimple } from 'hooks/useRestApi';
import useStorage from 'hooks/useStorage';
import { getExperimentSummaries } from 'services/api';
import { ExperimentsParams } from 'services/types';
import { ALL_VALUE, Experiment, ExperimentFilters, ExperimentItem } from 'types';
import { filterExperiments, processExperiments } from 'utils/task';

import css from './ExperimentList.module.scss';
import { columns } from './ExperimentList.table';

const defaultFilters: ExperimentFilters = {
  limit: 25,
  showArchived: false,
  states: [ ALL_VALUE ],
  username: undefined,
};

const ExperimentList: React.FC = () => {
  const auth = Auth.useStateContext();
  const users = Users.useStateContext();
  const [ experiments, setExperiments ] = useState<ExperimentItem[]>([]);
  const [ experimentsResponse, requestExperiments ] =
    useRestApiSimple<ExperimentsParams, Experiment[]>(getExperimentSummaries, {});
  const storage = useStorage('experiment-list');
  const initFilters = storage.getWithDefault('filters',
    { ...defaultFilters, username: (auth.user || {}).username });
  const [ filters, setFilters ] = useState<ExperimentFilters>(initFilters);
  const [ search, setSearch ] = useState('');

  const filteredExperiments = useMemo(() => {
    return filterExperiments(experiments, filters, users.data || [], search);
  }, [ experiments, filters, search, users.data ]);

  const fetchExperiments = useCallback((): void => {
    requestExperiments({});
  }, [ requestExperiments ]);

  usePolling(fetchExperiments);

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
            <Toggle prefixLabel="Show Archived" onChange={handleArchiveChange} />
            <StateSelectFilter
              showCommandStates={false}
              value={filters.states}
              onChange={handleStateChange} />
            <UserSelectFilter value={filters.username} onChange={handleUserChange} />
          </div>
        </div>
        <Table
          columns={columns}
          dataSource={filteredExperiments}
          loading={!experimentsResponse.hasLoaded}
          rowClassName={(): string => linkCss.base}
          rowKey="id"
          size="small"
          onRow={handleTableRow} />
      </div>
    </Page>
  );
};

export default ExperimentList;
