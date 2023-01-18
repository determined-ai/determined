import React, { useMemo } from 'react';
import AutoSizer from 'react-virtualized-auto-sizer';
import { FixedSizeGrid } from 'react-window';
import { AlignedData } from 'uplot';

import { SyncProvider } from 'components/UPlot/SyncableBounds';
import UPlotChart, { Options } from 'components/UPlot/UPlotChart';
import { tooltipsPlugin } from 'components/UPlot/UPlotChart/tooltipsPlugin2';
import { glasbeyColor } from 'shared/utils/color';
import { Scale } from 'types';

import css from './LineChart.module.scss';

export interface Serie {
  color?: string;
  data: (number | null)[];
  name?: string;
}

interface Props {
  focusedSeries?: number;
  height?: number;
  scale?: Scale;
  series: Serie[];
  showLegend?: boolean;
  title?: string;
  xLabel?: string;
  yLabel?: string;
}

export const LineChart: React.FC<Props> = ({
  focusedSeries,
  height = 400,
  scale = Scale.Linear,
  series,
  showLegend = false,
  title,
  xLabel,
  yLabel,
}: Props) => {
  const chartOptions: Options = useMemo(() => {
    const plugins = [tooltipsPlugin({ isShownEmptyVal: false })];

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
      cursor: { drag: { x: true, y: false } },
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
        ...series.slice(1).map((serie, idx) => {
          return {
            label: serie.name ?? `Series ${idx + 1}`,
            scale: 'y',
            spanGaps: true,
            stroke: serie.color ?? glasbeyColor(idx),
          };
        }),
      ],
    };
  }, [series, height, scale, showLegend, xLabel, yLabel]);

  return (
    <>
      {title && <h5 className={css.chartTitle}>{title}</h5>}
      <UPlotChart
        data={series.map((s) => s.data) as AlignedData}
        focusIndex={focusedSeries}
        options={chartOptions}
      />
    </>
  );
};

interface GroupProps {
  chartsProps: Props[];
  rowHeight?: number;
  scale?: Scale;
}

export const ChartGrid: React.FC<GroupProps> = ({
  chartsProps,
  rowHeight = 480,
  scale = Scale.Linear,
}: GroupProps) => {
  // calculate xMin / xMax for shared group
  let xMin = Infinity,
    xMax = -Infinity;
  chartsProps.forEach((chartProp) => {
    chartProp.series[0].data.forEach((xVal) => {
      if (!isFinite(xMin || 0)) {
        if (xVal !== null && !isNaN(xVal * 1)) {
          xMin = xVal;
          xMax = xVal;
        }
      } else if (xMin !== undefined && xMax !== undefined && xVal !== null) {
        xMin = Math.min(xMin, xVal);
        xMax = Math.max(xMax, xVal);
      }
    });
  });

  return (
    <SyncProvider>
      <AutoSizer>
        {({ width }) => {
          const columnCount = Math.max(1, Math.floor(width / 540));
          return (
            <FixedSizeGrid
              columnCount={columnCount}
              columnWidth={Math.floor(width / columnCount) - 10}
              height={(chartsProps.length > columnCount ? 2.1 : 1.05) * (rowHeight ?? 480)}
              rowCount={Math.ceil(chartsProps.length / columnCount)}
              rowHeight={rowHeight ?? 480}
              width={width}>
              {({ columnIndex, rowIndex, style }) => {
                const cellIndex = rowIndex * columnCount + columnIndex;
                return (
                  <div key={cellIndex} style={style}>
                    {cellIndex < chartsProps.length && (
                      <LineChart {...chartsProps[cellIndex]} height={rowHeight} scale={scale} />
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
