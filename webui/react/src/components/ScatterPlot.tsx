import React, { useEffect, useMemo, useRef, useState } from 'react';
import { throttle } from 'throttle-debounce';

import useResize, { DEFAULT_RESIZE_THROTTLE_TIME } from 'hooks/useResize';
import Plotly, { Layout, PlotData } from 'Plotly';
import { clone } from 'utils/data';
import { generateAlphaNumeric } from 'utils/string';

interface Props {
  data: Record<'x' | 'y' | 'values', number[]>;
  height?: number;
  id?: string;
  padding?: number;
  title?: string;
  width?: number;
}

const plotlyLayout: Partial<Layout> = {
  autosize: false,
  height: 250,
  margin: { b: 0, l: 0, r: 0, t: 0 },
  paper_bgcolor: 'transparent',
  width: 300,
  yaxis: { automargin: true },
};
const plotlyConfig: Partial<Plotly.Config> = {
  displayModeBar: false,
  responsive: true,
};

const ScatterPlot: React.FC<Props> = ({
  title,
  data,
  padding = 10,
  ...props
}: Props) => {
  const chartRef = useRef<HTMLDivElement>(null);
  const resize = useResize(chartRef);
  const [ id ] = useState(props.id ? props.id : generateAlphaNumeric());

  const chartData: Partial<PlotData> = useMemo(() => {
    return {
      mode: 'markers',
      x: data.x,
      y: data.y,
    };
  }, [ data ]);

  useEffect(() => {
    const ref = chartRef.current;
    if (!ref) return;

    const layout = clone(plotlyLayout);

    if (title) layout.title = title;
    if (padding) layout.margin = { b: padding, l: padding, r: padding, t: padding };

    console.log('layout', layout);
    Plotly.react(ref, [ chartData ], layout, plotlyConfig);

    return () => {
      if (ref) Plotly.purge(ref);
    };
    /* eslint-disable-next-line react-hooks/exhaustive-deps */
  }, [ chartData ]);

  // Resize the chart when resize events happen.
  useEffect(() => {
    const throttleResize = throttle(DEFAULT_RESIZE_THROTTLE_TIME, () => {
      if (!chartRef.current) return;
      const rect = chartRef.current.getBoundingClientRect();
      const layout = { ...plotlyLayout, width: rect.width };
      Plotly.react(chartRef.current, [ chartData ], layout, plotlyConfig);
    });

    throttleResize();
  }, [ chartData, resize ]);

  return <div id={id} ref={chartRef} />;
};

export default ScatterPlot;
