import React, { useMemo, useState } from 'react';
import { AlignedData } from 'uplot';

import SelectFilter from 'components/SelectFilter';
import { SyncProvider } from 'components/UPlot/SyncableBounds';
import UPlotChart, { Options } from 'components/UPlot/UPlotChart';
import { closestPointPlugin } from 'components/UPlot/UPlotChart/closestPointPlugin';
import { tooltipsPlugin } from 'components/UPlot/UPlotChart/tooltipsPlugin';
import { glasbeyColor } from 'shared/utils/color';
import { Scale } from 'types';

import css from './LineChart.module.scss';

interface Series {
  color?: string;
  data: number[][];
  name?: string;
}

interface Props {
  closestPoint?: boolean;
  data: Series[];
  focusedSeries?: number;
  height?: number;
  onSeriesHover?: (seriesIdx: number | null) => void;
  onSeriesSelect?: (seriesIdx: number | null) => void;
  scale?: Scale;
  showLegend?: boolean;
  showTooltip?: boolean;
  title?: string;
  width?: number;
  xLabel?: string;
  yLabel?: string;
}

export const LineChart: React.FC<Props> = ({
  closestPoint = false,
  data,
  focusedSeries = -1,
  height = 400,
  onSeriesSelect,
  scale = Scale.Linear,
  showLegend = false,
  showTooltip = false,
  title,
  xLabel,
  yLabel,
}: Props) => {
  const [focusSeriesIdx, setFocusSeries] = useState<number>(focusedSeries);

  const chartData: AlignedData = useMemo(() => {
    const xValues: number[] = [];
    const yValues: Record<string, Record<string, number | null>> = {};

    data.forEach((series, seriesIndex) => {
      yValues[seriesIndex] = {};
      series.data.forEach((pt) => {
        xValues.push(pt[0]);
        yValues[seriesIndex][pt[0]] = Number.isFinite(pt[1]) ? pt[1] : null;
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
          //   console.log(point.seriesIdx);
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
            label: series.name ?? `Series ${idx + 1}`,
            scale: 'y',
            spanGaps: true,
            stroke: series.color ?? glasbeyColor(idx),
          };
        }),
      ],
    };
  }, [closestPoint, data, height, scale, showLegend, showTooltip, xLabel, yLabel]);

  return (
    <>
      {title && <h5 className={css.chartTitle}>{title}</h5>}
      {data.length > 1 && (
        <SelectFilter
          defaultValue={focusSeriesIdx}
          options={[
            {
              label: 'No metric selection',
              value: -1,
            },
            ...data.map((series, idx) => ({
              label: `Series ${idx}`,
              value: idx,
            })),
          ]}
          onSelect={(seriesKey) => {
            if (onSeriesSelect) {
              onSeriesSelect(focusSeriesIdx < 0 ? null : focusSeriesIdx);
            }
            setFocusSeries(Number(seriesKey));
          }}
        />
      )}
      <UPlotChart
        data={chartData}
        focusIndex={focusSeriesIdx < 0 ? undefined : focusSeriesIdx}
        options={chartOptions}
      />
    </>
  );
};

interface GroupProps {
  chartsProps: Props[];
  rowHeight?: number;
  showTooltip?: boolean;
}

export const ChartGrid: React.FC<GroupProps> = ({
  chartsProps,
  rowHeight,
  showTooltip,
}: GroupProps) => {
  // calculate xMin / xMax for shared group
  let xMin = Infinity,
    xMax = -Infinity;
  chartsProps.forEach((chartProp) => {
    chartProp.data.forEach((series) => {
      series.data.forEach((pt) => {
        if (!isFinite(xMin || 0)) {
          if (!isNaN(pt[0] * 1)) {
            xMin = pt[0];
            xMax = pt[0];
          }
        } else if (xMin !== undefined && xMax !== undefined) {
          xMin = Math.min(xMin, pt[0]);
          xMax = Math.max(xMax, pt[0]);
        }
      });
    });
  });

  return (
    <SyncProvider xMax={xMax} xMin={xMin}>
      {chartsProps.map((chartProp, cidx) => (
        <div key={cidx} style={{ width: '40%' }}>
          <LineChart
            {...chartProp}
            height={rowHeight}
            showTooltip={chartProp.showTooltip ?? showTooltip}
          />
        </div>
      ))}
    </SyncProvider>
  );
};
