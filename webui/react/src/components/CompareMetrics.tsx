import { ChartGrid, ChartsProps } from 'hew/LineChart';
import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import _ from 'lodash';
import React, { useCallback, useMemo, useState } from 'react';

import MetricBadgeTag from 'components/MetricBadgeTag';
import { MapOfIdsToColors } from 'hooks/useGlasbey';
import { RunMetricData } from 'hooks/useMetrics';
import { ExperimentWithTrial, FlatRun, Serie, TrialItem, XAxisDomain, XOR } from 'types';
import handleError from 'utils/error';
import { metricToKey } from 'utils/metric';

interface BaseProps {
  metricData: RunMetricData;
  colorMap: MapOfIdsToColors;
}

type Props = XOR<
  { selectedExperiments: ExperimentWithTrial[]; trials: TrialItem[] },
  { selectedRuns: FlatRun[] }
> &
  BaseProps;

const CompareMetrics: React.FC<Props> = ({
  selectedExperiments,
  trials,
  metricData,
  selectedRuns,
  colorMap,
}) => {
  const [xAxis, setXAxis] = useState<XAxisDomain>(XAxisDomain.Batches);
  const { scale, setScale } = metricData;

  const calculateExperimentChartProps = useCallback(
    (
      metricData: RunMetricData,
      experiments: ExperimentWithTrial[],
      trials: TrialItem[],
      xAxis: XAxisDomain,
      colorMap: MapOfIdsToColors,
    ): { chartProps: ChartsProps; chartedMetrics: Record<string, boolean> } => {
      const { metrics, data } = metricData;
      const chartedMetrics: Record<string, boolean> = {};
      const expNameById = experiments.reduce(
        (acc, cur) => {
          acc[cur.experiment.id] = cur.experiment.name;
          return acc;
        },
        {} as Record<number, string>,
      );

      const chartProps: ChartsProps = metrics.map((metric) => {
        const series: Serie[] = [];
        const key = metricToKey(metric);

        trials.forEach((t) => {
          const m = data[t.id];
          m?.[key] &&
            series.push({
              ...m[key],
              color: colorMap[t.experimentId],
              name: expNameById[t.experimentId]
                ? `${expNameById[t.experimentId]} (${t.experimentId})`
                : String(t.experimentId),
            });
          chartedMetrics[key] ||= series.length > 0;
        });

        return {
          series: Loaded(series),
          title: <MetricBadgeTag metric={metric} />,
          xAxis,
          xLabel: String(xAxis),
        };
      });

      return { chartedMetrics, chartProps };
    },
    [],
  );

  const calculateRunsChartProps = useCallback(
    (
      metricData: RunMetricData,
      runs: FlatRun[],
      xAxis: XAxisDomain,
      colorMap: MapOfIdsToColors,
    ): { chartProps: ChartsProps; chartedMetrics: Record<string, boolean> } => {
      const { metrics, data } = metricData;
      const chartedMetrics: Record<string, boolean> = {};
      const chartProps: ChartsProps = metrics.map((metric) => {
        const series: Serie[] = [];
        const key = metricToKey(metric);
        runs.forEach((r) => {
          const m = data[r.id];
          m?.[key] &&
            series.push({
              ...m[key],
              color: colorMap[r.id],
              name: `Run (${r.id})`,
            });
          chartedMetrics[key] ||= series.length > 0;
        });

        return {
          series: Loaded(series),
          title: <MetricBadgeTag metric={metric} />,
          xAxis,
          xLabel: String(xAxis),
        };
      });

      return { chartedMetrics, chartProps };
    },
    [],
  );

  const chartsProps: Loadable<ChartsProps> = useMemo(() => {
    const { metricHasData, metrics, isLoaded, selectedMetrics } = metricData;
    const { chartProps, chartedMetrics } = selectedRuns
      ? calculateRunsChartProps(metricData, selectedRuns, xAxis, colorMap)
      : calculateExperimentChartProps(metricData, selectedExperiments, trials, xAxis, colorMap);

    // In order to show the spinner for each chart in the ChartGrid until
    // metrics are visible, we must determine whether the metrics have been
    // loaded and whether the chart props have been updated.
    // If any metric has data but no chartProps contain data for the metric,
    // then the charts have not been updated and we need to continue to show the
    // spinner.
    const chartDataIsLoaded = metrics.every((metric) => {
      const metricKey = metricToKey(metric);
      return metricHasData?.[metricKey] ? !!chartedMetrics?.[metricKey] : true;
    });
    if (!isLoaded) {
      // When trial metrics hasn't loaded metric names or individual trial metrics.
      return NotLoaded;
    } else if (!chartDataIsLoaded || !_.isEqual(selectedMetrics, metrics)) {
      // In some cases the selectedMetrics returned may not be up to date
      // with the metrics selected by the user. In this case we want to
      // show a loading state until the metrics match.

      // returns the chartProps with a NotLoaded series which enables
      // the ChartGrid to show a spinner for the loading charts.
      return Loaded(chartProps.map((chartProps) => ({ ...chartProps, series: NotLoaded })));
    } else {
      return Loaded(chartProps);
    }
  }, [
    calculateExperimentChartProps,
    calculateRunsChartProps,
    colorMap,
    metricData,
    selectedExperiments,
    selectedRuns,
    trials,
    xAxis,
  ]);

  return (
    <ChartGrid
      chartsProps={chartsProps}
      handleError={handleError}
      scale={scale}
      setScale={setScale}
      xAxis={xAxis}
      onXAxisChange={setXAxis}
    />
  );
};

export default CompareMetrics;
