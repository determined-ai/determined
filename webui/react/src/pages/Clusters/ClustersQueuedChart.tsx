import { Radio } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';

import Page from 'components/Page';
import usePolling from 'hooks/usePolling';
import { getJobQStats } from 'services/api';
import * as Api from 'services/api-ts-sdk';
import { DURATION_DAY, secondToHour } from 'utils/datetime';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';

import ClusterHistoricalUsageChart from '../Cluster/ClusterHistoricalUsageChart';

import css from './ClustersQueuedChart.module.scss';
import { JobQueuedTimeChartSeries } from './utils';

interface Props {
  poolName: string;
}

const ClustersQueuedChart: React.FC<Props> = ({ poolName }:Props) => {
  const [ canceler ] = useState(new AbortController());
  const [ queuedStats, setQueuedStats ] = useState<JobQueuedTimeChartSeries>();
  const [ viewDays, setViewDays ] = useState(7);
  const [ isLoading, setIsLoading ] = useState(false);

  const fetchStats = useCallback(async () => {
    try {
      const promises = [
        getJobQStats({}, { signal: canceler.signal }),
      ] as [ Promise<Api.V1GetJobQueueStatsResponse> ];
      const [ stats ] = await Promise.all(promises);
      const pool = stats.results.find(p => p.resourcePool === poolName);

      if(!pool) return;
      const { aggregates } = pool;
      if(aggregates) {
        const agg = aggregates.filter(
          item => Date.parse(item.periodStart) >= Date.now() - viewDays * DURATION_DAY,
        );
        setQueuedStats({
          hoursAverage: { average: agg.map(item => secondToHour(item.seconds)) },
          time: agg.map(item => item.periodStart),
        });
      }

    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicSubject: 'Unable to fetch job queue stats.',
        silent: false,
        type: ErrorType.Server,
      });
    } finally {
      setIsLoading(false);
    }
  }, [ canceler.signal, viewDays, poolName ]);

  usePolling(fetchStats);

  useEffect(() => {
    setIsLoading(true);
    fetchStats();
    return () => canceler.abort();
  }, [ canceler, fetchStats ]);

  if(!queuedStats) return <div />;
  return (
    <Page loading={isLoading} title="Daily Avg Queued Time (In Hours)">
      <Radio.Group
        className={css.filter}
        value={viewDays}
        onChange={e => setViewDays(e.target.value)}>
        <Radio.Button value={7}>7 days</Radio.Button>
        <Radio.Button value={30}>30 days</Radio.Button>
      </Radio.Group>
      <ClusterHistoricalUsageChart
        hoursByLabel={queuedStats.hoursAverage}
        label="Queued Hours"
        time={queuedStats.time}
      />
    </Page>
  );
};

export default ClustersQueuedChart;
