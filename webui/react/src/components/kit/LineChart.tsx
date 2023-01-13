import React, { useMemo, useState } from 'react';
import AutoSizer from 'react-virtualized-auto-sizer';
import { FixedSizeGrid } from 'react-window';
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
  onXAxisSelect?: (axisName: string) => void;
  scale?: Scale;
  showLegend?: boolean;
  showTooltip?: boolean;
  title?: string;
  width?: number;
  xAxisOptions?: string[];
  xLabel?: string;
  yLabel?: string;
}

export const LineChart: React.FC<Props> = ({
  closestPoint = false,
  data,
  focusedSeries = -1,
  height = 400,
  onSeriesSelect,
  onXAxisSelect,
  scale = Scale.Linear,
  showLegend = false,
  showTooltip = false,
  title,
  xAxisOptions = [],
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
      {xAxisOptions && xAxisOptions.length > 1 && onXAxisSelect && (
        <SelectFilter
          defaultValue={xAxisOptions[0]}
          options={xAxisOptions.map((axisName) => ({
            label: axisName,
            value: axisName,
          }))}
          onSelect={(axisName) => {
            onXAxisSelect(String(axisName));
          }}
        />
      )}
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
      <AutoSizer>
        {({ width }) => {
          const columnCount = Math.max(1, Math.floor(width / 540));
          return (
            <FixedSizeGrid
              columnCount={columnCount}
              columnWidth={Math.floor(width / columnCount) - 10}
              height={chartsProps.length > 1 ? 1000 : 500}
              rowCount={Math.ceil(chartsProps.length / columnCount)}
              rowHeight={rowHeight ?? 480}
              width={width}>
              {({ columnIndex, rowIndex, style }) => {
                const cellIndex = rowIndex * columnCount + columnIndex;
                return (
                  <div key={cellIndex} style={style}>
                    {cellIndex < chartsProps.length && (
                      <LineChart
                        {...chartsProps[cellIndex]}
                        height={rowHeight}
                        showTooltip={chartsProps[cellIndex].showTooltip ?? showTooltip}
                      />
                    )}
                  </div>
                );
              }}
            </FixedSizeGrid>
          );
        }}
      </AutoSizer>
    </SyncProvider>
  );
};
