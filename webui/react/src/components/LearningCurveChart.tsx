import React, { useEffect, useMemo, useRef } from 'react';
import uPlot, { AlignedData } from 'uplot';

import UPlotChart, { Options } from 'components/UPlotChart';
import { closestPointPlugin } from 'components/UPlotChart/closestPointPlugin';
import { MetricName } from 'types';
import { glasbeyColor } from 'utils/color';
import { metricNameToStr } from 'utils/string';

interface Props {
  data: (number | null)[][];
  focusedTrialId?: number;
  onTrialClick?: (event: React.MouseEvent, trialId: number) => void;
  onTrialFocus?: (trialId: number | null) => void;
  selectedMetric: MetricName;
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
  trialIds,
  xValues,
}: Props) => {
  const chart = useRef<uPlot>();

  // const handleClick = useCallback((event: React.MouseEvent) => {
  //   if (!chart || !mouseDownPoint) return;
  //
  //   /*
  //    * Make sure the mouse down and mouse up distance is fairly close
  //    * to be considered a click instead of a drag movement for chart zoom.
  //    */
  //   const dist = distance(event.clientX, event.clientY, mouseDownPoint?.x, mouseDownPoint?.y);
  //   if (dist < MOUSE_CLICK_THRESHOLD) {
  //     if (focusedPoint && focusedPoint.seriesIdx != null && onTrialClick) {
  //       onTrialClick(event, trialIds[focusedPoint.seriesIdx]);
  //     }
  //     setShowZoomOutTip(false);
  //   } else {
  //     setShowZoomOutTip(true);
  //   }
  //
  //   setMouseDownPoint(undefined);
  // }, [ chart, focusedPoint, mouseDownPoint, onTrialClick, trialIds ]);
  //
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
          label: metricNameToStr(selectedMetric),
          scale: 'metric',
          side: 3,
        },
      ],
      focus: { alpha: SERIES_UNFOCUSED_ALPHA },
      height: CHART_HEIGHT,
      legend: { show: false },
      plugins: [ closestPointPlugin({
        onPointFocus: (point) => {
          if (typeof onTrialFocus !== 'function') return;
          onTrialFocus(point ? trialIds[point.seriesIdx - 1] : null);
        },
        yScale: 'metric',
      }) ],
      scales: { x: { time: false } },
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
  }, [ onTrialFocus, selectedMetric, trialIds ]);

  /*
   * Focus on a trial series if provided.
   */
  useEffect(() => {
    if (!chart.current) return;

    let seriesIdx = -1;
    if (focusedTrialId && trialIds.includes(focusedTrialId)) {
      seriesIdx = trialIds.findIndex(id => id === focusedTrialId);
    }

    if (seriesIdx === -1) {
      chart.current.setSeries(null as unknown as number, { focus: false });
    } else {
      chart.current.setSeries(seriesIdx + 1, { focus: true });
    }
  }, [ focusedTrialId, trialIds ]);

  return <UPlotChart data={chartData} options={chartOptions} ref={chart} />;
};

export default LearningCurveChart;
