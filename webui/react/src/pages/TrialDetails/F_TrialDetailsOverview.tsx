import React, { useMemo } from 'react';

import { ChartGrid, ChartProps, Serie, TRAIN_PREFIX, VAL_PREFIX } from 'components/kit/LineChart';
import usePermissions from 'hooks/usePermissions';
import TrialInfoBox from 'pages/TrialDetails/TrialInfoBox';
import { ExperimentBase, Metric, MetricType, TrialDetails } from 'types';
import { metricSorter, metricToKey } from 'utils/metric';

import { useTrialMetrics } from './useTrialMetrics';

export interface Props {
  experiment: ExperimentBase;
  trial?: TrialDetails;
}

const isMetricNameMatch = (t: Metric, v: Metric) => {
  const t_stripped = t.name.replace(TRAIN_PREFIX, '');
  const v_stripped = v.name.replace(VAL_PREFIX, '');
  return t_stripped === v_stripped;
};

const TrialDetailsOverview: React.FC<Props> = ({ experiment, trial }: Props) => {
  const showExperimentArtifacts = usePermissions().canViewExperimentArtifacts({
    workspace: { id: experiment.workspaceId },
  });

  const { metrics, data } = useTrialMetrics(trial);

  const pairedMetrics: ([Metric] | [Metric, Metric])[] | undefined = useMemo(() => {
    const val = metrics.filter((m) => m.type === MetricType.Validation).sort(metricSorter);
    const train = metrics.filter((m) => m.type === MetricType.Training).sort(metricSorter);
    let out: ([Metric] | [Metric, Metric])[] = [];
    while (val.length) {
      const v = val.shift();
      if (!v) return;
      let pair: [Metric] | [Metric, Metric] = [v];
      const t_match = train.findIndex((t) => isMetricNameMatch(t, v));
      if (t_match !== -1) pair = pair.concat(train.splice(t_match, 1)) as [Metric, Metric];
      out.push(pair);
    }
    out = out.concat(train.map((t) => [t]));
    return out;
  }, [metrics]);

  const chartsProps = useMemo(() => {
    const out: ChartProps = [];

    pairedMetrics?.forEach(([trainingMetric, valMetric]) => {
      // this code doesnt depend on their being training or validation metrics
      // naming just makes it easier to read
      const trainingMetricKey = metricToKey(trainingMetric);
      const trainingMetricSeries = data?.[trainingMetricKey];
      if (!trainingMetricSeries) return;

      const series: Serie[] = [trainingMetricSeries];

      if (valMetric) {
        const valMetricKey = metricToKey(valMetric);
        const valMetricData = data?.[valMetricKey];
        if (valMetricData) series.push(valMetricData);
      }

      out.push({
        series,
        xLabel: 'Batch',
      });
    });
    return out;
  }, [pairedMetrics, data]);

  return (
    <>
      <TrialInfoBox experiment={experiment} trial={trial} />
      {showExperimentArtifacts ? <ChartGrid chartsProps={chartsProps} /> : null}
    </>
  );
};

export default TrialDetailsOverview;
