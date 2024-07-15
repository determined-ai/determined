import dayjs from 'dayjs';
import Button from 'hew/Button';
import { SyncProvider } from 'hew/LineChart/SyncProvider';
import { useModal } from 'hew/Modal';
import Row from 'hew/Row';
import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Section from 'components/Section';
import { useSettings } from 'hooks/useSettings';
import { getResourceAllocationAggregated } from 'services/api';
import { V1ResourceAllocationAggregatedResponse } from 'services/api-ts-sdk';
import userStore from 'stores/users';
import handleError from 'utils/error';
import { useObservable } from 'utils/observable';

import settingsConfig, { GroupBy, Settings } from './ClusterHistoricalUsage.settings';
import ClusterHistoricalUsageChart from './ClusterHistoricalUsageChart';
import ClusterHistoricalUsageCsvModalComponent, {
  CSVGroupBy,
} from './ClusterHistoricalUsageCsvModalComponent';
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
  const clusterHistoricalUsageCsvModal = useModal(ClusterHistoricalUsageCsvModalComponent);
  const users = Loadable.getOrElse([], loadableUsers);

  useEffect(() => {
    if (isCsvModalVisible) clusterHistoricalUsageCsvModal.open();
  }, [clusterHistoricalUsageCsvModal, isCsvModalVisible]);

  const filters = useMemo(() => {
    const filters: ClusterHistoricalUsageFiltersInterface = {
      fromDate: dayjs().subtract(1 + DEFAULT_RANGE_DAY, 'day'),
      groupBy: GroupBy.Day,
      toDate: dayjs().subtract(1, 'day'),
    };

    if (settings.from) {
      const after = dayjs(settings.from || '');
      if (after.isValid() && after.isBefore(dayjs())) filters.fromDate = after;
    }
    if (settings.to) {
      const before = dayjs(settings.to || '');
      if (before.isValid() && before.isBefore(dayjs())) filters.toDate = before;
    }
    if (settings.groupBy && Object.values(GroupBy).includes(settings.groupBy as GroupBy)) {
      filters.groupBy = settings.groupBy as GroupBy;
    }

    // Validate filter dates.
    const dateDiff = filters.toDate.diff(filters.fromDate, filters.groupBy);
    if (filters.groupBy === GroupBy.Day && (dateDiff >= MAX_RANGE_DAY || dateDiff < 1)) {
      filters.fromDate = filters.toDate.clone().subtract(MAX_RANGE_DAY - 1, 'day');
    }
    if (filters.groupBy === GroupBy.Month && (dateDiff >= MAX_RANGE_MONTH || dateDiff < 1)) {
      filters.fromDate = filters.toDate.clone().subtract(MAX_RANGE_MONTH - 1, 'month');
    }

    return filters;
  }, [settings]);

  const handleFilterChange = useCallback(
    (newFilter: ClusterHistoricalUsageFiltersInterface) => {
      const dateFormat = 'YYYY-MM' + (newFilter.groupBy === GroupBy.Day ? '-DD' : '');
      updateSettings({
        from: newFilter.fromDate.format(dateFormat),
        groupBy: newFilter.groupBy,
        to: newFilter.toDate.format(dateFormat),
      });
    },
    [updateSettings],
  );

  /**
   * When grouped by month force csv modal to display start/end of month.
   */
  let csvAfterDate = filters.fromDate;
  let csvtoDate = filters.toDate;
  if (filters.groupBy === GroupBy.Month) {
    csvAfterDate = csvAfterDate.startOf('month');
    csvtoDate = csvtoDate.endOf('month');
    if (csvtoDate.isAfter(dayjs())) {
      csvtoDate = dayjs().startOf('day');
    }
  }

  const fetchResourceAllocationAggregated = useCallback(async () => {
    try {
      const response = await getResourceAllocationAggregated({
        endDate: filters.toDate,
        period:
          filters.groupBy === GroupBy.Month
            ? 'RESOURCE_ALLOCATION_AGGREGATION_PERIOD_MONTHLY'
            : 'RESOURCE_ALLOCATION_AGGREGATION_PERIOD_DAILY',
        startDate: filters.fromDate,
      });
      setAggRes(Loaded(response));
    } catch (e) {
      handleError(e);
    }
  }, [filters.fromDate, filters.toDate, filters.groupBy]);

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
    <SyncProvider>
      <Row justifyContent="flex-end">
        <ClusterHistoricalUsageFilters value={filters} onChange={handleFilterChange} />
        <Button onClick={() => setIsCsvModalVisible(true)}>Download CSV</Button>
      </Row>
      <Section
        bodyBorder
        loading={Loadable.isNotLoaded(chartSeries)}
        title="Compute Hours Allocated">
        {Loadable.match(chartSeries, {
          Failed: () => null, // TODO inform user if chart fails to load
          Loaded: (series) => (
            <ClusterHistoricalUsageChart
              chartKey={filters.fromDate.unix() + filters.toDate.unix()}
              dateRange={[filters.fromDate.unix(), filters.toDate.unix()]}
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
              chartKey={filters.fromDate.unix() + filters.toDate.unix()}
              dateRange={[filters.fromDate.unix(), filters.toDate.unix()]}
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
              chartKey={filters.fromDate.unix() + filters.toDate.unix()}
              dateRange={[filters.fromDate.unix(), filters.toDate.unix()]}
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
              chartKey={filters.fromDate.unix() + filters.toDate.unix()}
              dateRange={[filters.fromDate.unix(), filters.toDate.unix()]}
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
        <clusterHistoricalUsageCsvModal.Component
          fromDate={csvAfterDate}
          groupBy={CSVGroupBy.Workloads}
          toDate={csvtoDate}
          onVisibleChange={setIsCsvModalVisible}
        />
      )}
    </SyncProvider>
  );
};

export default ClusterHistoricalUsage;
