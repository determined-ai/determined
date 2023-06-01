import { useEffect, useMemo, useState } from 'react';

import { ChartGrid, ChartsProps, Serie } from 'components/kit/LineChart';
import { XAxisDomain } from 'components/kit/LineChart/XAxisFilter';
import { useTrialMetrics } from 'pages/TrialDetails/useTrialMetrics';
import { isEqual } from 'shared/utils/data';
import { ExperimentWithTrial, TrialItem } from 'types';

import { useGlasbey } from './useGlasbey';

interface Props {
  selectedExperiments: ExperimentWithTrial[];
}

const CompareMetrics: React.FC<Props> = ({ selectedExperiments }) => {
  const colorMap = useGlasbey(selectedExperiments.map((e) => e.experiment.id));
  const [xAxis, setXAxis] = useState<XAxisDomain>(XAxisDomain.Batches);
  const [trials, setTrials] = useState<TrialItem[]>([]);

  useEffect(() => {
    const ts: TrialItem[] = [];
    selectedExperiments.forEach((e) => e.bestTrial && ts.push(e.bestTrial));
    setTrials((prev: TrialItem[]) => {
      return isEqual(
        prev?.map((e) => e.id),
        ts?.map((e) => e?.id),
      )
        ? prev
        : ts;
    });
  }, [selectedExperiments]);

  const { metrics, data, scale, setScale } = useTrialMetrics(trials);

  const chartsProps = useMemo(() => {
    const out: ChartsProps = [];
    if (!data) return out;
    metrics.forEach((metric) => {
      const series: Serie[] = [];
      const key = `${metric.type}|${metric.name}`;
      trials.forEach((t) => {
        const m = data[t?.id || 0];
        m?.[key] && series.push({ ...m[key], color: colorMap[t?.experimentId || 0] });
      });
      out.push({
        series,
        title: `${metric.type}.${metric.name}`,
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
        scale={scale}
        setScale={setScale}
        xAxis={xAxis}
        onXAxisChange={setXAxis}
      />
    </div>
  );
};

export default CompareMetrics;
