import { useMemo, useState } from 'react';

import { ChartGrid, ChartsProps, Serie } from 'components/kit/LineChart';
import { XAxisDomain } from 'components/kit/LineChart/XAxisFilter';
import MetricBadgeTag from 'components/MetricBadgeTag';
import { useTrialMetrics } from 'pages/TrialDetails/useTrialMetrics';
import { ExperimentWithTrial, TrialItem } from 'types';
import handleError from 'utils/error';

import { useGlasbey } from './useGlasbey';

interface Props {
  selectedExperiments: ExperimentWithTrial[];
  trials: TrialItem[];
}

const CompareMetrics: React.FC<Props> = ({ selectedExperiments, trials }) => {
  const colorMap = useGlasbey(selectedExperiments.map((e) => e.experiment.id));
  const [xAxis, setXAxis] = useState<XAxisDomain>(XAxisDomain.Batches);

  const { metrics, data, scale, setScale } = useTrialMetrics(trials);

  const chartsProps = useMemo(() => {
    const out: ChartsProps = [];
    if (!data) return out;
    metrics.forEach((metric) => {
      const series: Serie[] = [];
      const key = `${metric.type}|${metric.name}`;
      trials.forEach((t) => {
        const m = data[t?.id || 0];
        m?.[key] && t && series.push({ ...m[key], color: colorMap[t.experimentId] });
      });
      out.push({
        series,
        title: <MetricBadgeTag metric={metric} />,
        xAxis,
        xLabel: String(xAxis),
      });
    });
    return out;
  }, [metrics, data, colorMap, trials, xAxis]);

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
