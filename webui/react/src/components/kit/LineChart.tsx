import dayjs from 'dayjs';
import { ECElementEvent, EChartsOption } from 'echarts';
import { CallbackDataParams, TopLevelFormatterParams } from 'echarts/types/dist/shared';
import _ from 'lodash';
import React, { ReactNode, useMemo } from 'react';
import { FixedSizeGrid, GridChildComponentProps } from 'react-window';

import ScaleSelect from 'components/kit/internal/ScaleSelect';
import {
  CheckpointsDict,
  ErrorHandler,
  Scale,
  Serie,
  XAxisDomain,
} from 'components/kit/internal/types';
import useResize from 'components/kit/internal/useResize';
import XAxisFilter from 'components/kit/LineChart/XAxisFilter';
import Message from 'components/kit/Message';
import ReactECharts, { EchartsEventFunction } from 'components/kit/ReactEchart';
import Spinner from 'components/kit/Spinner';
import { Loadable } from 'components/kit/utils/loadable';
import { alphaNumericSorter } from 'utils/sort';

import css from './LineChart.module.scss';

type SeriesDataType = {
  x: number[];
  y: { data: (number | '-')[]; name?: string; key?: number; color?: string }[];
};

export const TRAINING_SERIES_COLOR = '#009BDE';
export const VALIDATION_SERIES_COLOR = '#F77B21';

export type OnClickPointType = (id: number, data: ECElementEvent['data']) => void;

// JS Date adjust time to local time instead of UTC
// This func adjusts the time for UTC
const adjustLocalTimeToUTC = (unixTime: number): Date => {
  return dayjs.unix(unixTime).subtract(dayjs.unix(unixTime).utcOffset(), 'm').toDate();
};

/**
 * @typedef ChartProps {object}
 * Config for a single LineChart component.
 * @param {number} [height=360] - Height in pixels.
 * @param {Scale} [scale=Scale.Linear] - Linear or Log Scale for the y-axis.
 * @param {Serie[]} series - Array of valid series to plot onto the chart.
 * @param {boolean} [showLegend=false] - Display a custom legend below the chart with each metric's color, name, and type.
 * @param {string} [title] - Title for the chart.
 * @param {XAxisDomain} [xAxis=XAxisDomain.Batches] - Set the x-axis of the chart (example: batches, time).
 * @param {string} [xLabel] - Directly set label below the x-axis.
 * @param {string} [yLabel] - Directly set label left of the y-axis.
 */
interface ChartProps {
  height?: number;
  group?: string;
  onClickPoint?: OnClickPointType;
  scale?: Scale;
  yValueFormatter?: (value: number) => string;
  xValueRange?: [min: number, max: number];
  confine?: boolean;
  checkpointsDict?: CheckpointsDict;
  series: Serie[] | Loadable<Serie[]>;
  showLegend?: boolean;
  title?: ReactNode;
  xAxis?: XAxisDomain;
  xLabel?: string;
  yLabel?: string;
}

interface LineChartProps extends Omit<ChartProps, 'series'> {
  series: Serie[] | Loadable<Serie[]>;
  handleError?: ErrorHandler;
}

export const LineChart: React.FC<LineChartProps> = ({
  height = 360,
  onClickPoint,
  scale = Scale.Linear,
  group,
  yValueFormatter,
  xValueRange,
  series: propSeries,
  showLegend = false,
  confine = false,
  title,
  xAxis = XAxisDomain.Batches,
  xLabel,
  yLabel,
  checkpointsDict,
}: LineChartProps) => {
  const series = useMemo(() => {
    return Loadable.ensureLoadable(propSeries).getOrElse([]);
  }, [propSeries]);

  const seriesData: SeriesDataType = useMemo(() => {
    const allXValues: number[] = Array.from(
      new Set(
        series.flatMap((s) => {
          const data = s.data[xAxis];
          return data === undefined ? [] : data.map((d) => d[0]);
        }),
      ),
    ).sort((a, b) => alphaNumericSorter(a, b));
    const data: SeriesDataType = { x: allXValues, y: [] };
    for (let i = 0; i < series.length; i++) {
      data.y.push({ color: series[i].color, data: [], key: series[i].key, name: series[i].name });
      const map = new Map();
      for (const [key, value] of series[i].data[xAxis] ?? []) {
        map.set(key, value);
      }
      for (const xVal of allXValues) {
        data.y[i].data.push(map.has(xVal) ? map.get(xVal) : '-');
      }
    }
    return data;
  }, [series, xAxis]);

  const isLoading = Loadable.isLoadable(propSeries) && Loadable.isNotLoaded(propSeries);

  const EventFunctions: EchartsEventFunction[] = useMemo(() => {
    return [
      {
        eventName: 'click',
        handler: (params) => {
          if (xAxis !== XAxisDomain.Time) {
            onClickPoint?.(Number(params.name), [Number(params.name), Number(params.value)]);
          }
        },
      },
    ];
  }, [onClickPoint, xAxis]);

  const echartOption: EChartsOption = useMemo(() => {
    let currentYAxis = 0;

    const formatterFunc = (params: TopLevelFormatterParams) => {
      type SeriesType = {
        marker: CallbackDataParams['marker'];
        seriesName: string;
        value: number;
      };
      const data = params as CallbackDataParams[];

      const seriesList: SeriesType[] = data
        .filter((d: CallbackDataParams) => d.seriesName && typeof d.value === 'number')
        .map((d: CallbackDataParams): SeriesType => {
          return {
            marker: d.marker,
            seriesName: d.seriesName ?? '',
            value: typeof d.value === 'number' ? d.value : 0,
          };
        })
        .sort((a: SeriesType, b: SeriesType) => b.value - a.value);

      const closestPoint = [...seriesList]
        .sort((a: SeriesType, b: SeriesType) =>
          Math.abs(a.value - currentYAxis) > Math.abs(b.value - currentYAxis) ? 1 : -1,
        )
        .shift();

      const tooltip = `
        <div style="font-size: 11px;">
          <div>${xAxis === XAxisDomain.Batches ? 'Batches Processed' : ''} ${data[0].name}</div>
          ${seriesList
            .map((d) => {
              const fontWeight = closestPoint?.seriesName === d.seriesName ? 'bold' : 'normal';
              return `
              <div>
                ${d.marker}
                <span style="font-weight: ${fontWeight};">${d.seriesName}</span>:
                ${yValueFormatter?.(d.value) ?? d.value.toFixed(4)}
              </div>`;
            })
            .join('')}
        </div>
      `;
      return tooltip;
    };

    const generateOption = (): EChartsOption => {
      if (xAxis === XAxisDomain.Time) {
        const option: EChartsOption = {
          series: series.map((serie) => ({
            connectNulls: true,
            data: (() => {
              const set = new Set();
              const arr: [x: Date, y: number][] = [];
              for (const d of serie.data[xAxis] ?? []) {
                const [xValue, yValue] = d;
                if (set.has(xValue)) {
                  continue;
                }
                set.add(xValue);
                arr.push([adjustLocalTimeToUTC(xValue), yValue]);
              }
              return arr;
            })(),
            emphasis: { focus: 'series' },
            id: serie.key,
            itemStyle: { color: serie.color },
            name: serie.name ?? yLabel,
            type: 'line',
          })),
          xAxis: {
            max: xValueRange ? adjustLocalTimeToUTC(xValueRange[1]) : undefined,
            min: xValueRange ? adjustLocalTimeToUTC(xValueRange[0]) : undefined,
            name: xLabel,
            type: 'time',
          },
        };
        return option;
      } else {
        const option: EChartsOption = {
          series: seriesData.y.map((serie) => ({
            connectNulls: true,
            data: serie.data,
            emphasis: { focus: 'series' },
            id: serie.key,
            itemStyle: { color: serie.color },
            name: serie.name ?? yLabel,
            symbol: (value, params) => {
              if (checkpointsDict === undefined) return 'circle';
              return Number(params.name) in checkpointsDict ? 'diamond' : 'circle';
            },
            symbolSize: (value, params) => {
              const DEFAULT_SIZE = 4;
              if (checkpointsDict === undefined) return DEFAULT_SIZE;
              return Number(params.name) in checkpointsDict ? 10 : DEFAULT_SIZE;
            },
            type: 'line',
          })),
          tooltip: {
            axisPointer: {
              label: {
                formatter: (params) => {
                  const value = params.value;
                  if (params.axisDimension === 'y' && typeof value === 'number') {
                    currentYAxis = value;
                    return value.toFixed(4).toString();
                  }
                  return value.toString();
                },
              },
            },
            formatter: formatterFunc,
          },
          xAxis: {
            boundaryGap: false,
            data: seriesData.x,
            max: xValueRange?.[1],
            min: xValueRange?.[0],
            name: xLabel,
            type: 'category',
          },
        };
        return option;
      }
    };

    const baseOption: EChartsOption = {
      dataZoom: [
        { realtime: true, show: true, type: 'slider' },
        { realtime: true, show: true, type: 'inside', zoomLock: true },
      ],
      legend: showLegend
        ? {
            data: series.map((serie) => serie.name ?? 'n/a'),
            left: '10%',
            padding: [8, 200, 0, 0],
            type: 'scroll',
          }
        : undefined,
      toolbox: {
        feature: {
          dataView: { readOnly: true },
          dataZoom: { yAxisIndex: false },
          saveAsImage: { excludeComponents: ['toolbox', 'dataZoom', 'legend'], name: 'line-chart' },
        },
      },
      tooltip: { axisPointer: { type: 'cross' }, confine, trigger: 'axis' },
      yAxis: {
        ...(scale === Scale.Log // write like this due to type issue
          ? { type: 'log' }
          : { axisLabel: { formatter: yValueFormatter }, type: 'value' }),
        minorSplitLine: { show: true },
        name: yLabel,
      },
    };

    const option: EChartsOption = _.merge(baseOption, generateOption());
    return option;
  }, [
    checkpointsDict,
    confine,
    scale,
    series,
    seriesData.x,
    seriesData.y,
    showLegend,
    xAxis,
    xLabel,
    xValueRange,
    yLabel,
    yValueFormatter,
  ]);

  return (
    <>
      <div>{title && <h5 className={css.chartTitle}>{title}</h5>}</div>
      {isLoading ? (
        <div>Loading</div>
      ) : (
        <div style={{ height }}>
          {series.every((serie) => (serie.data[xAxis]?.length ?? 0) === 0) ? (
            <div className={css.chartgridEmpty}>No data to plot.</div>
          ) : (
            <ReactECharts eventFunctions={EventFunctions} group={group} option={echartOption} />
          )}
        </div>
      )}
    </>
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
  group?: string;
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
    group?: string;
    scale: Scale;
    xAxis: XAxisDomain;
    handleError: ErrorHandler;
  }>
> = ({ columnIndex, rowIndex, style, data }) => {
  const { chartsProps, columnCount, scale, xAxis, handleError, group } = data;

  const cellIndex = rowIndex * columnCount + columnIndex;

  if (chartsProps === undefined || cellIndex >= chartsProps.length) return null;
  const chartProps = chartsProps[cellIndex];

  return (
    <div className={css.chartgridCell} key={`${rowIndex}, ${columnIndex}`} style={style}>
      <div className={css.chartgridCellCard}>
        <LineChart
          group={group}
          {...chartProps}
          handleError={handleError}
          scale={scale}
          xAxis={xAxis}
        />
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
    group,
    handleError,
  }: GroupProps) => {
    const { refCallback, size } = useResize();
    const height = size.height ?? 0;
    const width = size.width ?? 0;
    const columnCount = Math.max(1, Math.floor(width / 540));
    const chartsProps = Loadable.ensureLoadable(propChartsProps)
      .getOrElse([])
      .filter(
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
        const series = Loadable.ensureLoadable(chart.series).getOrElse([]);
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

    if (chartsProps.length === 0 && !isLoading)
      return <Message icon="warning" title="No data available." />;

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
                <FixedSizeGrid
                  columnCount={columnCount}
                  columnWidth={Math.floor(width / columnCount)}
                  height={height - 40}
                  itemData={{
                    chartsProps,
                    columnCount,
                    group,
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
              </>
            )}
          </Spinner>
        </div>
      </div>
    );
  },
);
