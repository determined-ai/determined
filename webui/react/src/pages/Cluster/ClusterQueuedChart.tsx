import RadioGroup from 'hew/RadioGroup';
import React, { useMemo, useState } from 'react';

import Section from 'components/Section';
import ClusterHistoricalUsageChart from 'pages/Cluster/ClusterHistoricalUsageChart';
import { V1RPQueueStat } from 'services/api-ts-sdk';
import { DURATION_DAY, durationInEnglish } from 'utils/datetime';

interface Props {
  poolStats: V1RPQueueStat | undefined;
}

const ClusterQueuedChart: React.FC<Props> = ({ poolStats }: Props) => {
  const [viewDays, setViewDays] = useState(7);

  const queuedStats = useMemo(() => {
    if (!poolStats?.aggregates) return;
    const { aggregates } = poolStats;
    const aggd = aggregates.filter(
      (item) => Date.parse(item.periodStart) >= Date.now() - viewDays * DURATION_DAY,
    );
    return {
      hoursAverage: { average: aggd.map((item) => item.seconds) },
      time: aggd.map((item) => item.periodStart),
    };
  }, [poolStats, viewDays]);

  const dateRange: [number, number] = useMemo(() => {
    const now = Date.now() / 1000;
    return [now - viewDays * 86400, now];
  }, [viewDays]);

  if (!queuedStats) return <div />;
  return (
    <>
      <Section
        bodyBorder
        options={
          <RadioGroup
            options={[
              { id: 7, label: '7 days' },
              { id: 30, label: '30 days' },
            ]}
            value={viewDays}
            onChange={setViewDays}
          />
        }
        title="Avg Queue Time">
        <ClusterHistoricalUsageChart
          chartKey={viewDays}
          dateRange={dateRange}
          formatValues={(_: uPlot, splits: number[]) =>
            splits.map((n) => durationInEnglish(n * 1000))
          }
          hoursByLabel={queuedStats.hoursAverage}
          label=" "
          time={queuedStats.time}
        />
      </Section>
    </>
  );
};

export default ClusterQueuedChart;
