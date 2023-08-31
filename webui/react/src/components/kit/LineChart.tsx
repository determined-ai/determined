import _ from 'lodash';
import React, { ReactNode, useMemo, useRef } from 'react';
import { FixedSizeGrid, GridChildComponentProps } from 'react-window';
import uPlot, { AlignedData, Plugin } from 'uplot';

import { getCssVar, getTimeTickValues, glasbeyColor } from 'components/kit/internal/functions';
import ScaleSelect from 'components/kit/internal/ScaleSelect';
import { ErrorHandler, Scale } from 'components/kit/internal/types';
import { SyncProvider } from 'components/kit/internal/UPlot/SyncProvider';
import { UPlotPoint } from 'components/kit/internal/UPlot/types';
import UPlotChart, { Options } from 'components/kit/internal/UPlot/UPlotChart';
import { closestPointPlugin } from 'components/kit/internal/UPlot/UPlotChart/closestPointPlugin';
import { tooltipsPlugin } from 'components/kit/internal/UPlot/UPlotChart/tooltipsPlugin';
import useResize from 'components/kit/internal/useResize';
import { XAxisDomain, XAxisFilter } from 'components/kit/LineChart/XAxisFilter';
import css from 'components/kit/LineChart.module.scss';
import Spinner from 'components/kit/Spinner';
import Message from 'components/Message';
import MetricBadgeTag from 'components/MetricBadgeTag';
import { MapOfIdsToColors } from 'hooks/useGlasbey';
import { TrialMetricData } from 'pages/TrialDetails/useTrialMetrics';
import { ExperimentWithTrial, TrialItem } from 'types';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';
import { metricToKey, metricToStr } from 'utils/metric';

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
  metricType?: string;
  name?: string;
}

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
  const isLoading = Loadable.isLoadable(propSeries) && Loadable.isLoading(propSeries);

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

export const calculateChartProps = (
  metricData: TrialMetricData,
  experiments: ExperimentWithTrial[],
  trials: TrialItem[],
  xAxis: XAxisDomain,
  colorMap: MapOfIdsToColors,
): Loadable<ChartsProps> => {
  const { metricHasData, metrics, data, isLoaded, selectedMetrics } = metricData;
  const chartedMetrics: Record<string, boolean> = {};
  const out: ChartsProps = [];
  const expNameById: Record<number, string> = {};
  experiments.forEach((e) => {
    expNameById[e.experiment.id] = e.experiment.name;
  });
  metrics.forEach((metric) => {
    const series: Serie[] = [];
    const key = metricToKey(metric);
    trials.forEach((t) => {
      const m = data[t?.id || 0];
      m?.[key] &&
        t &&
        series.push({
          ...m[key],
          color: colorMap[t.experimentId],
          metricType: undefined,
          name: expNameById[t.experimentId]
            ? `${expNameById[t.experimentId]} (${t.experimentId})`
            : String(t.experimentId),
        });
      chartedMetrics[key] ||= series.length > 0;
    });
    out.push({
      series: Loaded(series),
      title: <MetricBadgeTag metric={metric} />,
      xAxis,
      xLabel: String(xAxis),
    });
  });

  // In order to show the spinner for each chart in the ChartGrid until
  // metrics are visible, we must determine whether the metrics have been
  // loaded and whether the chart props have been updated.
  // If any metric has data but no chartProps contain data for the metric,
  // then the charts have not been updated and we need to continue to show the
  // spinner.
  const chartDataIsLoaded = metrics.every((metric) => {
    const metricKey = metricToKey(metric);
    return metricHasData?.[metricKey] ? !!chartedMetrics?.[metricKey] : true;
  });
  if (!isLoaded) {
    // When trial metrics hasn't loaded metric names or individual trial metrics.
    return NotLoaded;
  } else if (!chartDataIsLoaded || !_.isEqual(selectedMetrics, metrics)) {
    // In some cases the selectedMetrics returned may not be up to date
    // with the metrics selected by the user. In this case we want to
    // show a loading state until the metrics match.

    // returns the chartProps with a NotLoaded series which enables
    // the ChartGrid to show a spinner for the loading charts.
    return Loaded(out.map((chartProps) => ({ ...chartProps, series: NotLoaded })));
  } else {
    return Loaded(out);
  }
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
    const chartGridRef = useRef<HTMLDivElement | null>(null);
    const { width, height } = useResize(chartGridRef);
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
    const isLoading = Loadable.isLoadable(propChartsProps) && Loadable.isLoading(propChartsProps);
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
        <div className={css.chartgridContainer} ref={chartGridRef}>
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
