import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { AlignedData } from 'uplot';

import MetricSelect from 'components/MetricSelect';
import ResponsiveFilters from 'components/ResponsiveFilters';
import ScaleSelect from 'components/ScaleSelect';
import Section from 'components/Section';
import UPlotChart, { Options } from 'components/UPlot/UPlotChart';
import { tooltipsPlugin } from 'components/UPlot/UPlotChart/tooltipsPlugin';
import { trackAxis } from 'components/UPlot/UPlotChart/trackAxis';
import usePolling from 'hooks/usePolling';
import css from 'pages/TrialDetails/TrialChart.module.scss';
import { timeSeries } from 'services/api';
import { Metric, MetricContainer, Scale } from 'types';
import { glasbeyColor } from 'utils/color';
import handleError, { ErrorType } from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';
import { metricToStr } from 'utils/metric';

interface Props {
  defaultMetricNames: Metric[];
  id?: string;
  metricNames: Metric[];
  metrics: Metric[];
  onMetricChange: (value: Metric[]) => void;
  trialId?: number;
  trialTerminated: boolean;
}

const TrialChart: React.FC<Props> = ({
  defaultMetricNames,
  metricNames,
  metrics,
  onMetricChange,
  trialId,
  trialTerminated,
}: Props) => {
  const [scale, setScale] = useState<Scale>(Scale.Linear);
  const [trialSummary, setTrialSummary] = useState<Loadable<MetricContainer[]>>(NotLoaded);

  const fetchTrialSummary = useCallback(async () => {
    if (trialId) {
      try {
        const summary = await timeSeries({
          maxDatapoints: screen.width > 1600 ? 1500 : 1000,
          metrics: metricNames,
          startBatches: 0,
          trialIds: [trialId],
        });
        setTrialSummary(Loaded(summary[0].metrics));
      } catch (e) {
        handleError(e, {
          publicMessage: `Failed to load trial summary for trial ${trialId}.`,
          publicSubject: 'Trial summary fail to load.',
          type: ErrorType.Api,
        });
        setTrialSummary(Loaded([]));
      }
    }
  }, [metricNames, trialId]);

  const { stopPolling } = usePolling(fetchTrialSummary, { interval: 2000, rerunOnNewFn: true });

  useEffect(() => {
    if (trialTerminated) {
      stopPolling();
    }
  }, [trialTerminated, stopPolling]);

  if (trialTerminated) {
    stopPolling();
  }

  const chartData: AlignedData = useMemo(() => {
    const xValues: number[] = [];
    const yValues: Record<string, Record<string, number | null>> = {};

    metrics.forEach((metric, index) => {
      yValues[index] = {};

      const summary = Loadable.getOrElse([], trialSummary);
      const mWrapper = summary.find((mContainer) => mContainer.group === metric.group);
      if (!mWrapper?.data) return;

      mWrapper.data.forEach((avgMetrics) => {
        if (avgMetrics.values[metric.name] || avgMetrics.values[metric.name] === 0) {
          if (!xValues.includes(avgMetrics.batches)) {
            xValues.push(avgMetrics.batches);
          }
          yValues[index][avgMetrics.batches] = Number.isFinite(avgMetrics.values[metric.name])
            ? avgMetrics.values[metric.name]
            : null;
        }
      });
    });

    xValues.sort((a, b) => a - b);

    const yValuesArray: (number | null)[][] = Object.values(yValues).map((yValue) => {
      return xValues.map((xValue) => (yValue[xValue] != null ? yValue[xValue] : null));
    });

    return [xValues, ...yValuesArray];
  }, [metrics, trialSummary]);

  const chartOptions: Options = useMemo(() => {
    return {
      axes: [
        { label: 'Batches' },
        { label: metrics.length === 1 ? metricToStr(metrics[0]) : 'Metric Value' },
      ],
      height: 400,
      key: trialId,
      legend: { show: false },
      plugins: [
        tooltipsPlugin({ closeOnMouseExit: true, isShownEmptyVal: true, seriesColors: [] }),
        trackAxis(),
      ],
      scales: { x: { time: false }, y: { distr: scale === Scale.Log ? 3 : 1 } },
      series: [
        { label: 'Batch' },
        ...metrics.map((metric, index) => ({
          label: metricToStr(metric),
          spanGaps: true,
          stroke: glasbeyColor(index),
          width: 2,
        })),
      ],
    };
  }, [metrics, scale, trialId]);

  const options = (
    <ResponsiveFilters>
      <MetricSelect
        defaultMetrics={defaultMetricNames}
        metrics={metricNames}
        multiple
        value={metrics}
        onChange={onMetricChange}
      />
      <ScaleSelect value={scale} onChange={setScale} />
    </ResponsiveFilters>
  );

  return (
    <Section bodyBorder options={options} title="Metrics">
      <div className={css.base}>
        <UPlotChart
          data={chartData}
          isLoading={!trialId || Loadable.isLoading(trialSummary)}
          options={chartOptions}
        />
      </div>
    </Section>
  );
};

export default TrialChart;
