import React, { ReactNode, useMemo } from 'react';
import { FixedSizeGrid, GridChildComponentProps } from 'react-window';
import uPlot, { AlignedData, Plugin } from 'uplot';

import {
  getCssVar,
  getTimeTickValues,
  glasbeyColor,
  metricToStr,
} from 'components/kit/internal/functions';
import Message from 'components/kit/internal/Message';
import ScaleSelect from 'components/kit/internal/ScaleSelect';
import { ErrorHandler, Scale, Serie, XAxisDomain } from 'components/kit/internal/types';
import { SyncProvider } from 'components/kit/internal/UPlot/SyncProvider';
import { UPlotPoint } from 'components/kit/internal/UPlot/types';
import UPlotChart, { Options } from 'components/kit/internal/UPlot/UPlotChart';
import { closestPointPlugin } from 'components/kit/internal/UPlot/UPlotChart/closestPointPlugin';
import { tooltipsPlugin } from 'components/kit/internal/UPlot/UPlotChart/tooltipsPlugin';
import useResize from 'components/kit/internal/useResize';
import XAxisFilter from 'components/kit/LineChart/XAxisFilter';
import Spinner from 'components/kit/Spinner';
import { Loadable } from 'components/kit/utils/loadable';

import css from './LineChart.module.scss';

export const TRAINING_SERIES_COLOR = '#009BDE';
export const VALIDATION_SERIES_COLOR = '#F77B21';

/**
 * @typedef ChartProps {object}
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
interface ChartProps {
  experimentId?: number;
  focusedSeries?: number;
  height?: number;
  onPointClick?: (event: MouseEvent, point: UPlotPoint) => void;
  onPointFocus?: (point: UPlotPoint | undefined) => void;
  plugins?: Plugin[];
  scale?: Scale;
  series: Serie[] | Loadable<Serie[]>;
  showLegend?: boolean;
  title?: ReactNode;
  xAxis?: XAxisDomain;
  xLabel?: string;
  yLabel?: string;
  yTickValues?: uPlot.Axis.Values;
}

interface LineChartProps extends Omit<ChartProps, 'series'> {
  series: Serie[] | Loadable<Serie[]>;
  handleError: ErrorHandler;
}

export const LineChart: React.FC<LineChartProps> = ({
  experimentId,
  focusedSeries,
  handleError,
  height = 350,
  onPointClick,
  onPointFocus,
  scale = Scale.Linear,
  plugins: propPlugins,
  series: propSeries,
  showLegend = false,
  title,
  xAxis = XAxisDomain.Batches,
  xLabel,
  yLabel,
  yTickValues,
}: LineChartProps) => {
  const series = Loadable.isLoadable(propSeries) ? Loadable.getOrElse([], propSeries) : propSeries;
  const isLoading = Loadable.isLoadable(propSeries) && Loadable.isNotLoaded(propSeries);

  const hasPopulatedSeries: boolean = useMemo(
    () => !!series.find((serie) => serie.data[xAxis]?.length),
    [series, xAxis],
  );

  const seriesColors: string[] = useMemo(
    () => series.map((s, i) => s.color ?? glasbeyColor(i)),
    [series],
  );

  const seriesNames: string[] = useMemo(() => {
    return series.map((s) => {
      return metricToStr({ group: s.metricType ?? 'unknown', name: s.name ?? 'unknown' });
    });
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

  const xTickValues: uPlot.Axis.Values | undefined = useMemo(
    () =>
      xAxis === XAxisDomain.Time &&
      chartData.length > 0 &&
      chartData[0].length > 0 &&
      chartData[0][chartData[0].length - 1] - chartData[0][0] < 43200 // 12 hours
        ? getTimeTickValues
        : undefined,
    [chartData, xAxis],
  );

  const chartOptions: Options = useMemo(() => {
    const plugins: Plugin[] = propPlugins ?? [
      tooltipsPlugin({
        closeOnMouseExit: true,
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
          font: `12px ${getCssVar('--theme-font-family')}`,
          grid: { show: false },
          incrs:
            xAxis === XAxisDomain.Time
              ? undefined
              : [
                  /* eslint-disable array-element-newline */
                  1, 2, 3, 4, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10_000, 25_000,
                  50_000, 100_000, 250_000, 500_000, 1_000_000, 2_500_000, 5_000_000,
                  /* eslint-enable array-element-newline */
                ],
          label: xLabel,
          scale: 'x',
          side: 2,
          space: 120,
          ticks: { show: false },
          values: xTickValues,
        },
        {
          font: `12px ${getCssVar('--theme-font-family')}`,
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
            alpha: focusedSeries === undefined || focusedSeries === idx ? 1 : 0.4,
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
    focusedSeries,
  ]);

  return (
    <div className="diamond-cursor">
      {title && <h5 className={css.chartTitle}>{title}</h5>}
      <UPlotChart
        allowDownload={hasPopulatedSeries}
        data={chartData}
        experimentId={experimentId}
        handleError={handleError}
        isLoading={isLoading}
        options={chartOptions}
        xAxis={xAxis}
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

export type ChartsProps = ChartProps[];

/**
 * @typedef GroupProps {object}
 * Config for a grid of LineCharts.
 * @param {ChartsProps} chartsProps - Provide series to plot on each chart, and any chart-specific config.
 * @param {XAxisDomain[]} [xAxisOptions] - A list of possible x-axes to select in a dropdown; examples: Batches, Time, Epoch.
 * @param {Scale} scale - Scale of chart, can be linear or log
 * @param {handleError} handleError - Error handler
 */
export interface GroupProps {
  chartsProps: ChartsProps | Loadable<ChartsProps>;
  onXAxisChange: (ax: XAxisDomain) => void;
  scale: Scale;
  setScale: React.Dispatch<React.SetStateAction<Scale>>;
  xAxis: XAxisDomain;
  handleError: ErrorHandler;
}

/**
 * VirtualChartRenderer is used by FixedSizeGrid to virtually render individual charts.
 * `data` comes from the itemData prop that is passed to FixedSizeGrid.
 */
const VirtualChartRenderer: React.FC<
  GridChildComponentProps<{
    chartsProps: ChartsProps;
    columnCount: number;
    scale: Scale;
    xAxis: XAxisDomain;
    handleError: ErrorHandler;
  }>
> = ({ columnIndex, rowIndex, style, data }) => {
  const { chartsProps, columnCount, scale, xAxis, handleError } = data;

  const cellIndex = rowIndex * columnCount + columnIndex;

  if (chartsProps === undefined || cellIndex >= chartsProps.length) return null;
  const chartProps = chartsProps[cellIndex];

  return (
    <div className={css.chartgridCell} key={`${rowIndex}, ${columnIndex}`} style={style}>
      <div className={css.chartgridCellCard}>
        <LineChart {...chartProps} handleError={handleError} scale={scale} xAxis={xAxis} />
      </div>
    </div>
  );
};

export const ChartGrid: React.FC<GroupProps> = React.memo(
  ({
    chartsProps: propChartsProps,
    xAxis,
    onXAxisChange,
    scale,
    setScale,
    handleError,
  }: GroupProps) => {
    const { refCallback, size } = useResize();
    const height = size.height ?? 0;
    const width = size.width ?? 0;
    const columnCount = Math.max(1, Math.floor(width / 540));
    const chartsProps = (
      Loadable.isLoadable(propChartsProps)
        ? Loadable.getOrElse([], propChartsProps)
        : propChartsProps
    ).filter(
      (c) =>
        // filter out Loadable series which are Loaded yet have no serie with more than 0 points.
        !Loadable.isLoadable(c.series) ||
        !Loadable.isLoaded(c.series) ||
        Loadable.getOrElse([], c.series).find((serie) =>
          Object.entries(serie.data).find(([, points]) => points.length > 0),
        ),
    );
    const isLoading = Loadable.isLoadable(propChartsProps) && Loadable.isNotLoaded(propChartsProps);
    // X-Axis control

    const xAxisOptions = useMemo(() => {
      const xOpts = new Set<string>();
      chartsProps.forEach((chart) => {
        const series = Loadable.isLoadable(chart.series)
          ? Loadable.getOrElse([], chart.series)
          : chart.series;
        series.forEach((serie) => {
          Object.entries(serie.data).forEach(([xAxisOption, dataPoints]) => {
            if (dataPoints.length > 0) {
              xOpts.add(xAxisOption);
            }
          });
        });
      });
      return Array.from(xOpts).sort();
    }, [chartsProps]);

    if (chartsProps.length === 0 && !isLoading) return <Message title="No data available." />;

    return (
      <div className={css.scrollContainer}>
        <div className={css.chartgridContainer} ref={refCallback}>
          <Spinner center spinning={isLoading} tip="Loading chart data...">
            {chartsProps.length > 0 && (
              <>
                <div className={css.filterContainer}>
                  <ScaleSelect value={scale} onChange={setScale} />
                  {xAxisOptions && xAxisOptions.length > 1 && (
                    <XAxisFilter options={xAxisOptions} value={xAxis} onChange={onXAxisChange} />
                  )}
                </div>
                <SyncProvider>
                  <FixedSizeGrid
                    columnCount={columnCount}
                    columnWidth={Math.floor(width / columnCount)}
                    height={height - 40}
                    itemData={{
                      chartsProps,
                      columnCount,
                      handleError,
                      scale,
                      xAxis,
                    }}
                    rowCount={Math.ceil(chartsProps.length / columnCount)}
                    rowHeight={465}
                    style={{ height: '100%' }}
                    width={width}>
                    {VirtualChartRenderer}
                  </FixedSizeGrid>
                </SyncProvider>
              </>
            )}
          </Spinner>
        </div>
      </div>
    );
  },
);
