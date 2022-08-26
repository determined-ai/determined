import React, { useEffect, useMemo, useState } from 'react';
import { AlignedData } from 'uplot';

import UPlotChart, { Options } from 'components/UPlot/UPlotChart';
import { closestPointPlugin } from 'components/UPlot/UPlotChart/closestPointPlugin';
import { glasbeyColor } from 'shared/utils/color';
import { Metric, Scale } from 'types';
import { metricToStr } from 'utils/metric';

interface Props {
  colorMap?: Record<number, string>;
  data: (number | null)[][];
  focusedTrialId?: number;
  onTrialClick?: (event: MouseEvent, trialId: number) => void;
  onTrialFocus?: (trialId: number | null) => void;
  selectedMetric: Metric;
  selectedScale: Scale
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
  const [ focusIndex, setFocusIndex ] = useState<number>();

  const selectedTrialsIdsSet = useMemo(() => new Set(selectedTrialIds), [ selectedTrialIds ]);

  const chartData: AlignedData = useMemo(() => {
    return [ xValues, ...data ];
  }, [ data, xValues ]);

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
      plugins: [ closestPointPlugin({
        getPointTooltipHTML: (x, y, point) => {
          const trialId = trialIds[point.seriesIdx - 1];
          return `Trial ID: ${trialId}<br />Batches: ${x}<br />Metric: ${y}`;
        },
        onPointClick: (e, point) => {
          if (typeof onTrialClick !== 'function') return;
          onTrialClick(e, trialIds[point.seriesIdx - 1]);
        },
        onPointFocus: (point) => {
          if (typeof onTrialFocus !== 'function') return;
          onTrialFocus(point ? trialIds[point.seriesIdx - 1] : null);
        },
        yScale: 'y',
      }) ],
      scales: { x: { time: false }, y: { distr: selectedScale === Scale.Log ? 3 : 1 } },
      series: [
        { label: 'batches' },
        ...trialIds.map((trialId) => {
          return {
            label: `trial ${trialId}`,
            scale: 'y',
            // show: true,
            show: !selectedTrialsIdsSet.size || selectedTrialsIdsSet.has(trialId),
            spanGaps: true,
            stroke: glasbeyColor(trialId),
            width: SERIES_WIDTH / window.devicePixelRatio,
          };
        }),
      ],
    };
  }, [ onTrialClick, onTrialFocus, selectedMetric, selectedScale, trialIds, selectedTrialsIdsSet ]);

  /*
   * Focus on a trial series if provided.
   */
  useEffect(() => {
    let seriesIdx = -1;
    if (focusedTrialId && trialIds.includes(focusedTrialId)) {
      seriesIdx = trialIds.findIndex((id) => id === focusedTrialId);
    }
    setFocusIndex(seriesIdx !== -1 ? seriesIdx : undefined);
  }, [ focusedTrialId, trialIds ]);

  return <UPlotChart data={chartData} focusIndex={focusIndex} options={chartOptions} />;
};

export default LearningCurveChart;
