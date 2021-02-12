import React, { useEffect, useMemo, useRef, useState } from 'react';
import { throttle } from 'throttle-debounce';

import useResize, { DEFAULT_RESIZE_THROTTLE_TIME } from 'hooks/useResize';
import Plotly, { Layout, PlotData, PlotMarker } from 'Plotly';
import themes, { defaultThemeId } from 'themes';
import { getNumericRange } from 'utils/chart';
import { hex2rgb, rgba2str, rgbaFromGradient } from 'utils/color';
import { clone } from 'utils/data';
import { roundToPrecision } from 'utils/number';
import { generateAlphaNumeric } from 'utils/string';

interface Props {
  height?: number;
  id?: string;
  padding?: number;
  title?: string;
  valueLabel?: string;
  values?: number[];
  width?: number;
  x: number[];
  xLabel?: string;
  xLogScale?: boolean;
  y: number[];
  yLabel?: string;
  yLogScale?: boolean;
}

const plotlyLayout: Partial<Layout> = {
  autosize: false,
  height: 350,
  hovermode: 'closest',
  margin: { b: 32, l: 32, r: 32, t: 48 },
  paper_bgcolor: 'transparent',
  plot_bgcolor: themes[defaultThemeId].colors.monochrome[17],
  title: { font: { family: themes[defaultThemeId].font.family, size: 13 } },
  xaxis: { automargin: true },
  yaxis: { automargin: true },
};
const plotlyConfig: Partial<Plotly.Config> = {
  displayModeBar: false,
  responsive: true,
};

const ScatterPlot: React.FC<Props> = ({
  title,
  height,
  padding = 0,
  valueLabel,
  values,
  width,
  x,
  xLabel,
  xLogScale,
  y,
  yLabel,
  yLogScale,
  ...props
}: Props) => {
  const chartRef = useRef<HTMLDivElement>(null);
  const resize = useResize(chartRef);
  const [ id ] = useState(props.id ? props.id : generateAlphaNumeric());

  const valueRange = useMemo(() => getNumericRange(values || [], false), [ values ]);

  const chartData = useMemo(() => {
    const hovertemplate = [
      `${xLabel || 'x'}: %{x:.6f}`,
      `${yLabel || 'y'}: %{y:.6f}`,
    ];

    if (values) hovertemplate.push(`${valueLabel || 'value'}: %{text}`);

    const trace: Partial<PlotData> = {
      hovertemplate: `${hovertemplate.join('<br>')}<extra></extra>`,
      marker: { color: themes[defaultThemeId].colors.action.normal },
      mode: 'markers',
      x,
      y,
    };

    if (values && valueRange) {
      const rgb0 = hex2rgb(themes[defaultThemeId].colors.danger.light);
      const rgb1 = hex2rgb(themes[defaultThemeId].colors.action.normal);

      /*
       * There is an issue with plotly's typing for `marker.color`.
       * It also takes in type of `string[]` but currently it's typed as `string` only.
       * So we cast it to `unknown` then to a `string` as a workaround.
       */
      (trace.marker as Partial<PlotMarker>).color = values.map(value => {
        const distance = (value - valueRange[0]) / (valueRange[1] - valueRange[0]);
        const rgb = rgbaFromGradient(rgb0, rgb1, distance);
        return rgba2str(rgb);
      }) as unknown as string;

      trace.text = values.map(value => roundToPrecision(value).toString());
    }
    return trace;
  }, [ valueLabel, values, valueRange, x, xLabel, y, yLabel ]);

  const chartLayout: Partial<Layout> = useMemo(() => {
    const layout = clone(plotlyLayout);
    if (title) {
      layout.title.text = title;
    } else if (xLabel && yLabel) {
      layout.title.text = `${xLabel} vs ${yLabel}`;
    }
    if (xLogScale) layout.xaxis.type = 'log';
    if (yLogScale) layout.yaxis.type = 'log';
    if (height) layout.height = height;
    if (width) layout.width = width;
    return layout;
  }, [ height, title, width, xLabel, xLogScale, yLabel, yLogScale ]);

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
