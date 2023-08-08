import React, { useMemo, useState } from 'react';

import { calculateChartProps, ChartGrid } from 'components/kit/LineChart';
import { XAxisDomain } from 'components/kit/LineChart/XAxisFilter';
import { useGlasbey } from 'hooks/useGlasbey';
import { TrialMetricData } from 'pages/TrialDetails/useTrialMetrics';
import { ExperimentWithTrial, TrialItem } from 'types';
import handleError from 'utils/error';

interface Props {
  selectedExperiments: ExperimentWithTrial[];
  trials: TrialItem[];
  metricData: TrialMetricData;
}

const CompareMetrics: React.FC<Props> = ({ selectedExperiments, trials, metricData }) => {
  const colorMap = useGlasbey(selectedExperiments.map((e) => e.experiment.id));
  const [xAxis, setXAxis] = useState<XAxisDomain>(XAxisDomain.Batches);
  const { scale, setScale } = metricData;

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
