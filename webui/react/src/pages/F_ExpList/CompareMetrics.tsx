import _ from 'lodash';
import React, { useMemo, useState } from 'react';

import { ChartGrid, ChartsProps } from 'components/kit/LineChart';
import { Loadable, Loaded, NotLoaded } from 'components/kit/utils/loadable';
import MetricBadgeTag from 'components/MetricBadgeTag';
import { MapOfIdsToColors, useGlasbey } from 'hooks/useGlasbey';
import { TrialMetricData } from 'pages/TrialDetails/useTrialMetrics';
import { ExperimentWithTrial, Serie, TrialItem, XAxisDomain } from 'types';
import handleError from 'utils/error';
import { metricToKey } from 'utils/metric';

interface Props {
  selectedExperiments: ExperimentWithTrial[];
  trials: TrialItem[];
  metricData: TrialMetricData;
}

const CompareMetrics: React.FC<Props> = ({ selectedExperiments, trials, metricData }) => {
  const colorMap = useGlasbey(selectedExperiments.map((e) => e.experiment.id));
  const [xAxis, setXAxis] = useState<XAxisDomain>(XAxisDomain.Batches);
  const { scale, setScale } = metricData;

  const calculateChartProps = (
    metricData: TrialMetricData,
    experiments: ExperimentWithTrial[],
    trials: TrialItem[],
    xAxis: XAxisDomain,
    colorMap: MapOfIdsToColors,
  ): Loadable<ChartsProps> => {
    const { metricHasData, metrics, data, isLoaded, selectedMetrics } = metricData;
    const chartedMetrics: Record<string, boolean> = {};
    const out: ChartsProps = [];
    const expNameById: Record<number, string> = {};
    experiments.forEach((e) => {
      expNameById[e.experiment.id] = e.experiment.name;
    });
    metrics.forEach((metric) => {
      const series: Serie[] = [];
      const key = metricToKey(metric);
      trials.forEach((t) => {
        const m = data[t?.id || 0];
        m?.[key] &&
          t &&
          series.push({
            ...m[key],
            color: colorMap[t.experimentId],
            metricType: '',
            name: expNameById[t.experimentId]
              ? `${expNameById[t.experimentId]} (${t.experimentId})`
              : String(t.experimentId),
          });
        chartedMetrics[key] ||= series.length > 0;
      });
      out.push({
        series: Loaded(series),
        title: <MetricBadgeTag metric={metric} />,
        xAxis,
        xLabel: String(xAxis),
      });
    });

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
      return Loaded(out.map((chartProps) => ({ ...chartProps, series: NotLoaded })));
    } else {
      return Loaded(out);
    }
  };

  const chartsProps = useMemo(
    () => calculateChartProps(metricData, selectedExperiments, trials, xAxis, colorMap),
    [colorMap, trials, xAxis, metricData, selectedExperiments],
  );

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
