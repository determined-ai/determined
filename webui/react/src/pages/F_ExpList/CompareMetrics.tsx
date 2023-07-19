import { useMemo, useState } from 'react';

import { ChartGrid, ChartsProps, Serie } from 'components/kit/LineChart';
import { XAxisDomain } from 'components/kit/LineChart/XAxisFilter';
import MetricBadgeTag from 'components/MetricBadgeTag';
import { TrialMetricData } from 'pages/TrialDetails/useTrialMetrics';
import { ExperimentWithTrial, TrialItem } from 'types';
import handleError from 'utils/error';
import { Loaded, NotLoaded } from 'utils/loadable';

import { useGlasbey } from './useGlasbey';

interface Props {
  selectedExperiments: ExperimentWithTrial[];
  trials: TrialItem[];
  metricData: TrialMetricData;
}

const CompareMetrics: React.FC<Props> = ({ selectedExperiments, trials, metricData }) => {
  const colorMap = useGlasbey(selectedExperiments.map((e) => e.experiment.id));
  const [xAxis, setXAxis] = useState<XAxisDomain>(XAxisDomain.Batches);
  const { scale, setScale } = metricData;

  const chartsProps = useMemo(() => {
    const { metrics, data, metricHasData, isLoaded } = metricData;
    const chartedMetrics: Record<string, boolean> = {};
    const out: ChartsProps = [];
    metrics.forEach((metric) => {
      const series: Serie[] = [];
      const key = `${metric.type}|${metric.name}`;
      trials.forEach((t) => {
        const m = data[t?.id || 0];
        m?.[key] && t && series.push({ ...m[key], color: colorMap[t.experimentId] });
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
      const metricKey = `${metric.type}|${metric.name}`;
      return !!metricHasData?.[metricKey] && !!chartedMetrics?.[metricKey];
    });
    if (isLoaded && chartDataIsLoaded) {
      return Loaded(out);
    } else {
      // returns the chartProps with a NotLoaded series which enables
      // the ChartGrid to show a spinner for the loading charts.
      return Loaded(out.map((chartProps) => ({ ...chartProps, series: NotLoaded })));
    }
  }, [colorMap, trials, xAxis, metricData]);

  return (
    <div style={{ height: 'calc(100vh - 250px)', overflow: 'auto' }}>
      <ChartGrid
        chartsProps={chartsProps}
        handleError={handleError}
        scale={scale}
        setScale={setScale}
        xAxis={xAxis}
        onXAxisChange={setXAxis}
      />
    </div>
  );
};

export default CompareMetrics;
