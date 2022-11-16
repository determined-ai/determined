import type { EChartsOption, SetOptionOpts } from 'echarts';
import { BarChart, LineChart } from 'echarts/charts';
import {
  DatasetComponent,
  DataZoomComponent,
  GridComponent,
  LegendComponent,
  TitleComponent,
  ToolboxComponent,
  TooltipComponent,
  TransformComponent,
} from 'echarts/components';
import * as echarts from 'echarts/core';
import { LabelLayout, UniversalTransition } from 'echarts/features';
import { CanvasRenderer } from 'echarts/renderers';
import React, { useEffect, useRef } from 'react';
import type { CSSProperties } from 'react';

import useResize from 'hooks/useResize';
import useUI from 'shared/contexts/stores/UI';

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
  LabelLayout,
  UniversalTransition,
  CanvasRenderer,
]);

export interface Props {
  onClick?: (param: any) => void;
  option: EChartsOption;
  rendererType?: 'canvas' | 'svg';
  settings?: SetOptionOpts;
  style?: CSSProperties;
}

const ReactECharts: React.FC<Props> = ({
  option,
  style,
  settings = {},
  rendererType = 'canvas',
  onClick,
}: Props) => {
  const { ui } = useUI();
  const chartRef = useRef<HTMLDivElement>(null);
  const resize = useResize(chartRef);

  useEffect(() => {
    // Initialize chart
    const echart: echarts.ECharts | undefined = (() => {
      if (chartRef.current) {
        const chart = echarts.init(chartRef.current, ui.darkLight, {
          height: resize.height,
          renderer: rendererType,
          useDirtyRect: false,
          width: resize.width,
        });
        chart.on('click', onClick ?? (() => undefined));

        return chart;
      }
      return undefined;
    })();

    // Add chart resize listener
    echart?.resize();

    return () => {
      echart?.dispose();
    };
  }, [onClick, rendererType, resize.height, resize.width, ui.darkLight]);

  useEffect(() => {
    // Update chart
    if (chartRef.current) {
      const echart = echarts.getInstanceByDom(chartRef.current);
      echart?.setOption(option, { ...settings, notMerge: true }); // notMerge should be true
    }
  }, [option, settings]);

  return (
    <div
      ref={chartRef}
      style={{
        height: '100%',
        padding: '12px',
        position: 'relative',
        width: '100%',
        ...style,
      }}
    />
  );
};

export default ReactECharts;
