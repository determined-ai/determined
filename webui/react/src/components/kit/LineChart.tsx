import React, { MouseEvent, ReactElement, useMemo } from 'react';
import { AlignedData } from 'uplot';

import { SyncProvider } from 'components/UPlot/SyncableBounds';
import UPlotChart, { Options } from 'components/UPlot/UPlotChart';
import { closestPointPlugin } from 'components/UPlot/UPlotChart/closestPointPlugin';
import { tooltipsPlugin } from 'components/UPlot/UPlotChart/tooltipsPlugin';
import { glasbeyColor } from 'shared/utils/color';
import { Scale } from 'types';

import css from './LineChart.module.scss';

interface Props {
  closestPoint?: boolean;
  colors?: string[];
  data: number[][][];
  focusedSeries?: number;
  height?: number;
  onSeriesHover?: (seriesIdx: number | null) => void;
  onSeriesSelect?: (event: MouseEvent, seriesIdx: number) => void;
  scale?: Scale;
  seriesNames?: string;
  showLegend?: boolean;
  showTooltip?: boolean;
  title?: string;
  width?: number;
  xLabel?: string;
  xMax?: number;
  xMin?: number;
  yLabel?: string;
}

export const LineChart: React.FC<Props> = ({
  closestPoint = false,
  colors,
  data,
  focusedSeries,
  height = 400,
  scale = Scale.Linear,
  seriesNames,
  showLegend = false,
  showTooltip = false,
  title,
  xLabel,
  xMin,
  xMax,
  yLabel,
}: Props) => {
  const chartData: AlignedData = useMemo(() => {
    const xValues: number[] = [];
    const yValues: Record<string, Record<string, number | null>> = {};

    data.forEach((line, lineIndex) => {
      yValues[lineIndex] = {};
      line.forEach((pt) => {
        xValues.push(pt[0]);
        yValues[lineIndex][pt[0]] = Number.isFinite(pt[1]) ? pt[1] : null;
      });
    });

    xValues.sort((a, b) => a - b);
    const yValuesArray: (number | null)[][] = Object.values(yValues).map((yValue) => {
      return xValues.map((xValue) => (yValue[xValue] != null ? yValue[xValue] : null));
    });

    return [xValues, ...yValuesArray];
  }, [data]);

  const chartOptions: Options = useMemo(() => {
    const plugins = [];
    if (closestPoint) {
      plugins.push(
        closestPointPlugin({
          // onPointClick: (e, point) => {
          // if (typeof onTrialClick !== 'function') return;
          // onTrialClick(e, trialIds[point.seriesIdx - 1]);
          // },
          // onPointFocus: (point) => {
          // if (typeof onTrialFocus !== 'function') return;
          // onTrialFocus(point ? trialIds[point.seriesIdx - 1] : null);
          // },
          yScale: 'y',
        }),
      );
    }
    if (showTooltip) {
      plugins.push(tooltipsPlugin({ isShownEmptyVal: false }));
    }

    return {
      axes: [
        {
          grid: { width: 1 },
          label: xLabel,
          scale: 'x',
          side: 2,
        },
        {
          grid: { width: 1 },
          label: yLabel,
          scale: 'y',
          side: 3,
        },
      ],
      height,
      legend: { show: showLegend },
      plugins,
      scales: {
        x: {
          range: xMin || xMax ? [Number(xMin), Number(xMax)] : undefined,
          time: false,
        },
        y: {
          distr: scale === Scale.Log ? 3 : 1,
        },
      },
      series: [
        { label: xLabel || 'X' },
        ...data.map((series, idx) => {
          return {
            label: seriesNames ? seriesNames[idx] : `Series ${idx + 1}`,
            scale: 'y',
            spanGaps: true,
            stroke: colors ? colors[idx] : glasbeyColor(idx),
          };
        }),
      ],
    };
  }, [
    closestPoint,
    colors,
    data,
    height,
    scale,
    seriesNames,
    showLegend,
    showTooltip,
    xLabel,
    xMax,
    xMin,
    yLabel,
  ]);

  return (
    <>
      {title && <h5 className={css.chartTitle}>{title}</h5>}
      <UPlotChart data={chartData} focusIndex={focusedSeries} options={chartOptions} />
    </>
  );
};

interface GroupProps {
  children: ReactElement[];
  data: number[][][];
}

export const ChartGroup: React.FC<GroupProps> = ({ children, data }: GroupProps) => {
  let xMin = Infinity,
    xMax = -Infinity;
  data.forEach((series) => {
    series.forEach((pt) => {
      if (!isFinite(xMin) || xMin === undefined) {
        xMin = pt[0];
        xMax = pt[0];
      } else {
        xMin = Math.min(xMin, pt[0]);
        xMax = Math.max(xMax, pt[0]);
      }
    });
  });

  return (
    <SyncProvider>
      {children.map((chart: ReactElement) => React.cloneElement(chart, { xMax, xMin }))}
    </SyncProvider>
  );
};
