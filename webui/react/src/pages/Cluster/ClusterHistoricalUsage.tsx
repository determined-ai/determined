import { Button, Col, Row } from 'antd';
import dayjs from 'dayjs';
import React, { useCallback, useEffect, useState } from 'react';

import Section from 'components/Section';
import { parseUrl } from 'routes/utils';
import { getResourceAllocationAggregated } from 'services/api';
import { sumArrays } from 'utils/array';

import css from './ClusterHistoricalUsage.module.scss';
import ClusterHistoricalUsageChart from './ClusterHistoricalUsageChart';
import ClusterHistoricalUsageCsvModal from './ClusterHistoricalUsageCsvModal';
import ClusterHistoricalUsageFilters, {
  ClusterHistoricalUsageFiltersInterface,
} from './ClusterHistoricalUsageFilters';
import { generateFakeUsagePeriod } from './tmp';

export const DEFAULT_RANGE_DAY = 14;
export const DEFAULT_RANGE_MONTH = 6;
export const MAX_RANGE_DAY = 31;
export const MAX_RANGE_MONTH = 36;

export enum GroupBy {
  Day = 'day',
  Month = 'month',
}

export interface UsagePeriod {
  hoursByAgentLabel: Record<string, number>,
  hoursByExperimentLabel: Record<string, number>,
  hoursByResourcePool: Record<string, number>,
  hoursByUsername: Record<string, number>,
  hoursTotal: number,
  periodStart: string;
  periodType: GroupBy,
}

interface ChartSeries {
  groupedBy: GroupBy,
  hoursByAgentLabel: Record<string, number[]>,
  hoursByExperimentLabel: Record<string, number[]>,
  hoursByResourcePool: Record<string, number[]>,
  hoursByUsername: Record<string, number[]>,
  hoursTotal: Record<string, number[]>,
  time: string[],
}

const mapToChartSeries = (labelByPeriod: Record<string, number>[]): Record<string, number[]> => {
  // 1. convert [periodIndex: {label: hours}, ...] to {label: {periodIndex: hours}, ...}
  const periodByLabelIndexed: Record<string, Record<number, number>> = {};
  labelByPeriod.forEach((period, periodIndex) => {
    Object.keys(period).forEach(label => {
      periodByLabelIndexed[label] = {
        ...(periodByLabelIndexed[label] || {}),
        [periodIndex]: period[label],
      };
    });
  });

  // 2. convert {label: {periodIndex: hours}, ...} to {label: [hours, ...], ...}
  const periodByLabelIndexedFlat: Record<string, number[]> = {};
  Object.keys(periodByLabelIndexed).forEach(label => {
    periodByLabelIndexedFlat[label] = [];
    for (let i = 0; i < labelByPeriod.length; i++) {
      periodByLabelIndexedFlat[label].push(periodByLabelIndexed[label][i] || 0);
    }
  });

  // 3. find top 5 labels
  const topLabels = Object.keys(periodByLabelIndexedFlat).map(label => {
    const hours = periodByLabelIndexedFlat[label].reduce((acc, val) => acc + val, 0);
    return [ label, hours ];
  })
    .sort((a, b) => ((b[1] as number) - (a[1] as number)))
    .slice(0, 5)
    .map(item => item[0]);

  // 4. sum non-top labels hours into "other labels"
  let ret = {};
  let otherLabels: number[] = [];
  Object.keys(periodByLabelIndexedFlat).forEach(label => {
    if (topLabels.includes(label)) {
      ret = { ...ret, [label]: periodByLabelIndexedFlat[label] };
    } else {
      otherLabels = sumArrays(otherLabels, periodByLabelIndexedFlat[label]);
    }
  });
  if (otherLabels.length > 0) {
    ret = { ...ret, ['other labels']: otherLabels };
  }

  return ret;
};

const ClusterHistoricalUsage: React.FC = () => {
  const [ chartSeries, setChartSeries ] = useState<ChartSeries|null>(null);
  const [ filters, setFilters ] = useState<ClusterHistoricalUsageFiltersInterface>({
    afterDate: dayjs().subtract(1 + DEFAULT_RANGE_DAY, 'day'),
    beforeDate: dayjs().subtract(1, 'day'),
    groupBy: GroupBy.Day,
  });
  const [ isCsvModalVisible, setIsCsvModalVisible ] = useState<boolean>(false);
  const [ isUrlParsed, setIsUrlParsed ] = useState<boolean>(false);

  /*
  * When filters changes update the page URL.
  */
  useEffect(() => {
    if (!isUrlParsed) return;

    const dateFormat = 'YYYY-MM' + (filters.groupBy === GroupBy.Day ? '-DD' : '');
    const searchParams = new URLSearchParams;
    const url = parseUrl(window.location.href);

    // after
    searchParams.append('after', filters.afterDate.format(dateFormat));

    // before
    searchParams.append('before', filters.beforeDate.format(dateFormat));

    // group-by
    searchParams.append('group-by', filters.groupBy);

    window.history.pushState(
      {},
      '',
      url.origin + url.pathname + '?' + searchParams.toString(),
    );
  }, [ filters, isUrlParsed ]);

  /*
   * On first load: if filters are specified in URL, override default.
   */
  useEffect(() => {
    if (isUrlParsed) return;

    const urlSearchParams = parseUrl(window.location.href).searchParams;

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
  }, [ filters, isUrlParsed ]);

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
  // const fetchResourceAllocationData =
  //   useCallback(async (reqFilters: ClusterHistoricalUsageFiltersInterface) => {
  //     setChartSeries(null);
  //
  //     const res = await getResourceAllocationAggregated({
  //       endDate: reqFilters.beforeDate,
  //       period: (reqFilters.groupBy === GroupBy.Month
  //         ? 'RESOURCE_ALLOCATION_AGGREGATION_PERIOD_MONTHLY'
  //         : 'RESOURCE_ALLOCATION_AGGREGATION_PERIOD_DAILY'),
  //       startDate: reqFilters.afterDate,
  //     });
  //
  //     console.log('res', res);
  //   }, []);
  useEffect(() => {
    if (!isUrlParsed) return;
    setChartSeries(null);

    const timeout = setTimeout(() => {
      const apiRes = generateFakeUsagePeriod(
        filters.groupBy,
        filters.afterDate,
        filters.beforeDate,
      );
      const chartSeries = {
        groupedBy: filters.groupBy,
        hoursByAgentLabel: mapToChartSeries(apiRes.map(item => item.hoursByAgentLabel)),
        hoursByExperimentLabel: mapToChartSeries(apiRes.map(item => item.hoursByExperimentLabel)),
        hoursByResourcePool: mapToChartSeries(apiRes.map(item => item.hoursByResourcePool)),
        hoursByUsername: mapToChartSeries(apiRes.map(item => item.hoursByUsername)),
        hoursTotal: { total: apiRes.map(item => item.hoursTotal) },
        time: apiRes.map(item => item.periodStart),
      };
      setChartSeries(chartSeries);
    }, 1000);

    return () => clearTimeout(timeout);
  }, [ filters.afterDate, filters.beforeDate, filters.groupBy, isUrlParsed ]);

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

      <Section bodyBorder loading={chartSeries == null} title="GPU Hours Allocated">
        { chartSeries != null && (
          <ClusterHistoricalUsageChart
            groupBy={chartSeries.groupedBy}
            hoursByLabel={chartSeries.hoursTotal}
            time={chartSeries.time}
          />
        ) }
      </Section>

      <Section bodyBorder loading={chartSeries == null} title="GPU Hours by User">
        { chartSeries != null && (
          <ClusterHistoricalUsageChart
            groupBy={chartSeries.groupedBy}
            hoursByLabel={chartSeries.hoursByUsername}
            hoursTotal={chartSeries?.hoursTotal?.total}
            time={chartSeries.time}
          />
        ) }
      </Section>

      <Section bodyBorder loading={chartSeries == null} title="GPU Hours by Label">
        { chartSeries != null && (
          <ClusterHistoricalUsageChart
            groupBy={chartSeries.groupedBy}
            hoursByLabel={chartSeries.hoursByExperimentLabel}
            hoursTotal={chartSeries?.hoursTotal?.total}
            time={chartSeries.time}
          />
        ) }
      </Section>

      <Section bodyBorder loading={chartSeries == null} title="GPU Hours by Resource Pool">
        { chartSeries != null && (
          <ClusterHistoricalUsageChart
            groupBy={chartSeries.groupedBy}
            hoursByLabel={chartSeries.hoursByResourcePool}
            hoursTotal={chartSeries?.hoursTotal?.total}
            time={chartSeries.time}
          />
        ) }
      </Section>

      <Section bodyBorder loading={chartSeries == null} title="GPU Hours by Agent Label">
        { chartSeries != null && (
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
