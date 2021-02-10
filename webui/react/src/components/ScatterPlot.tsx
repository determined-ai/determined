import React, { useEffect, useMemo, useRef, useState } from 'react';
import { throttle } from 'throttle-debounce';

import useResize, { DEFAULT_RESIZE_THROTTLE_TIME } from 'hooks/useResize';
import Plotly, { Layout, PlotData } from 'Plotly';
import themes, { defaultThemeId } from 'themes';
import { clone } from 'utils/data';
import { generateAlphaNumeric } from 'utils/string';

interface Props {
  height?: number;
  id?: string;
  padding?: number;
  title?: string;
  values?: number[];
  width?: number;
  x: number[];
  xLogScale?: boolean;
  y: number[];
  yLogScale?: boolean;
}

const plotlyLayout: Partial<Layout> = {
  autosize: false,
  height: 350,
  margin: { b: 32, l: 32, r: 32, t: 48 },
  paper_bgcolor: 'transparent',
  plot_bgcolor: themes[defaultThemeId].colors.monochrome[17],
  xaxis: { automargin: true },
  yaxis: { automargin: true },
};
const plotlyConfig: Partial<Plotly.Config> = {
  displayModeBar: false,
  responsive: true,
};

const ScatterPlot: React.FC<Props> = ({
  title,
  padding = 0,
  x,
  xLogScale,
  y,
  yLogScale,
  ...props
}: Props) => {
  const chartRef = useRef<HTMLDivElement>(null);
  const resize = useResize(chartRef);
  const [ id ] = useState(props.id ? props.id : generateAlphaNumeric());

  const chartData: Partial<PlotData> = useMemo(() => {
    return {
      marker: { color: themes[defaultThemeId].colors.action.normal },
      mode: 'markers',
      x,
      y,
    };
  }, [ x, y ]);

  const chartLayout: Partial<Layout> = useMemo(() => {
    const layout = clone(plotlyLayout);
    if (title) {
      layout.title = {
        font: { family: themes[defaultThemeId].font.family, size: 13 },
        text: title,
      };
    }
    if (xLogScale) layout.xaxis.type = 'log';
    if (yLogScale) layout.yaxis.type = 'log';
    return layout;
  }, [ title, xLogScale, yLogScale ]);

  useEffect(() => {
    const ref = chartRef.current;
    if (!ref) return;

    Plotly.react(ref, [ chartData ], chartLayout, plotlyConfig);

    return () => {
      if (ref) Plotly.purge(ref);
    };
  }, [ chartData, chartLayout, padding, title ]);

  // Resize the chart when resize events happen.
  useEffect(() => {
    const throttleResize = throttle(DEFAULT_RESIZE_THROTTLE_TIME, () => {
      if (!chartRef.current) return;
      const rect = chartRef.current.getBoundingClientRect();
      const layout = { ...chartLayout, height: rect.height, width: rect.width };
      Plotly.react(chartRef.current, [ chartData ], layout, plotlyConfig);
    });

    throttleResize();
  }, [ chartData, chartLayout, resize ]);

  return <div id={id} ref={chartRef} />;
};

export default ScatterPlot;
