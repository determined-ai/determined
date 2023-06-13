import React, { useMemo } from 'react';
import { AlignedData } from 'uplot';

import UPlotChart, { Options } from 'components/UPlot/UPlotChart';
import { closestPointPlugin } from 'components/UPlot/UPlotChart/closestPointPlugin';
import { tooltipsPlugin } from 'components/UPlot/UPlotChart/tooltipsPlugin';
import { Metric, Scale } from 'types';
import { glasbeyColor } from 'utils/color';
import { metricToStr } from 'utils/metric';

interface Props {
  data: (number | null)[][];
  focusedTrialId?: number;
  onTrialClick?: (event: MouseEvent, trialId: number) => void;
  onTrialFocus?: (trialId: number | null) => void;
  selectedMetric: Metric;
  selectedScale: Scale;
  selectedTrialIds: number[];
  trialIds: number[];
  xValues: number[];
}

const CHART_HEIGHT = 400;
const SERIES_UNFOCUSED_ALPHA = 0.2;
const SERIES_WIDTH = 3;

const LearningCurveChart: React.FC<Props> = ({
  data,
  focusedTrialId,
  onTrialClick,
  onTrialFocus,
  selectedMetric,
  selectedScale,
  selectedTrialIds,
  trialIds,
  xValues,
}: Props) => {
  const selectedTrialsIdsSet = useMemo(() => new Set(selectedTrialIds), [selectedTrialIds]);

  const chartData: AlignedData = useMemo(() => {
    return [xValues, ...data];
  }, [data, xValues]);

  const chartOptions: Options = useMemo(() => {
    return {
      axes: [
        {
          grid: { width: 1 },
          label: 'Batches Processed',
          scale: 'x',
          side: 2,
        },
        {
          grid: { width: 1 },
          label: metricToStr(selectedMetric),
          scale: 'y',
          side: 3,
        },
      ],
      focus: { alpha: SERIES_UNFOCUSED_ALPHA },
      height: CHART_HEIGHT,
      legend: { show: false },
      plugins: [
        closestPointPlugin({
          onPointClick: (e, point) => {
            if (typeof onTrialClick !== 'function') return;
            onTrialClick(e, trialIds[point.seriesIdx - 1]);
          },
          onPointFocus: (point) => {
            if (typeof onTrialFocus !== 'function') return;
            onTrialFocus(point ? trialIds[point.seriesIdx - 1] : null);
          },
          yScale: 'y',
        }),
        tooltipsPlugin({
          closeOnMouseExit: true,
          isShownEmptyVal: false,
          seriesColors: trialIds.map((t) => glasbeyColor(t)),
        }),
      ],
      scales: { x: { time: false }, y: { distr: selectedScale === Scale.Log ? 3 : 1 } },
      series: [
        { label: 'batches' },
        ...trialIds.map((trialId) => {
          return {
            alpha: focusedTrialId === undefined || trialId === focusedTrialId ? 1 : 0.4,
            label: `trial ${trialId}`,
            scale: 'y',
            show:
              !selectedTrialsIdsSet.size ||
              selectedTrialsIdsSet.has(trialId) ||
              focusedTrialId === trialId,
            spanGaps: true,
            stroke: glasbeyColor(trialId),
            width: SERIES_WIDTH / window.devicePixelRatio,
          };
        }),
      ],
    };
  }, [
    onTrialClick,
    onTrialFocus,
    selectedMetric,
    selectedScale,
    trialIds,
    selectedTrialsIdsSet,
    focusedTrialId,
  ]);

  return <UPlotChart data={chartData} options={chartOptions} />;
};

export default LearningCurveChart;
