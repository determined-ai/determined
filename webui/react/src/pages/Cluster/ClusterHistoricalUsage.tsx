import { Space } from 'antd';
import dayjs from 'dayjs';
import Button from 'hew/Button';
import { SyncProvider } from 'hew/LineChart/SyncProvider';
import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Section from 'components/Section';
import { useSettings } from 'hooks/useSettings';
import { getResourceAllocationAggregated } from 'services/api';
import { V1ResourceAllocationAggregatedResponse } from 'services/api-ts-sdk';
import userStore from 'stores/users';
import handleError from 'utils/error';
import { useObservable } from 'utils/observable';

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
  const [aggRes, setAggRes] = useState<Loadable<V1ResourceAllocationAggregatedResponse>>(NotLoaded);
  const [isCsvModalVisible, setIsCsvModalVisible] = useState<boolean>(false);
  const { settings, updateSettings } = useSettings<Settings>(settingsConfig);
  const loadableUsers = useObservable(userStore.getUsers());
  const users = Loadable.getOrElse([], loadableUsers);

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
      const response = await getResourceAllocationAggregated({
        endDate: filters.beforeDate,
        period:
          filters.groupBy === GroupBy.Month
            ? 'RESOURCE_ALLOCATION_AGGREGATION_PERIOD_MONTHLY'
            : 'RESOURCE_ALLOCATION_AGGREGATION_PERIOD_DAILY',
        startDate: filters.afterDate,
      });
      setAggRes(Loaded(response));
    } catch (e) {
      handleError(e);
    }
  }, [filters.afterDate, filters.beforeDate, filters.groupBy]);

  const chartSeries = useMemo(() => {
    return Loadable.map(aggRes, (response) => {
      return mapResourceAllocationApiToChartSeries(
        response.resourceEntries,
        filters.groupBy,
        users,
      );
    });
  }, [aggRes, filters.groupBy, users]);

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
        <Section
          bodyBorder
          loading={Loadable.isNotLoaded(chartSeries)}
          title="Compute Hours Allocated">
          {Loadable.match(chartSeries, {
            Failed: () => null, // TODO inform user if chart fails to load
            Loaded: (series) => (
              <ClusterHistoricalUsageChart
                chartKey={filters.afterDate.unix() + filters.beforeDate.unix()}
                dateRange={[filters.afterDate.unix(), filters.beforeDate.unix()]}
                groupBy={series.groupedBy}
                hoursByLabel={series.hoursTotal}
                time={series.time}
              />
            ),
            NotLoaded: () => null,
          })}
        </Section>
        <Section
          bodyBorder
          loading={Loadable.isNotLoaded(Loadable.all([loadableUsers, chartSeries]))}
          title="Compute Hours by User">
          {Loadable.match(chartSeries, {
            Failed: () => null, // TODO inform user if chart fails to load
            Loaded: (series) => (
              <ClusterHistoricalUsageChart
                chartKey={filters.afterDate.unix() + filters.beforeDate.unix()}
                dateRange={[filters.afterDate.unix(), filters.beforeDate.unix()]}
                groupBy={series.groupedBy}
                hoursByLabel={{
                  ...series.hoursByUsername,
                  total: series?.hoursTotal?.total,
                }}
                time={series.time}
              />
            ),
            NotLoaded: () => null,
          })}
        </Section>
        <Section
          bodyBorder
          loading={Loadable.isNotLoaded(chartSeries)}
          title="Compute Hours by Label">
          {Loadable.match(chartSeries, {
            Failed: () => null, // TODO inform user if chart fails to load
            Loaded: (series) => (
              <ClusterHistoricalUsageChart
                chartKey={filters.afterDate.unix() + filters.beforeDate.unix()}
                dateRange={[filters.afterDate.unix(), filters.beforeDate.unix()]}
                groupBy={series.groupedBy}
                hoursByLabel={{
                  ...series.hoursByExperimentLabel,
                  total: series?.hoursTotal?.total,
                }}
                time={series.time}
              />
            ),
            NotLoaded: () => null,
          })}
        </Section>
        <Section
          bodyBorder
          loading={Loadable.isNotLoaded(chartSeries)}
          title="Compute Hours by Resource Pool">
          {Loadable.match(chartSeries, {
            Failed: () => null, // TODO inform user if chart fails to load
            Loaded: (series) => (
              <ClusterHistoricalUsageChart
                chartKey={filters.afterDate.unix() + filters.beforeDate.unix()}
                dateRange={[filters.afterDate.unix(), filters.beforeDate.unix()]}
                groupBy={series.groupedBy}
                hoursByLabel={{
                  ...series.hoursByResourcePool,
                  total: series?.hoursTotal?.total,
                }}
                time={series.time}
              />
            ),
            NotLoaded: () => null,
          })}
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
