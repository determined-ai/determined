import React, { useCallback, useMemo, useState } from 'react';
import { AlignedData } from 'uplot';

import MetricSelectFilter from 'components/MetricSelectFilter';
import ResponsiveFilters from 'components/ResponsiveFilters';
import ScaleSelectFilter, { Scale } from 'components/ScaleSelectFilter';
import Section from 'components/Section';
import UPlotChart, { Options } from 'components/UPlotChart';
import { tooltipsPlugin } from 'components/UPlotChart/tooltipsPlugin';
import { trackAxis } from 'components/UPlotChart/trackAxis';
import useStorage from 'hooks/useStorage';
import { MetricName, MetricType, RunState, WorkloadWrapper } from 'types';
import { glasbeyColor } from 'utils/color';

import css from '../ExperimentDetails/ExperimentChart.module.scss';

interface Props {
  defaultMetricNames: MetricName[];
  id?: string;
  metricNames: MetricName[];
  storageKey?: string;
  validationMetric?: string;
  workloads?: WorkloadWrapper[];
}

const STORAGE_PATH = 'trial-detail';

const TrialChart: React.FC<Props> = ({
  defaultMetricNames,
  metricNames,
  storageKey,
  validationMetric,
  workloads,
}: Props) => {
  const [ scale, setScale ] = useState<Scale>(Scale.Linear);
  const defaultMetric = useMemo(() => {
    return metricNames.find(metricName => (
      metricName.name === validationMetric && metricName.type === MetricType.Validation
    ));
  }, [ metricNames, validationMetric ]);
  const fallbackMetric = metricNames && metricNames.length !== 0 ? metricNames[0] : undefined;
  const initMetric = defaultMetric || fallbackMetric;
  const storage = useStorage(STORAGE_PATH);

  const [ metrics, setMetrics ] = useState<MetricName[]>(
    storage.getWithDefault(storageKey || '', initMetric ? [ initMetric ] : []),
  );

  const chartData: AlignedData = useMemo(() => {
    const xValues: number[] = [];
    const yValues: Record<string, Record<string, number>> = {};
    metricNames.forEach((metric, index) => yValues[index] = {});

    (workloads || []).forEach(wlWrapper => {
      metricNames.forEach((metric, index) => {
        const metricsWl = metric.type === MetricType.Training ?
          wlWrapper.training : wlWrapper.validation;
        if (!metricsWl || !metricsWl.metrics || metricsWl.state !== RunState.Completed) return;

        const x = metricsWl.numBatches + metricsWl.priorBatchesProcessed;
        if (!xValues.includes(x)) xValues.push(x);

        yValues[index][x] = metricsWl.metrics[metric.name];
      });
    });

    xValues.sort((a,b) => a-b);

    const yValuesArray: (number | null)[][] = Object.values(yValues).map(yValue => {
      return xValues.map(xValue => yValue[xValue] || null);
    });

    return [ xValues, ...yValuesArray ];
  }, [ metricNames, workloads ]);
  const chartOptions: Options = useMemo(() => {
    return {
      axes: [
        { label: 'Batches' },
        { label: 'Metric Value' },
      ],
      height: 400,
      legend: { show: false },
      plugins: [ tooltipsPlugin(), trackAxis() ],
      scales: {
        x: { time: false },
        y: { distr: scale === Scale.Log ? 3 : 1 },
      },
      series: [
        { label: 'Batch' },
        ...metricNames.map((metricName, index) => ({
          label: metricName.name,
          show: (metrics.find(metric => (
            metricName.name === metric.name && metricName.type === metric.type
          ))) != null,
          spanGaps: true,
          stroke: glasbeyColor(index),
          width: 2,
        })),
      ],
    };
  }, [ metricNames, metrics, scale ]);

  const handleMetricChange = useCallback((value: MetricName[]) => {
    setMetrics(value);

    if (storageKey) storage.set(storageKey, value);
  }, [ storage, storageKey ]);

  const options = (
    <ResponsiveFilters>
      <MetricSelectFilter
        defaultMetricNames={defaultMetricNames}
        metricNames={metricNames}
        multiple
        value={metrics}
        onChange={handleMetricChange} />
      <ScaleSelectFilter value={scale} onChange={setScale} />
    </ResponsiveFilters>
  );

  return (
    <Section bodyBorder maxHeight options={options} title='Metrics'>
      <div className={css.base}>
        <UPlotChart data={chartData} options={chartOptions} />
      </div>
    </Section>
  );
};

export default TrialChart;
