import React, { useMemo, useRef, useState } from 'react';
import { FixedSizeGrid, GridChildComponentProps } from 'react-window';
import uPlot, { AlignedData, Plugin } from 'uplot';

import { XAxisDomain, XAxisFilter } from 'components/kit/LineChart/XAxisFilter';
import ScaleSelectFilter from 'components/ScaleSelectFilter';
import { SyncProvider } from 'components/UPlot/SyncProvider';
import { UPlotPoint } from 'components/UPlot/types';
import UPlotChart, { Options } from 'components/UPlot/UPlotChart';
import { closestPointPlugin } from 'components/UPlot/UPlotChart/closestPointPlugin';
import { tooltipsPlugin } from 'components/UPlot/UPlotChart/tooltipsPlugin2';
import useResize from 'hooks/useResize';
import { glasbeyColor } from 'shared/utils/color';
import { MetricType, Scale } from 'types';

import css from './LineChart.module.scss';

export const TRAINING_SERIES_COLOR = '#009BDE';
export const VALIDATION_SERIES_COLOR = '#F77B21';

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
  onPointClick?: (event: MouseEvent, point: UPlotPoint) => void;
  onPointFocus?: (point: UPlotPoint | undefined) => void;
  plugins?: Plugin[];
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
  onPointClick,
  onPointFocus,
  scale = Scale.Linear,
  plugins: propPlugins,
  series,
  showLegend = false,
  title,
  xAxis = XAxisDomain.Batches,
  xLabel,
  yLabel,
  xTickValues,
  yTickValues,
}: Props) => {
  const hasPopulatedSeries: boolean = useMemo(
    () => !!series.find((serie) => serie.data[xAxis]?.length),
    [series, xAxis],
  );

  const seriesColors: string[] = useMemo(
    () => series.map((s, i) => s.color ?? glasbeyColor(i)),
    [series],
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
    const plugins: Plugin[] = propPlugins ?? [
      tooltipsPlugin({
        isShownEmptyVal: false,
        // use specified color on Serie, or glasbeyColor
        seriesColors,
      }),
      closestPointPlugin({
        onPointClick,
        onPointFocus,
        yScale: 'y',
      }),
    ];

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
        { label: xLabel ?? xAxis ?? 'X' },
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
    onPointClick,
    onPointFocus,
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
    propPlugins,
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
 * @param {Scale} scale - Scale of chart, can be linear or log
 */
export interface GroupProps {
  chartsProps: ChartsProps;
  onXAxisChange: (ax: XAxisDomain) => void;
  scale: Scale;
  setScale: React.Dispatch<React.SetStateAction<Scale>>;
  xAxis: XAxisDomain;
}

/**
 * VirtualChartRenderer is used by FixedSizeGrid to virtually render individual charts.
 * `data` comes from the itemData prop that is passed to FixedSizeGrid.
 */
const VirtualChartRenderer: React.FC<
  GridChildComponentProps<{ chartsProps: ChartsProps; columnCount: number }>
> = ({ columnIndex, rowIndex, style, data }) => {
  const { chartsProps, columnCount } = data;

  const cellIndex = rowIndex * columnCount + columnIndex;

  if (chartsProps === undefined || cellIndex >= chartsProps.length) return null;
  const chartProps = chartsProps[cellIndex];

  return (
    <div className={css.chartgridCell} key={`${rowIndex}, ${columnIndex}`} style={style}>
      <div className={css.chartgridCellCard}>
        <LineChart {...chartProps} scale={Scale.Linear} xAxis={XAxisDomain.Batches} />
      </div>
    </div>
  );
};

export const ChartGrid: React.FC<GroupProps> = React.memo(
  ({ chartsProps, xAxis, onXAxisChange, scale, setScale }: GroupProps) => {
    const chartGridRef = useRef<HTMLDivElement | null>(null);
    const { width, height } = useResize(chartGridRef);
    const columnCount = Math.max(1, Math.floor(width / 540));

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

    return (
      <div className={css.chartgridContainer} ref={chartGridRef}>
        <div className={css.filterContainer}>
          <ScaleSelectFilter value={scale} onChange={setScale} />
          {xAxisOptions && xAxisOptions.length > 1 && (
            <XAxisFilter options={xAxisOptions} value={xAxis} onChange={onXAxisChange} />
          )}
        </div>
        <SyncProvider>
          <FixedSizeGrid
            columnCount={columnCount}
            columnWidth={Math.floor(width / columnCount)}
            height={Math.min(height - 40, (chartsProps.length > columnCount ? 2.1 : 1.05) * 480)}
            itemData={{ chartsProps, columnCount }}
            rowCount={Math.ceil(chartsProps.length / columnCount)}
            rowHeight={480}
            width={width}>
            {VirtualChartRenderer}
          </FixedSizeGrid>
        </SyncProvider>
      </div>
    );
  },
);
