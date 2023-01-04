import { Empty } from 'antd';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { AlignedData } from 'uplot';

import ResponsiveFilters from 'components/ResponsiveFilters';
import ScaleSelectFilter from 'components/ScaleSelectFilter';
import Section from 'components/Section';
import UPlotChart, { Options } from 'components/UPlot/UPlotChart';
import { tooltipsPlugin } from 'components/UPlot/UPlotChart/tooltipsPlugin';
import { trackAxis } from 'components/UPlot/UPlotChart/trackAxis';
import css from 'pages/TrialDetails/TrialCharts.module.scss';
import { compareTrials } from 'services/api';
import Spinner from 'shared/components/Spinner';
import usePolling from 'shared/hooks/usePolling';
import { glasbeyColor } from 'shared/utils/color';
import { Metric, MetricContainer, Scale } from 'types';

interface Props {
  id?: string;
  metricNames: Metric[];
  trialId?: number;
  trialTerminated: boolean;
}

const getChartMetricLabel = (metric: Metric): string => {
  if (metric.type === 'training') return `[T] ${metric.name}`;
  if (metric.type === 'validation') return `[V] ${metric.name}`;
  return metric.name;
};

const TrialCharts: React.FC<Props> = ({ metricNames, trialId, trialTerminated }: Props) => {
  const [scale, setScale] = useState<Scale>(Scale.Linear);
  const [trialSumm, setTrialSummary] = useState<MetricContainer[]>([]);

  const fetchTrialSummary = useCallback(async () => {
    if (trialId) {
      const summ = await compareTrials({
        maxDatapoints: screen.width > 1600 ? 1500 : 1000,
        metricNames: metricNames,
        scale: scale,
        startBatches: 0,
        trialIds: [trialId],
      });
      setTrialSummary(summ[0].metrics);
    }
  }, [metricNames, scale, trialId]);

  const { stopPolling } = usePolling(fetchTrialSummary, { interval: 2000, rerunOnNewFn: true });

  useEffect(() => {
    if (trialTerminated) {
      stopPolling();
    }
  }, [trialTerminated, stopPolling]);

  if (trialTerminated) {
    stopPolling();
  }

  const chartData: AlignedData[] = useMemo(
    () =>
      metricNames.map((metric) => {
        const xValues: number[] = [];
        const yValues: Record<string, Record<string, number | null>> = {};
        yValues[0] = {};

        const mWrapper = trialSumm.find(
          (mContainer) => mContainer.name === metric.name && mContainer.type === metric.type,
        );
        if (mWrapper?.data) {
          mWrapper.data.forEach((pt) => {
            if (!xValues.includes(pt.batches)) {
              xValues.push(pt.batches);
            }
            yValues[0][pt.batches] = Number.isFinite(pt.value) ? pt.value : null;
          });
        }

        xValues.sort((a, b) => a - b);

        const yValuesArray: (number | null)[][] = Object.values(yValues).map((yValue) => {
          return xValues.map((xValue) => (yValue[xValue] != null ? yValue[xValue] : null));
        });

        return [xValues, ...yValuesArray];
      }),
    [metricNames, trialSumm],
  );

  const chartOptions: Options[] = useMemo(
    () =>
      metricNames.map((metric, index) => {
        const color = glasbeyColor(index);
        return {
          axes: [{ label: 'Batches' }, { label: 'Metric Value' }],
          height: 400,
          key: `${trialId}_${metric.name}`,
          legend: { show: false },
          plugins: [tooltipsPlugin({ color }), trackAxis()],
          scales: { x: { time: false }, y: { distr: scale === Scale.Log ? 3 : 1 } },
          series: [
            { label: 'Batch' },
            {
              label: getChartMetricLabel(metric),
              spanGaps: true,
              stroke: color,
              width: 2,
            },
          ],
        };
      }),
    [metricNames, scale, trialId],
  );

  const options = (
    <ResponsiveFilters>
      <ScaleSelectFilter value={scale} onChange={setScale} />
    </ResponsiveFilters>
  );

  return (
    <>
      <Section options={options} title="Metrics" />
      <Spinner className={css.spinner} conditionalRender spinning={!trialId}>
        {chartData.length === 0 || chartData[0].length === 0 || chartData[0][0].length === 0 ? (
          <Empty description="No data to plot." image={Empty.PRESENTED_IMAGE_SIMPLE} />
        ) : (
          chartOptions.map((co, idx) => (
            <Section bodyBorder key={idx}>
              <div className={css.base}>
                <UPlotChart data={chartData[idx]} options={co} title={co.series[1].label} />
              </div>
            </Section>
          ))
        )}
      </Spinner>
    </>
  );
};

export default TrialCharts;
