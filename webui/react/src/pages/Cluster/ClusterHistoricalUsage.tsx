import { Button, Col, Row } from 'antd';
import dayjs from 'dayjs';
import React, { useEffect, useState } from 'react';
import { useHistory } from 'react-router';

import Section from 'components/Section';
import { getResourceAllocationAggregated } from 'services/api';

import css from './ClusterHistoricalUsage.module.scss';
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

export enum GroupBy {
  Day = 'day',
  Month = 'month',
}

const ClusterHistoricalUsage: React.FC = () => {
  const [ chartSeries, setChartSeries ] = useState<ResourceAllocationChartSeries>();
  const [ filters, setFilters ] = useState<ClusterHistoricalUsageFiltersInterface>({
    afterDate: dayjs().subtract(1 + DEFAULT_RANGE_DAY, 'day'),
    beforeDate: dayjs().subtract(1, 'day'),
    groupBy: GroupBy.Day,
  });
  const [ isCsvModalVisible, setIsCsvModalVisible ] = useState<boolean>(false);
  const [ isUrlParsed, setIsUrlParsed ] = useState<boolean>(false);
  const history = useHistory();

  /*
  * When filters changes update the page URL.
  */
  useEffect(() => {
    if (!isUrlParsed) return;

    const dateFormat = 'YYYY-MM' + (filters.groupBy === GroupBy.Day ? '-DD' : '');
    const searchParams = new URLSearchParams;

    // after
    searchParams.append('after', filters.afterDate.format(dateFormat));

    // before
    searchParams.append('before', filters.beforeDate.format(dateFormat));

    // group-by
    searchParams.append('group-by', filters.groupBy);

    history.push('/cluster/historical-usage?' + searchParams.toString());
  }, [ filters, history, isUrlParsed ]);

  /*
   * On first load: if filters are specified in URL, override default.
   */
  useEffect(() => {
    if (isUrlParsed) return;

    const urlSearchParams = new URLSearchParams(history.location.search);

    // after
    const after = dayjs(urlSearchParams.get('after') || '');
    if (after.isValid() && after.isBefore(dayjs())) {
      filters.afterDate = after;
    }

    // before
    const before = dayjs(urlSearchParams.get('before') || '');
    if (before.isValid() && before.isBefore(dayjs())) {
      filters.beforeDate = before;
    }

    // group-by
    const groupBy = urlSearchParams.get('group-by');
    if (groupBy != null && Object.values(GroupBy).includes(groupBy as GroupBy)) {
      filters.groupBy = groupBy as GroupBy;
    }

    // check valid dates
    const dateDiff = filters.beforeDate.diff(filters.afterDate, filters.groupBy);
    if (filters.groupBy === GroupBy.Day && (dateDiff >= MAX_RANGE_DAY || dateDiff < 1)) {
      filters.afterDate = filters.beforeDate.clone().subtract(MAX_RANGE_DAY - 1, 'day');
    }
    if (filters.groupBy === GroupBy.Month && (dateDiff >= MAX_RANGE_MONTH || dateDiff < 1)) {
      filters.afterDate = filters.beforeDate.clone().subtract(MAX_RANGE_MONTH - 1, 'month');
    }

    setFilters(filters);
    setIsUrlParsed(true);
  }, [ filters, history.location.search, isUrlParsed ]);

  /*
   * When grouped by month force csv modal to display start/end of month
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
   * Load chart data
   */
  useEffect(() => {
    if (!isUrlParsed) return;
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
  }, [ filters, isUrlParsed ]);

  return (
    <>
      <Row className={css.filter} justify="end">
        <Col>
          <ClusterHistoricalUsageFilters
            value={filters}
            onChange={setFilters}
          />
        </Col>
        <Col>
          <Button onClick={() => setIsCsvModalVisible(true)}>
            Download CSV
          </Button>
        </Col>
      </Row>

      <Section bodyBorder loading={!chartSeries} title="GPU Hours Allocated">
        { chartSeries && (
          <ClusterHistoricalUsageChart
            groupBy={chartSeries.groupedBy}
            hoursByLabel={chartSeries.hoursTotal}
            time={chartSeries.time}
          />
        ) }
      </Section>

      <Section bodyBorder loading={!chartSeries} title="GPU Hours by User">
        { chartSeries && (
          <ClusterHistoricalUsageChart
            groupBy={chartSeries.groupedBy}
            hoursByLabel={chartSeries.hoursByUsername}
            hoursTotal={chartSeries?.hoursTotal?.total}
            time={chartSeries.time}
          />
        ) }
      </Section>

      <Section bodyBorder loading={!chartSeries} title="GPU Hours by Label">
        { chartSeries && (
          <ClusterHistoricalUsageChart
            groupBy={chartSeries.groupedBy}
            hoursByLabel={chartSeries.hoursByExperimentLabel}
            hoursTotal={chartSeries?.hoursTotal?.total}
            time={chartSeries.time}
          />
        ) }
      </Section>

      <Section bodyBorder loading={!chartSeries} title="GPU Hours by Resource Pool">
        { chartSeries && (
          <ClusterHistoricalUsageChart
            groupBy={chartSeries.groupedBy}
            hoursByLabel={chartSeries.hoursByResourcePool}
            hoursTotal={chartSeries?.hoursTotal?.total}
            time={chartSeries.time}
          />
        ) }
      </Section>

      <Section bodyBorder loading={!chartSeries} title="GPU Hours by Agent Label">
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
