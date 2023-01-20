import React, { useMemo, useState } from 'react';
import AutoSizer from 'react-virtualized-auto-sizer';
import { FixedSizeGrid } from 'react-window';
import { AlignedData } from 'uplot';

import XAxisFilter from 'components/kit/LineChart/XAxisFilter';
import ScaleSelectFilter from 'components/ScaleSelectFilter';
import { SyncProvider } from 'components/UPlot/SyncProvider';
import UPlotChart, { Options } from 'components/UPlot/UPlotChart';
import { tooltipsPlugin } from 'components/UPlot/UPlotChart/tooltipsPlugin2';
import { glasbeyColor } from 'shared/utils/color';
import { Metric, MetricType, Scale } from 'types';

import css from './LineChart.module.scss';

export interface Serie {
  color?: string;
  data: (number | null)[];
  metricType?: MetricType;
}

interface Props {
  focusedSeries?: number;
  height?: number;
  metric: Metric;
  scale?: Scale;
  series: Serie[];
  showLegend?: boolean;
  xLabel?: string;
  yLabel?: string;
}

export const LineChart: React.FC<Props> = ({
  focusedSeries,
  height = 350,
  metric,
  scale = Scale.Linear,
  series,
  showLegend = false,
  xLabel,
  yLabel,
}: Props) => {
  const seriesColors: string[] = useMemo(
    () =>
      series
        .slice(1)
        .map(
          (s, idx) =>
            s.color ||
            (s.metricType === MetricType.Training && '#009BDE') ||
            (s.metricType === MetricType.Validation && '#F77B21') ||
            glasbeyColor(idx),
        ),
    [series],
  );

  const seriesNames: string[] = useMemo(
    () =>
      series
        .slice(1)
        .map(
          (s, idx) =>
            (series.length > 2 ? `${s.metricType}_` : '') + (metric.name || `Series ${idx + 1}`),
        ),
    [series, metric.name],
  );

  const chartOptions: Options = useMemo(() => {
    const plugins = [tooltipsPlugin({ isShownEmptyVal: false, seriesColors })];

    return {
      axes: [
        {
          font: '12px "Objektiv Mk3", Arial, Helvetica, sans-serif',
          grid: { show: false },
          label: xLabel,
          scale: 'x',
          side: 2,
          ticks: { show: false },
        },
        {
          font: '12px "Objektiv Mk3", Arial, Helvetica, sans-serif',
          grid: { stroke: '#E3E3E3', width: 1 },
          label: yLabel,
          scale: 'y',
          side: 3,
          ticks: { show: false },
        },
      ],
      cursor: {
        drag: { x: true, y: false },
      },
      height,
      legend: { show: false },
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
            label: seriesNames[idx],
            points: { show: false },
            scale: 'y',
            spanGaps: true,
            stroke: seriesColors[idx],
            type: 'line',
            width: 2,
          };
        }),
      ],
    };
  }, [series, seriesColors, seriesNames, height, scale, xLabel, yLabel]);

  return (
    <>
      {metric.name && <h5 className={css.chartTitle}>{metric.name}</h5>}
      <UPlotChart
        allowDownload
        data={series.map((s) => s.data) as AlignedData}
        focusIndex={focusedSeries}
        options={chartOptions}
      />
      {showLegend && (
        <div className={css.legendContainer}>
          {series.slice(1).map((s, idx) => (
            <li className={css.legendItem} key={idx}>
              <span className={css.colorButton} style={{ color: seriesColors[idx] }}>
                &mdash;
              </span>
              {seriesNames[idx]}
            </li>
          ))}
        </div>
      )}
    </>
  );
};

interface GroupProps {
  chartsProps: Props[];
  xAxisOptions?: string[];
}

export const ChartGrid: React.FC<GroupProps> = ({ chartsProps, xAxisOptions }: GroupProps) => {
  // Scale control
  const [scale, setScale] = useState<Scale>(Scale.Linear);

  // X-Axis control
  const [xAxis, setXAxis] = useState<string>('Batches');

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
    <div className={css.chartgridContainer}>
      <div className={css.filterContainer}>
        <ScaleSelectFilter value={scale} onChange={setScale} />
        {xAxisOptions && xAxisOptions.length > 1 && (
          <XAxisFilter options={xAxisOptions} value={xAxis} onChange={setXAxis} />
        )}
      </div>
      <SyncProvider>
        <AutoSizer>
          {({ width }) => {
            const columnCount = Math.max(1, Math.floor(width / 540));
            return (
              <FixedSizeGrid
                columnCount={columnCount}
                columnWidth={Math.floor(width / columnCount)}
                height={(chartsProps.length > columnCount ? 2.1 : 1.05) * 480}
                rowCount={Math.ceil(chartsProps.length / columnCount)}
                rowHeight={480}
                width={width}>
                {({ columnIndex, rowIndex, style }) => {
                  const cellIndex = rowIndex * columnCount + columnIndex;
                  return (
                    <div className={css.chartgridCell} key={cellIndex} style={style}>
                      <div className={css.chartgridCellCard}>
                        {cellIndex < chartsProps.length && (
                          <LineChart {...chartsProps[cellIndex]} scale={scale} />
                        )}
                      </div>
                    </div>
                  );
                }}
              </FixedSizeGrid>
            );
          }}
        </AutoSizer>
      </SyncProvider>
    </div>
  );
};
