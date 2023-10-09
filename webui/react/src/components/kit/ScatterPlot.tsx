import { EChartsOption } from 'echarts';
import { CallbackDataParams, TopLevelFormatterParams } from 'echarts/types/dist/shared';
import { useMemo } from 'react';

import { Scale } from 'components/kit/internal/types';
import ReactECharts from 'components/kit/ReactEchart';

type PointType = [x: number, y: number, label: string];

export interface SerieData {
  data: PointType[];
}

interface Props {
  title?: string;
  series: SerieData;
  height?: number;
  group?: string;
  tooltipFormatter?: (
    x: number,
    y: number,
    xLabel: string,
    yLabel: string,
    label: string,
  ) => string;
  xLabel?: string;
  yLabel?: string;
  scale?: Scale;
}

const ScatterPlot = ({
  title,
  series,
  height = 350,
  group,
  tooltipFormatter,
  xLabel,
  yLabel,
  scale = Scale.Linear,
}: Props): JSX.Element => {
  const echartOption: EChartsOption = useMemo(() => {
    const option: EChartsOption = {
      series: {
        data: series.data,
        encode: {
          itemName: 2,
          x: 0,
          y: 1,
        },
        symbolSize: 10,
        type: 'scatter',
      },
      title: {
        left: 'center',
        text: title,
        textStyle: {
          fontSize: 10,
        },
      },
      tooltip: {
        axisPointer: {
          type: 'cross',
        },
        confine: true,
        formatter: tooltipFormatter
          ? (param: TopLevelFormatterParams) => {
              const p = param as CallbackDataParams;
              const data = p.data as PointType;
              return tooltipFormatter(data[0], data[1], xLabel ?? '', yLabel ?? '', p.name);
            }
          : undefined,
        trigger: 'item',
      },
      xAxis: [{ type: 'value' }],
      yAxis: [
        {
          minorSplitLine: { show: true },
          type: scale === Scale.Log ? 'log' : 'value',
        },
      ],
    };
    return option;
  }, [series.data, title, tooltipFormatter, xLabel, yLabel, scale]);

  return (
    <div style={{ height }}>
      <ReactECharts group={group} option={echartOption} />
    </div>
  );
};

export default ScatterPlot;
