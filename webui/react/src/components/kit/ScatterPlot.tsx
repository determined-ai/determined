import { EChartsOption } from 'echarts';
import { CallbackDataParams, TopLevelFormatterParams } from 'echarts/types/dist/shared';
import { useMemo } from 'react';

import { Scale } from 'components/kit/internal/types';
import ReactECharts from 'components/kit/ReactEchart';
import { ColorScale } from 'utils/color';

type PointType = [x: number, y: number, z: number, label: string];

export interface SerieData {
  data: PointType[];
}

interface Props {
  title?: string;
  series: SerieData;
  height?: number;
  group?: string;
  xLabel?: string;
  yLabel?: string;
  scale?: Scale;
  visualMapColorScale?: ColorScale[];
}

const ScatterPlot = ({
  title,
  series,
  height = 350,
  group,
  xLabel,
  yLabel,
  scale = Scale.Linear,
  visualMapColorScale,
}: Props): JSX.Element => {
  const colorScale = (visualMapColorScale ?? []).map((v) => v.scale);
  const colorRange = (visualMapColorScale ?? []).map((v) => v.color);

  const echartOption: EChartsOption = useMemo(() => {
    const option: EChartsOption = {
      series: {
        data: series.data,
        encode: { itemName: 3, x: 0, y: 1 },
        symbolSize: 10,
        type: 'scatter',
      },
      title: {
        left: 'center',
        text: title,
        textStyle: { fontSize: 10 },
      },
      tooltip: {
        axisPointer: { type: 'cross' },
        confine: true,
        formatter: (param: TopLevelFormatterParams) => {
          const p = param as CallbackDataParams;
          const data = p.data as PointType;

          return `
          <div style="font-size: 11px">
            <div>${xLabel}: ${data[0]}</div>
            <div>${yLabel}:  ${data[1]}</div>
            <div>Trial ID: ${p.name}</div>
          </div>
        `;
        },
        trigger: 'item',
      },
      visualMap: visualMapColorScale
        ? {
            calculable: true,
            dimension: 2,
            inRange: { color: colorRange },
            max: Math.max(...colorScale),
            min: Math.min(...colorScale),
            orient: 'vertical',
            right: 5,
            top: 'center',
          }
        : undefined,
      xAxis: [{ type: 'value' }],
      yAxis: [
        {
          minorSplitLine: { show: true },
          type: scale === Scale.Log ? 'log' : 'value',
        },
      ],
    };
    return option;
  }, [series.data, title, visualMapColorScale, colorRange, colorScale, scale, xLabel, yLabel]);

  return (
    <div style={{ height }}>
      <ReactECharts group={group} option={echartOption} />
    </div>
  );
};

export default ScatterPlot;
