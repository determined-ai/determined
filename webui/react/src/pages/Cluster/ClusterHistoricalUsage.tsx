import { Space } from 'antd';
import dayjs from 'dayjs';
import { useObservable } from 'micro-observables';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Button from 'components/kit/Button';
import Section from 'components/Section';
import { SyncProvider } from 'components/UPlot/SyncProvider';
import { useSettings } from 'hooks/useSettings';
import { getResourceAllocationAggregated } from 'services/api';
import { V1ResourceAllocationAggregatedResponse } from 'services/api-ts-sdk';
import usersStore from 'stores/usersObserve';
import { DetailedUser } from 'types';
import handleError from 'utils/error';
import { Loadable } from 'utils/loadable';

import css from './ClusterHistoricalUsage.module.scss';
import settingsConfig, { GroupBy, Settings } from './ClusterHistoricalUsage.settings';
import ClusterHistoricalUsageChart from './ClusterHistoricalUsageChart';
import ClusterHistoricalUsageCsvModal, { CSVGroupBy } from './ClusterHistoricalUsageCsvModal';
import ClusterHistoricalUsageFilters, {
  ClusterHistoricalUsageFiltersInterface,
} from './ClusterHistoricalUsageFilters';
import { mapResourceAllocationApiToChartSeries } from './utils';

export const DEFAULT_RANGE_DAY = 14;
export const DEFAULT_RANGE_MONTH = 6;
export const MAX_RANGE_DAY = 31;
export const MAX_RANGE_MONTH = 36;

const ClusterHistoricalUsage: React.FC = () => {
  const [aggRes, setAggRes] = useState<V1ResourceAllocationAggregatedResponse>({
    resourceEntries: [],
  });
  const [isCsvModalVisible, setIsCsvModalVisible] = useState<boolean>(false);
  const { settings, updateSettings } = useSettings<Settings>(settingsConfig);
  const loadableUsers = useObservable(usersStore.getUsers());
  const users: Readonly<DetailedUser[]> = Loadable.match(loadableUsers, {
    Loaded: (usersPagination) => usersPagination.users,
    NotLoaded: () => [],
  }); // TODO: handle loading state

  const filters = useMemo(() => {
    const filters: ClusterHistoricalUsageFiltersInterface = {
      afterDate: dayjs().subtract(1 + DEFAULT_RANGE_DAY, 'day'),
      beforeDate: dayjs().subtract(1, 'day'),
      groupBy: GroupBy.Day,
    };

    if (settings.after) {
      const after = dayjs(settings.after || '');
      if (after.isValid() && after.isBefore(dayjs())) filters.afterDate = after;
    }
    if (settings.before) {
      const before = dayjs(settings.before || '');
      if (before.isValid() && before.isBefore(dayjs())) filters.beforeDate = before;
    }
    if (settings.groupBy && Object.values(GroupBy).includes(settings.groupBy as GroupBy)) {
      filters.groupBy = settings.groupBy as GroupBy;
    }

    // Validate filter dates.
    const dateDiff = filters.beforeDate.diff(filters.afterDate, filters.groupBy);
    if (filters.groupBy === GroupBy.Day && (dateDiff >= MAX_RANGE_DAY || dateDiff < 1)) {
      filters.afterDate = filters.beforeDate.clone().subtract(MAX_RANGE_DAY - 1, 'day');
    }
    if (filters.groupBy === GroupBy.Month && (dateDiff >= MAX_RANGE_MONTH || dateDiff < 1)) {
      filters.afterDate = filters.beforeDate.clone().subtract(MAX_RANGE_MONTH - 1, 'month');
    }

    return filters;
  }, [settings]);

  const handleFilterChange = useCallback(
    (newFilter: ClusterHistoricalUsageFiltersInterface) => {
      const dateFormat = 'YYYY-MM' + (newFilter.groupBy === GroupBy.Day ? '-DD' : '');
      updateSettings({
        after: newFilter.afterDate.format(dateFormat),
        before: newFilter.beforeDate.format(dateFormat),
        groupBy: newFilter.groupBy,
      });
    },
    [updateSettings],
  );

  /**
   * When grouped by month force csv modal to display start/end of month.
   */
  let csvAfterDate = filters.afterDate;
  let csvBeforeDate = filters.beforeDate;
  if (filters.groupBy === GroupBy.Month) {
    csvAfterDate = csvAfterDate.startOf('month');
    csvBeforeDate = csvBeforeDate.endOf('month');
    if (csvBeforeDate.isAfter(dayjs())) {
      csvBeforeDate = dayjs().startOf('day');
    }
  }

  const fetchResourceAllocationAggregated = useCallback(async () => {
    try {
      const res = await getResourceAllocationAggregated({
        endDate: filters.beforeDate,
        period:
          filters.groupBy === GroupBy.Month
            ? 'RESOURCE_ALLOCATION_AGGREGATION_PERIOD_MONTHLY'
            : 'RESOURCE_ALLOCATION_AGGREGATION_PERIOD_DAILY',
        startDate: filters.afterDate,
      });
      setAggRes(res);
    } catch (e) {
      handleError(e);
    }
  }, [filters.afterDate, filters.beforeDate, filters.groupBy]);

  const chartSeries = useMemo(() => {
    return mapResourceAllocationApiToChartSeries(aggRes.resourceEntries, filters.groupBy, users);
  }, [aggRes.resourceEntries, filters.groupBy, users]);

  useEffect(() => {
    fetchResourceAllocationAggregated();
  }, [fetchResourceAllocationAggregated]);

  return (
    <div className={css.base}>
      <SyncProvider>
        <Space align="end" className={css.filters}>
          <ClusterHistoricalUsageFilters value={filters} onChange={handleFilterChange} />
          <Button onClick={() => setIsCsvModalVisible(true)}>Download CSV</Button>
        </Space>
        <Section bodyBorder loading={!chartSeries} title="Compute Hours Allocated">
          {chartSeries && (
            <ClusterHistoricalUsageChart
              groupBy={chartSeries.groupedBy}
              hoursByLabel={chartSeries.hoursTotal}
              time={chartSeries.time}
            />
          )}
        </Section>
        <Section bodyBorder loading={!chartSeries} title="Compute Hours by User">
          {chartSeries && (
            <ClusterHistoricalUsageChart
              groupBy={chartSeries.groupedBy}
              hoursByLabel={chartSeries.hoursByUsername}
              hoursTotal={chartSeries?.hoursTotal?.total}
              time={chartSeries.time}
            />
          )}
        </Section>
        <Section bodyBorder loading={!chartSeries} title="Compute Hours by Label">
          {chartSeries && (
            <ClusterHistoricalUsageChart
              groupBy={chartSeries.groupedBy}
              hoursByLabel={chartSeries.hoursByExperimentLabel}
              hoursTotal={chartSeries?.hoursTotal?.total}
              time={chartSeries.time}
            />
          )}
        </Section>
        <Section bodyBorder loading={!chartSeries} title="Compute Hours by Resource Pool">
          {chartSeries && (
            <ClusterHistoricalUsageChart
              groupBy={chartSeries.groupedBy}
              hoursByLabel={chartSeries.hoursByResourcePool}
              hoursTotal={chartSeries?.hoursTotal?.total}
              time={chartSeries.time}
            />
          )}
        </Section>
        {isCsvModalVisible && (
          <ClusterHistoricalUsageCsvModal
            afterDate={csvAfterDate}
            beforeDate={csvBeforeDate}
            groupBy={CSVGroupBy.Workloads}
            onVisibleChange={setIsCsvModalVisible}
          />
        )}
      </SyncProvider>
    </div>
  );
};

export default ClusterHistoricalUsage;
