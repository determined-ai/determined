import { Radio } from 'antd';
import React, { useMemo, useState } from 'react';

import Section from 'components/Section';
import { V1RPQueueStat } from 'services/api-ts-sdk';
import { DURATION_DAY } from 'utils/datetime';

import ClusterHistoricalUsageChart from '../Cluster/ClusterHistoricalUsageChart';

import css from './ClustersQueuedChart.module.scss';

interface Props {
  poolStats: V1RPQueueStat | undefined;
}

const ClustersQueuedChart: React.FC<Props> = ({ poolStats }: Props) => {
  const [viewDays, setViewDays] = useState(7);

  const queuedStats = useMemo(() => {
    if (!poolStats?.aggregates) return;
    const { aggregates } = poolStats;
    const aggd = aggregates.filter(
      (item) => Date.parse(item.periodStart) >= Date.now() - viewDays * DURATION_DAY,
    );
    return {
      hoursAverage: { average: aggd.map((item) => item.seconds / 60) },
      time: aggd.map((item) => item.periodStart),
    };
  }, [poolStats, viewDays]);

  if (!queuedStats) return <div />;
  return (
    <>
      <Section
        bodyBorder
        options={
          <Radio.Group
            className={css.filter}
            value={viewDays}
            onChange={(e) => setViewDays(e.target.value)}>
            <Radio.Button value={7}>7 days</Radio.Button>
            <Radio.Button value={30}>30 days</Radio.Button>
          </Radio.Group>
        }
        title="Avg Queue Time">
        <ClusterHistoricalUsageChart
          chartKey={viewDays}
          hoursByLabel={queuedStats.hoursAverage}
          label="Queued Minutes"
          time={queuedStats.time}
        />
      </Section>
    </>
  );
};

export default ClustersQueuedChart;
