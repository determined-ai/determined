import { Empty } from 'antd';
import React, { useMemo, useState } from 'react';
import { AlignedData } from 'uplot';

import MetricSelectFilter from 'components/MetricSelectFilter';
import ResponsiveFilters from 'components/ResponsiveFilters';
import ScaleSelectFilter, { Scale } from 'components/ScaleSelectFilter';
import Section from 'components/Section';
import UPlotChart, { Options } from 'components/UPlot/UPlotChart';
import { tooltipsPlugin } from 'components/UPlot/UPlotChart/tooltipsPlugin';
import { trackAxis } from 'components/UPlot/UPlotChart/trackAxis';
import css from 'pages/TrialDetails/TrialChart.module.scss';
import { glasbeyColor } from 'shared/utils/color';
import { MetricName, MetricType, WorkloadGroup } from 'types';

interface Props {
  defaultMetricNames: MetricName[];
  id?: string;
  metricNames: MetricName[];
  metrics: MetricName[];
  onMetricChange: (value: MetricName[]) => void;
  trialId?: number;
  workloads?: WorkloadGroup[];
}

const getChartMetricLabel = (metric: MetricName): string => {
  if (metric.type === 'training') return `[T] ${metric.name}`;
  if (metric.type === 'validation') return `[V] ${metric.name}`;
  return metric.name;
};

const TrialChart: React.FC<Props> = ({
  defaultMetricNames,
  metricNames,
  metrics,
  onMetricChange,
  workloads,
  trialId,
}: Props) => {
  const [ scale, setScale ] = useState<Scale>(Scale.Linear);

  const chartData: AlignedData = useMemo(() => {
    const xValues: number[] = [];
    const yValues: Record<string, Record<string, number | null>> = {};
    metrics.forEach((metric, index) => yValues[index] = {});

    (workloads || []).forEach(wlWrapper => {
      metrics.forEach((metric, index) => {
        const metricsWl = metric.type === MetricType.Training ?
          wlWrapper.training : wlWrapper.validation;
        if (!metricsWl || !metricsWl.metrics) return;

        const x = metricsWl.totalBatches;
        if (!xValues.includes(x)) xValues.push(x);

        /**
         * TODO: filtering NaN, +/- Infinity for now, but handle it later with
         * dynamic min/max ranges via uPlot.Scales.
         */
        const y = metricsWl.metrics[metric.name];
        yValues[index][x] = Number.isFinite(y) ? y : null;
      });
    });

    xValues.sort((a, b) => a - b);

    const yValuesArray: (number | null)[][] = Object.values(yValues).map(yValue => {
      return xValues.map(xValue => yValue[xValue] != null ? yValue[xValue] : null);
    });

    return [ xValues, ...yValuesArray ];
  }, [ metrics, workloads ]);

  const chartOptions: Options = useMemo(() => {
    return {
      axes: [
        { label: 'Batches' },
        { label: metrics.length === 1 ? getChartMetricLabel(metrics[0]) : 'Metric Value' },
      ],
      height: 400,
      key: trialId,
      legend: { show: false },
      plugins: [ tooltipsPlugin(), trackAxis() ],
      scales: { x: { time: false }, y: { distr: scale === Scale.Log ? 3 : 1 } },
      series: [
        { label: 'Batch' },
        ...metrics.map((metric, index) => ({
          label: getChartMetricLabel(metric),
          spanGaps: true,
          stroke: glasbeyColor(index),
          width: 2,
        })),
      ],
    };
  }, [ metrics, scale, trialId ]);

  const options = (
    <ResponsiveFilters>
      <MetricSelectFilter
        defaultMetricNames={defaultMetricNames}
        metricNames={metricNames}
        multiple
        value={metrics}
        onChange={onMetricChange}
      />
      <ScaleSelectFilter value={scale} onChange={setScale} />
    </ResponsiveFilters>
  );

  return (
    <Section bodyBorder options={options} title="Metrics">
      <div className={css.base}>
        {chartData[0].length === 0 ?
          <Empty description="No data to plot." image={Empty.PRESENTED_IMAGE_SIMPLE} /> :
          <UPlotChart data={chartData} options={chartOptions} />}
      </div>
    </Section>
  );
};

export default TrialChart;
