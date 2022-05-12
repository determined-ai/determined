import React, { useEffect, useMemo, useState } from 'react';
import { AlignedData } from 'uplot';

import UPlotChart, { Options } from 'components/UPlot/UPlotChart';
import { closestPointPlugin } from 'components/UPlot/UPlotChart/closestPointPlugin';
import { glasbeyColor } from 'shared/utils/color';
import { MetricName } from 'types';
import { metricNameToStr } from 'utils/metric';

interface Props {
  data: (number | null)[][];
  focusedTrialId?: number;
  onTrialClick?: (event: MouseEvent, trialId: number) => void;
  onTrialFocus?: (trialId: number | null) => void;
  selectedMetric: MetricName;
  trialIds: number[];
  xValues: number[];
}

const A_REASONABLY_SMALL_NUMBER = 0.0001;
const CHART_HEIGHT = 400;
const SERIES_UNFOCUSED_ALPHA = 0.2;
const SERIES_WIDTH = 3;

const LearningCurveChart: React.FC<Props> = ({
  data,
  focusedTrialId,
  onTrialClick,
  onTrialFocus,
  selectedMetric,
  trialIds,
  xValues,
}: Props) => {
  const [ focusIndex, setFocusIndex ] = useState<number>();

  const chartData: AlignedData = useMemo(() => {
    return [ xValues, ...data ];
  }, [ data, xValues ]);

  const chartOptions: Options = useMemo(() => {

    const onlyOneXValue = chartData?.[0]?.length === 1;
    const scales = onlyOneXValue
      ? {
        x: {
          max: (chartData[0][0] ?? 0) + A_REASONABLY_SMALL_NUMBER,
          min: (chartData[0][0] ?? 0) - A_REASONABLY_SMALL_NUMBER,
          time: false,
        },
        y: {
          max:
              Math.max(
                ...chartData
                  .slice(1)
                  .filter((x) => x != null)
                  .map((x) => x[0] ?? Number.MIN_SAFE_INTEGER),
              ) + A_REASONABLY_SMALL_NUMBER ?? Number.MAX_SAFE_INTEGER,
          min:
              Math.min(
                ...chartData
                  .slice(1)
                  .filter((x) => x != null)
                  .map((x) => x[0] ?? Number.MAX_SAFE_INTEGER),
              ) + A_REASONABLY_SMALL_NUMBER ?? Number.MIN_SAFE_INTEGER,
        },
      }
      : { x: { time: false }, y: {} };

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
          label: metricNameToStr(selectedMetric),
          scale: 'metric',
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
        yScale: 'metric',
      }) ],
      scales,
      series: [
        { label: 'batches' },
        ...trialIds.map((trialId, index) => ({
          label: `trial ${trialId}`,
          scale: 'metric',
          spanGaps: true,
          stroke: glasbeyColor(index),
          width: SERIES_WIDTH / window.devicePixelRatio,
        })),
      ],
    };
  }, [ onTrialClick, onTrialFocus, selectedMetric, trialIds, chartData ]);

  /*
   * Focus on a trial series if provided.
   */
  useEffect(() => {
    let seriesIdx = -1;
    if (focusedTrialId && trialIds.includes(focusedTrialId)) {
      seriesIdx = trialIds.findIndex(id => id === focusedTrialId);
    }
    setFocusIndex(seriesIdx !== -1 ? seriesIdx : undefined);
  }, [ focusedTrialId, trialIds ]);

  return <UPlotChart data={chartData} focusIndex={focusIndex} options={chartOptions} />;
};

export default LearningCurveChart;
