import React, { useMemo, useState } from 'react';
import AutoSizer from 'react-virtualized-auto-sizer';
import { FixedSizeGrid } from 'react-window';
import uPlot, { AlignedData } from 'uplot';

import { XAxisDomain, XAxisFilter } from 'components/kit/LineChart/XAxisFilter';
import ScaleSelectFilter from 'components/ScaleSelectFilter';
import { SyncProvider } from 'components/UPlot/SyncProvider';
import UPlotChart, { Options } from 'components/UPlot/UPlotChart';
import { closestPointPlugin } from 'components/UPlot/UPlotChart/closestPointPlugin';
import { tooltipsPlugin } from 'components/UPlot/UPlotChart/tooltipsPlugin2';
import { glasbeyColor } from 'shared/utils/color';
import { MetricType, Scale } from 'types';

import css from './LineChart.module.scss';

/**
 * @typedef Serie
 * Represents a single Series to display on the chart.
 * @param {string} [color] - A CSS-compatible color to directly set the line and tooltip color for the Serie. Defaults to glasbeyColor.
 * @param {Partial<Record<XAxisDomain, [x: number, y: number][]>>} data - An array of ordered [x, y] points for each axis.
 * @param {MetricType} [metricType] - Indicator of a Serie representing a Training or Validation metric.
 * @param {string} [name] - Name to display in legend and toolip instead of Series number.
 */

export interface Serie {
  color?: string;
  data: Partial<Record<XAxisDomain, [x: number, y: number][]>>;
  key?: number;
  metricType?: MetricType;
  name?: string;
}

/**
 * @typedef Props {object}
 * Config for a single LineChart component.
 * @param {number} [focusedSeries] - Highlight one Serie's line and fade the others, given an index in the given series.
 * @param {number} [height=350] - Height in pixels.
 * @param {Scale} [scale=Scale.Linear] - Linear or Log Scale for the y-axis.
 * @param {Serie[]} series - Array of valid series to plot onto the chart.
 * @param {boolean} [showLegend=false] - Display a custom legend below the chart with each metric's color, name, and type.
 * @param {string} [title] - Title for the chart.
 * @param {XAxisDomain} [xAxis=XAxisDomain.Batches] - Set the x-axis of the chart (example: batches, time).
 * @param {string} [xLabel] - Directly set label below the x-axis.
 * @param {string} [yLabel] - Directly set label left of the y-axis.
 */
interface Props {
  focusedSeries?: number;
  height?: number;
  onSeriesClick?: (event: MouseEvent, arg1: number) => void;
  onSeriesFocus?: (arg0: number | null) => void;
  scale?: Scale;
  series: Serie[];
  showLegend?: boolean;
  title?: string;
  xAxis?: XAxisDomain;
  xLabel?: string;
  xTickValues?: uPlot.Axis.Values;
  yLabel?: string;
  yTickValues?: uPlot.Axis.Values;
}

export const LineChart: React.FC<Props> = ({
  focusedSeries,
  height = 350,
  onSeriesClick,
  onSeriesFocus,
  scale = Scale.Linear,
  series,
  showLegend = false,
  title,
  xAxis = XAxisDomain.Batches,
  xLabel,
  yLabel,
  xTickValues,
  yTickValues,
}: Props) => {
  const isMetricPair: boolean = useMemo(() => {
    const mTypes = series.map((s) => s.metricType);
    return (
      (series.length === 2 &&
        mTypes.includes(MetricType.Training) &&
        mTypes.includes(MetricType.Validation)) ||
      (series.length === 1 &&
        (mTypes.includes(MetricType.Training) || mTypes.includes(MetricType.Validation)))
    );
  }, [series]);

  const hasPopulatedSeries: boolean = useMemo(
    () => !!series.find((serie) => serie.data[xAxis]?.length),
    [series, xAxis],
  );

  const seriesColors: string[] = useMemo(
    () =>
      series.map(
        (s, idx) =>
          s.color ||
          (isMetricPair && s.metricType === MetricType.Training && '#009BDE') ||
          (isMetricPair && s.metricType === MetricType.Validation && '#F77B21') ||
          glasbeyColor(idx),
      ),
    [series, isMetricPair],
  );

  const seriesNames: string[] = useMemo(() => {
    return series.map(
      (s, idx) =>
        (s.metricType === MetricType.Training
          ? '[T] '
          : s.metricType === MetricType.Validation
          ? '[V] '
          : '') + (s.name || `Series ${idx + 1}`),
    );
  }, [series]);

  const chartData: AlignedData = useMemo(() => {
    const xSet = new Set<number>();
    const yValues: Record<string, Record<string, number | null>> = {};

    series.forEach((serie, serieIndex) => {
      yValues[serieIndex] = {};
      (serie.data[xAxis] || []).forEach((pt) => {
        const xVal = pt[0];
        xSet.add(xVal);
        yValues[serieIndex][xVal] = Number.isFinite(pt[1]) ? pt[1] : null;
      });
    });

    const xValues: number[] = Array.from(xSet);
    xValues.sort((a, b) => a - b);
    const yValuesArray: (number | null)[][] = Object.values(yValues).map((yValue) => {
      return xValues.map((xValue) => (yValue[xValue] != null ? yValue[xValue] : null));
    });

    return [xValues, ...yValuesArray];
  }, [series, xAxis]);

  const chartOptions: Options = useMemo(() => {
    const plugins = [
      tooltipsPlugin({
        isShownEmptyVal: false,
        // use specified color on Serie, or glasbeyColor
        seriesColors,
      }),
    ];
    if (onSeriesClick || onSeriesFocus) {
      plugins.push(
        closestPointPlugin({
          diamond: true,
          onPointClick: (e, point) => {
            if (onSeriesClick) {
              // correct seriesIdx (seriesIdx=0 on uPlot continues to be X)
              // return a serie.key (example: trialId), or the adjusted index
              onSeriesClick(e, series[point.seriesIdx - 1].key || point.seriesIdx - 1);
            }
          },
          onPointFocus: (point) => {
            if (onSeriesFocus) {
              // correct seriesIdx (seriesIdx=0 on uPlot continues to be X)
              // return a serie.key (example: trialId), or the adjusted index
              // returns null when switching to no point being hovered over
              onSeriesFocus(point ? series[point.seriesIdx - 1].key || point.seriesIdx - 1 : null);
            }
          },
          yScale: 'y',
        }),
      );
    }

    return {
      axes: [
        {
          font: '12px "Objektiv Mk3", Arial, Helvetica, sans-serif',
          grid: { show: false },
          label: xLabel,
          scale: 'x',
          side: 2,
          space: 120,
          ticks: { show: false },
          values: xTickValues,
        },
        {
          font: '12px "Objektiv Mk3", Arial, Helvetica, sans-serif',
          grid: { stroke: '#E3E3E3', width: 1 },
          label: yLabel,
          labelGap: 8,
          scale: 'y',
          side: 3,
          ticks: { show: false },
          values: yTickValues,
        },
      ],
      cursor: {
        drag: { x: true, y: false },
      },
      height: height - (hasPopulatedSeries ? 0 : 20),
      legend: { show: false },
      plugins,
      scales: {
        x: {
          time: xAxis === XAxisDomain.Time,
        },
        y: {
          distr: scale === Scale.Log ? 3 : 1,
        },
      },
      series: [
        { label: xAxis || xLabel || 'X' },
        ...series.map((serie, idx) => {
          return {
            label: seriesNames[idx],
            points: { show: (serie.data[xAxis] || []).length <= 1 },
            scale: 'y',
            spanGaps: true,
            stroke: seriesColors[idx],
            type: 'line',
            width: 2,
          };
        }),
      ],
    };
  }, [
    seriesColors,
    onSeriesClick,
    onSeriesFocus,
    xLabel,
    xTickValues,
    yLabel,
    yTickValues,
    height,
    xAxis,
    scale,
    series,
    seriesNames,
    hasPopulatedSeries,
  ]);

  return (
    <div className="diamond-cursor">
      {title && <h5 className={css.chartTitle}>{title}</h5>}
      <UPlotChart
        allowDownload={hasPopulatedSeries}
        data={chartData}
        focusIndex={focusedSeries}
        options={chartOptions}
      />
      {showLegend && (
        <div className={css.legendContainer}>
          {hasPopulatedSeries ? (
            series.map((s, idx) => (
              <li className={css.legendItem} key={idx}>
                <span className={css.colorButton} style={{ color: seriesColors[idx] }}>
                  &mdash;
                </span>
                {seriesNames[idx]}
              </li>
            ))
          ) : (
            <li>&nbsp;</li>
          )}
        </div>
      )}
    </div>
  );
};

export type ChartsProps = Props[];

/**
 * @typedef GroupProps {object}
 * Config for a grid of LineCharts.
 * @param {ChartsProps} chartsProps - Provide series to plot on each chart, and any chart-specific config.
 * @param {XAxisDomain[]} [xAxisOptions] - A list of possible x-axes to select in a dropdown; examples: Batches, Time, Epoch.
 */
interface GroupProps {
  chartsProps: ChartsProps;
}

export const ChartGrid: React.FC<GroupProps> = ({ chartsProps }: GroupProps) => {
  // Scale control
  const [scale, setScale] = useState<Scale>(Scale.Linear);

  // X-Axis control
  const xAxisOptions = useMemo(() => {
    const xOpts = new Set<string>();
    chartsProps.forEach((chart) => {
      chart.series.forEach((serie) => {
        Object.entries(serie.data).forEach(([xAxisOption, dataPoints]) => {
          if (dataPoints.length > 0) {
            xOpts.add(xAxisOption);
          }
        });
      });
    });
    return Array.from(xOpts).sort();
  }, [chartsProps]);
  const [xAxis, setXAxis] = useState<XAxisDomain>(XAxisDomain.Batches);

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
          {({ height, width }) => {
            const columnCount = Math.max(1, Math.floor(width / 540));
            return (
              <FixedSizeGrid
                columnCount={columnCount}
                columnWidth={Math.floor(width / columnCount)}
                height={Math.min(
                  height - 40,
                  (chartsProps.length > columnCount ? 2.1 : 1.05) * 480,
                )}
                rowCount={Math.ceil(chartsProps.length / columnCount)}
                rowHeight={480}
                width={width}>
                {({ columnIndex, rowIndex, style }) => {
                  const cellIndex = rowIndex * columnCount + columnIndex;
                  return (
                    <div className={css.chartgridCell} key={cellIndex} style={style}>
                      <div className={css.chartgridCellCard}>
                        {cellIndex < chartsProps.length && (
                          <LineChart {...chartsProps[cellIndex]} scale={scale} xAxis={xAxis} />
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
