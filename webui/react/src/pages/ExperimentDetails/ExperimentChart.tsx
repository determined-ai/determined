import React, { useEffect, useRef, useState } from 'react';
import { AlignedData } from 'uplot';

import ScaleSelectFilter, { Scale } from 'components/ScaleSelectFilter';
import Section from 'components/Section';
import UPlotChart, { Options } from 'components/UPlotChart';
import { ChartTooltip, tooltipsPlugin } from 'components/UPlotChart/tooltipsPlugin';
import { trackAxis } from 'components/UPlotChart/trackAxis';
import { ValidationHistory } from 'types';
import { glasbeyColor } from 'utils/color';

import css from './ExperimentChart.module.scss';

interface Props {
  startTime?: string;
  validationHistory?: ValidationHistory[];
  validationMetric?: string;
}

const ExperimentChart: React.FC<Props> = ({
  startTime,
  validationHistory,
  validationMetric,
}: Props) => {
  const [ chartData, setChartData ] = useState<AlignedData>();
  const [ chartOptions, setChartOptions ] = useState<Options>();
  const [ scale, setScale ] = useState<Scale>(Scale.Linear);
  const chartTooltipData = useRef<ChartTooltip[][]>();

  useEffect(() => {
    const getXTooltipHeader = (xIndex: number): ChartTooltip => {
      if (!chartTooltipData.current) return null;
      return chartTooltipData.current[0][xIndex] || null;
    };

    setChartOptions({
      axes: [
        { label: 'Time Elapsed (sec)' },
        { label: 'Metric Value' },
      ],
      height: 400,
      legend: { show: false },
      plugins: [
        tooltipsPlugin({ getXTooltipHeader }),
        trackAxis(),
      ],
      scales: {
        x: { time: false },
        y: { distr: scale === Scale.Log ? 3 : 1 },
      },
      series: [
        { label: 'Time Elapsed (sec)' },
        {
          label: validationMetric,
          stroke: glasbeyColor(0),
          width: 2,
        },
      ],
    });
  }, [ scale, validationMetric ]);

  useEffect(() => {
    if (!startTime || !validationHistory) return;

    const startTimestamp = new Date(startTime).getTime();
    const x: number[] = [];
    const y: number[] = [];
    const yTooltip: string[] = [];

    validationHistory.forEach(validation => {
      if (!validation.validationError) return;

      const endTimestamp = new Date(validation.endTime).getTime();

      x.push((endTimestamp - startTimestamp) / 1000);
      y.push(validation.validationError);
      yTooltip.push('Trial ' + validation.trialId);
    });

    chartTooltipData.current = [ yTooltip ];
    setChartData([ x, y ]);
  }, [ chartTooltipData, startTime, validationHistory ]);

  const options = <ScaleSelectFilter value={scale} onChange={setScale} />;
  const title = 'Best Validation Metric'
    + (validationMetric ? ` (${validationMetric})` : '');

  return (
    <Section bodyBorder maxHeight options={options} title={title}>
      <div className={css.base}>
        <UPlotChart data={chartData} options={chartOptions} />
      </div>
    </Section>
  );
};

export default ExperimentChart;
