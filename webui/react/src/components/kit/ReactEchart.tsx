import type { EChartsOption, SetOptionOpts } from 'echarts';
import { BarChart, LineChart, ScatterChart } from 'echarts/charts';
import {
  DatasetComponent,
  DataZoomComponent,
  GridComponent,
  LegendComponent,
  TitleComponent,
  ToolboxComponent,
  TooltipComponent,
  TransformComponent,
  VisualMapComponent,
} from 'echarts/components';
import * as echarts from 'echarts/core';
import { LabelLayout, UniversalTransition } from 'echarts/features';
import { CanvasRenderer } from 'echarts/renderers';
import React, { useEffect, useRef } from 'react';

import useUI from 'components/kit/Theme';
import useResize from 'hooks/useResize';

echarts.use([
  TitleComponent,
  TooltipComponent,
  ToolboxComponent,
  DataZoomComponent,
  LegendComponent,
  GridComponent,
  DatasetComponent,
  TransformComponent,
  BarChart,
  LineChart,
  ScatterChart,
  LabelLayout,
  UniversalTransition,
  CanvasRenderer,
  VisualMapComponent,
]);

export interface EchartsEventFunction {
  eventName: echarts.ElementEvent['type'];
  query?: echarts.ElementEvent;
  handler: (param: echarts.ECElementEvent) => void;
}

export interface Props {
  option: EChartsOption;
  eventFunctions?: EchartsEventFunction[];
  settings?: SetOptionOpts;
  group?: string;
}

const ReactECharts: React.FC<Props> = ({ option, settings = {}, group, eventFunctions }: Props) => {
  const { ui } = useUI();
  const chartRef = useRef<HTMLDivElement>(null);
  const resize = useResize(chartRef);

  useEffect(() => {
    // Initialize chart
    const echart: echarts.ECharts | undefined = (() => {
      if (chartRef.current) {
        const chart = echarts.init(chartRef.current, ui.darkLight, {
          renderer: 'canvas',
          useDirtyRect: false,
        });
        if (group) {
          chart.group = group;
          echarts.connect(group);
        }
        for (const eventFunc of eventFunctions ?? []) {
          chart.on(eventFunc.eventName, eventFunc.query ?? '', eventFunc.handler);
        }

        return chart;
      }
      return undefined;
    })();

    return () => {
      echart?.dispose();
    };
  }, [eventFunctions, group, ui.darkLight]);

  useEffect(() => {
    if (chartRef.current) {
      const echart = echarts.getInstanceByDom(chartRef.current);
      echart?.resize({ height: resize.height, width: resize.width });
    }
  }, [resize.height, resize.width]);

  useEffect(() => {
    if (chartRef.current) {
      const echart = echarts.getInstanceByDom(chartRef.current);
      echart?.setOption(
        { ...option, animation: false },
        {
          ...settings,
          notMerge: false,
          replaceMerge: ['xAxis', 'yAxis', 'series'],
        },
      );
    }
  }, [option, settings]);

  return (
    <div
      ref={chartRef}
      style={{
        height: '100%',
        position: 'relative',
        width: '100%',
      }}
    />
  );
};

export default ReactECharts;
