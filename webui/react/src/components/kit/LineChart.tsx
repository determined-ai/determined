import dayjs from 'dayjs';
import { ECElementEvent, EChartsOption } from 'echarts';
import { CallbackDataParams, TopLevelFormatterParams } from 'echarts/types/dist/shared';
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

import css from './LineChart.module.scss';

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
  const isLoading = Loadable.isLoadable(propSeries) && Loadable.isNotLoaded(propSeries);

  const EventFunctions: EchartsEventFunction[] = useMemo(() => {
    return [
      {
        eventName: 'click',
        handler: (params) => {
          onClickPoint?.(Number(params.seriesId), params.data);
        },
      },
    ];
  }, [onClickPoint]);

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
        .filter((d: CallbackDataParams) => d.seriesName)
        .map((d: CallbackDataParams): SeriesType => {
          return {
            marker: d.marker,
            seriesName: d.seriesName ?? '',
            value: (d.value as number[])[1],
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
                ${yValueFormatter ? yValueFormatter(d.value) : d.value.toFixed(4)}
              </div>`;
            })
            .join('')}
        </div>
      `;
      return tooltip;
    };

    const option: EChartsOption = {
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
      series: series.map((serie) => ({
        connectNulls: true,
        data: (() => {
          if (xAxis === XAxisDomain.Time) {
            const set = new Set();
            const arr: [x: Date, y: number][] = [];
            for (const d of serie.data[xAxis] ?? []) {
              const xValue = d[0];
              const yValue = d[1];
              if (set.has(xValue)) {
                continue;
              }
              set.add(xValue);
              arr.push([adjustLocalTimeToUTC(xValue), yValue]);
            }
            return arr;
          } else {
            return serie.data[xAxis];
          }
        })(),
        emphasis: { focus: 'series' },
        id: serie.key,
        itemStyle: { color: serie.color },
        name: serie.name ?? yLabel,
        symbol: (value) => {
          if (checkpointsDict === undefined) return 'circle';
          return value?.[0] in checkpointsDict ? 'diamond' : 'circle';
        },
        symbolSize: (value) => {
          const DEFAULT_SIZE = 4;
          if (checkpointsDict === undefined) return DEFAULT_SIZE;
          return value?.[0] in checkpointsDict ? 10 : DEFAULT_SIZE;
        },
        type: 'line',
      })),
      toolbox: {
        feature: {
          dataView: { readOnly: true },
          dataZoom: { yAxisIndex: false },
          saveAsImage: { excludeComponents: ['toolbox', 'dataZoom', 'legend'], name: 'line-chart' },
        },
      },
      tooltip: {
        axisPointer: {
          label: {
            formatter:
              xAxis === XAxisDomain.Time
                ? undefined
                : (params) => {
                    const value = params.value;
                    if (params.axisDimension === 'y' && typeof value === 'number') {
                      currentYAxis = value;
                      return value.toFixed(4).toString();
                    }
                    return value.toString();
                  },
          },
          type: 'cross',
        },
        confine,
        formatter: formatterFunc,
        trigger: 'axis',
      },
      xAxis: {
        ...(xAxis === XAxisDomain.Time
          ? {
              max: xValueRange ? adjustLocalTimeToUTC(xValueRange[1]) : undefined,
              min: xValueRange ? adjustLocalTimeToUTC(xValueRange[0]) : undefined,
              type: 'time',
            }
          : {
              max: xValueRange?.[1],
              min: xValueRange?.[0],
              type: 'value',
            }),
        boundaryGap: [0, 0],
        name: xLabel,
      },
      yAxis: {
        ...(scale === Scale.Log // write like this due to type issue
          ? { type: 'log' }
          : { axisLabel: { formatter: yValueFormatter }, type: 'value' }),
        minorSplitLine: { show: true },
        name: yLabel,
      },
    };

    return option;
  }, [
    checkpointsDict,
    confine,
    scale,
    series,
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
    xValueRange?: Partial<Record<XAxisDomain, [min: number, max: number]>>;
    columnCount: number;
    group?: string;
    scale: Scale;
    xAxis: XAxisDomain;
    handleError: ErrorHandler;
  }>
> = ({ columnIndex, rowIndex, style, data }) => {
  const { chartsProps, columnCount, scale, xAxis, handleError, group, xValueRange } = data;

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
          xValueRange={xValueRange?.[xAxis]}
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
      const xOpts = new Set<XAxisDomain>();
      chartsProps.forEach((chart) => {
        const series = Loadable.ensureLoadable(chart.series).getOrElse([]);
        series.forEach((serie) => {
          Object.entries(serie.data).forEach(([xAxisOption, dataPoints]) => {
            if (dataPoints.length > 0) {
              xOpts.add(xAxisOption as XAxisDomain);
            }
          });
        });
      });
      return Array.from(xOpts).sort();
    }, [chartsProps]);

    const minMaxRange: Partial<Record<XAxisDomain, [min: number, max: number]>> = useMemo(() => {
      const dic: Partial<Record<XAxisDomain, [min: number, max: number]>> = {};

      chartsProps.map((p) => {
        const series = Loadable.ensureLoadable(p.series).getOrElse([]);
        series.map((serie) => {
          for (const xDomain of xAxisOptions) {
            let xMin = serie.data[xDomain]?.at(0)?.[0];
            let xMax = serie.data[xDomain]?.at(-1)?.[0];
            if (xMin === undefined || xMax === undefined) {
              continue;
            }
            for (const [x] of serie.data[xDomain] ?? []) {
              xMin = Math.min(x, xMin);
              xMax = Math.max(x, xMax);
            }

            dic[xDomain] = [
              Math.min(dic[xDomain]?.[0] ?? xMin, xMin),
              Math.max(dic[xDomain]?.[1] ?? xMax, xMax),
            ];
          }
        });
      });

      return dic;
    }, [chartsProps, xAxisOptions]);

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
                    xValueRange: minMaxRange,
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
