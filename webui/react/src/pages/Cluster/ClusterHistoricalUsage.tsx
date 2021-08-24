import { Button, Col, Row } from 'antd';
import dayjs from 'dayjs';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import Section from 'components/Section';
import useResize from 'hooks/useResize';
import useSettings from 'hooks/useSettings';
import { getResourceAllocationAggregated } from 'services/api';

import css from './ClusterHistoricalUsage.module.scss';
import settingsConfig, { GroupBy, Settings } from './ClusterHistoricalUsage.settings';
import ClusterHistoricalUsageChart from './ClusterHistoricalUsageChart';
import ClusterHistoricalUsageCsvModal from './ClusterHistoricalUsageCsvModal';
import ClusterHistoricalUsageFilters, {
  ClusterHistoricalUsageFiltersInterface,
} from './ClusterHistoricalUsageFilters';
import { mapResourceAllocationApiToChartSeries, ResourceAllocationChartSeries } from './utils';

export const DEFAULT_RANGE_DAY = 14;
export const DEFAULT_RANGE_MONTH = 6;
export const MAX_RANGE_DAY = 31;
export const MAX_RANGE_MONTH = 36;

const ClusterHistoricalUsage: React.FC = () => {
  const [ chartSeries, setChartSeries ] = useState<ResourceAllocationChartSeries>();
  const [ isCsvModalVisible, setIsCsvModalVisible ] = useState<boolean>(false);
  const filterBarRef = useRef<HTMLDivElement>(null);
  const { settings, updateSettings } = useSettings<Settings>(settingsConfig);

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
  }, [ settings ]);

  const handleFilterChange = useCallback((newFilter: ClusterHistoricalUsageFiltersInterface) => {
    const dateFormat = 'YYYY-MM' + (newFilter.groupBy === GroupBy.Day ? '-DD' : '');
    updateSettings({
      after: newFilter.afterDate.format(dateFormat),
      before: newFilter.beforeDate.format(dateFormat),
      groupBy: newFilter.groupBy,
    });
  }, [ updateSettings ]);

  /* On first load: make sure filter bar doesn't overlap charts */
  const filterBarResize = useResize(filterBarRef);
  useEffect(() => {
    if (!filterBarRef.current || !filterBarRef.current.parentElement) return;
    filterBarRef.current.parentElement.style.height = filterBarResize.height + 'px';
  }, [ filterBarRef, filterBarResize ]);

  /*
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

  /*
   * Load chart data.
   */
  useEffect(() => {
    setChartSeries(undefined);

    (async () => {
      const res = await getResourceAllocationAggregated({
        endDate: filters.beforeDate,
        period: (filters.groupBy === GroupBy.Month
          ? 'RESOURCE_ALLOCATION_AGGREGATION_PERIOD_MONTHLY'
          : 'RESOURCE_ALLOCATION_AGGREGATION_PERIOD_DAILY'),
        startDate: filters.afterDate,
      });

      setChartSeries(
        mapResourceAllocationApiToChartSeries(res.resourceEntries, filters.groupBy),
      );
    })();
  }, [ filters ]);

  return (
    <>
      <div>
        <Row className={css.filter} justify="end" ref={filterBarRef}>
          <Col>
            <ClusterHistoricalUsageFilters value={filters} onChange={handleFilterChange} />
          </Col>
          <Col>
            <Button onClick={() => setIsCsvModalVisible(true)}>
              Download CSV
            </Button>
          </Col>
        </Row>
      </div>

      <Section bodyBorder loading={!chartSeries} title="Compute Hours Allocated">
        { chartSeries && (
          <ClusterHistoricalUsageChart
            groupBy={chartSeries.groupedBy}
            hoursByLabel={chartSeries.hoursTotal}
            time={chartSeries.time}
          />
        ) }
      </Section>

      <Section bodyBorder loading={!chartSeries} title="Compute Hours by User">
        { chartSeries && (
          <ClusterHistoricalUsageChart
            groupBy={chartSeries.groupedBy}
            hoursByLabel={chartSeries.hoursByUsername}
            hoursTotal={chartSeries?.hoursTotal?.total}
            time={chartSeries.time}
          />
        ) }
      </Section>

      <Section bodyBorder loading={!chartSeries} title="Compute Hours by Label">
        { chartSeries && (
          <ClusterHistoricalUsageChart
            groupBy={chartSeries.groupedBy}
            hoursByLabel={chartSeries.hoursByExperimentLabel}
            hoursTotal={chartSeries?.hoursTotal?.total}
            time={chartSeries.time}
          />
        ) }
      </Section>

      <Section bodyBorder loading={!chartSeries} title="Compute Hours by Resource Pool">
        { chartSeries && (
          <ClusterHistoricalUsageChart
            groupBy={chartSeries.groupedBy}
            hoursByLabel={chartSeries.hoursByResourcePool}
            hoursTotal={chartSeries?.hoursTotal?.total}
            time={chartSeries.time}
          />
        ) }
      </Section>

      <Section bodyBorder loading={!chartSeries} title="Compute Hours by Agent Label">
        { chartSeries && (
          <ClusterHistoricalUsageChart
            groupBy={chartSeries.groupedBy}
            hoursByLabel={chartSeries.hoursByAgentLabel}
            hoursTotal={chartSeries?.hoursTotal?.total}
            time={chartSeries.time}
          />
        ) }
      </Section>

      {isCsvModalVisible && (
        <ClusterHistoricalUsageCsvModal
          afterDate={csvAfterDate}
          beforeDate={csvBeforeDate}
          onVisibleChange={setIsCsvModalVisible}
        />
      )}
    </>
  );
};

export default ClusterHistoricalUsage;
