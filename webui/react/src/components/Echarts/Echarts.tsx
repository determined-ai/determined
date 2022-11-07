import { getInstanceByDom, init } from 'echarts';
import type { ECharts, EChartsOption, SetOptionOpts } from 'echarts';
import React, { useEffect, useRef } from 'react';
import type { CSSProperties } from 'react';

import useResize from 'hooks/useResize';

export interface Props {
  loading?: boolean;
  option: EChartsOption;
  rendererType?: 'canvas' | 'svg';
  settings?: SetOptionOpts;
  style?: CSSProperties;
  theme?: 'light' | 'dark';
}

const ReactECharts: React.FC<Props> = ({
  option,
  style,
  settings,
  loading,
  theme,
  rendererType,
}: Props) => {
  const chartRef = useRef<HTMLDivElement>(null);
  const resize = useResize(chartRef);

  useEffect(() => {
    // Initialize chart
    let chart: ECharts | undefined;

    if (chartRef.current) {
      chart = init(chartRef.current, theme, {
        height: resize.height,
        renderer: rendererType ?? 'canvas',
        useDirtyRect: false,
        width: resize.width,
      });
    }

    // Add chart resize listener
    const resizeChart = () => {
      chart?.resize();
    };
    document.addEventListener('resize', resizeChart);

    return () => {
      chart?.dispose();
      document.removeEventListener('resize', resizeChart);
    };
  }, [rendererType, resize.height, resize.width, theme]);

  useEffect(() => {
    // Update chart
    if (chartRef.current) {
      const chart = getInstanceByDom(chartRef.current);
      chart?.setOption(option, settings);
    }
  }, [option, settings, theme]);

  useEffect(() => {
    // Update chart
    if (chartRef.current) {
      const chart = getInstanceByDom(chartRef.current);
      loading === true ? chart?.showLoading() : chart?.hideLoading();
    }
  }, [loading, theme]);

  return (
    <div
      ref={chartRef}
      style={{
        height: '100vh',
        position: 'relative',
        width: '100%',
        ...style,
      }}
    />
  );
};

export default ReactECharts;
