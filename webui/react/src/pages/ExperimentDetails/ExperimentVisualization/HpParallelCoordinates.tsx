import React from 'react';

import ParallelCoordinates from 'components/ParallelCoordinates';
import Section from 'components/Section';
import { ExperimentBase, MetricName } from 'types';

import css from './HpParallelCoordinates.module.scss';

interface Props {
  batches: number[];
  experiment: ExperimentBase;
  metrics: MetricName[];
  onBatchChange?: (batch: number) => void;
  onMetricChange?: (metric: MetricName) => void;
  selectedBatch?: number;
  selectedMetric: MetricName;
}

const HpParallelCoordinates: React.FC<Props> = ({
  batches,
  experiment,
  metrics,
  onBatchChange,
  onMetricChange,
  selectedBatch,
  selectedMetric,
}: Props) => {
  return (
    <div className={css.base}>
      <Section title="HP Parallel Coordinates">
        <div className={css.container}>
          <ParallelCoordinates />
        </div>
      </Section>
    </div>
  );
};

export default HpParallelCoordinates;
