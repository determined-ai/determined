import { getInstanceByDom, init } from 'echarts';
import type { ECharts, EChartsOption, SetOptionOpts } from 'echarts';
import React, { useEffect, useRef } from 'react';
import type { CSSProperties } from 'react';

import useResize from 'hooks/useResize';

export interface Props {
  loading?: boolean;
  onClick?: (param) => void;
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
  rendererType = 'canvas',
  onClick,
}: Props) => {
  const chartRef = useRef<HTMLDivElement>(null);
  const resize = useResize(chartRef);

  useEffect(() => {
    // Initialize chart
    const echart: ECharts | undefined = (() => {
      if (chartRef.current) {
        const chart = init(chartRef.current, theme, {
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
    const resizeChart = () => {
      echart?.resize();
    };
    document.addEventListener('resize', resizeChart);

    return () => {
      echart?.dispose();
      document.removeEventListener('resize', resizeChart);
    };
  }, [onClick, rendererType, resize.height, resize.width, theme]);

  useEffect(() => {
    // Update chart
    if (chartRef.current) {
      const echart = getInstanceByDom(chartRef.current);
      echart?.setOption(option, settings);
    }
  }, [option, settings, theme]);

  useEffect(() => {
    // Update chart
    if (chartRef.current) {
      const echart = getInstanceByDom(chartRef.current);
      loading === true ? echart?.showLoading() : echart?.hideLoading();
    }
  }, [loading, theme]);

  return (
    <div
      ref={chartRef}
      style={{
        height: '50vh',
        padding: '12px',
        position: 'relative',
        width: '100%',
        ...style,
      }}
    />
  );
};

export default ReactECharts;
