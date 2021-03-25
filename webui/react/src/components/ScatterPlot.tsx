import React, { useEffect, useMemo, useRef, useState } from 'react';
import { throttle } from 'throttle-debounce';

import useResize, { DEFAULT_RESIZE_THROTTLE_TIME } from 'hooks/useResize';
import Plotly, { Layout, PlotData, PlotMarker } from 'Plotly';
import themes, { defaultThemeId } from 'themes';
import { getNumericRange } from 'utils/chart';
import { ColorScale, rgba2str, rgbaFromGradient, str2rgba } from 'utils/color';
import { clone, isString } from 'utils/data';
import { roundToPrecision } from 'utils/number';
import { generateAlphaNumeric, truncate } from 'utils/string';

interface Props {
  colorScale?: ColorScale[];
  height?: number;
  id?: string;
  title?: string;
  valueLabel?: string;
  values?: number[];
  width?: number;
  x: (number | string)[];
  xLabel?: string;
  xLogScale?: boolean;
  y: (number| string)[];
  yLabel?: string;
  yLogScale?: boolean;
}

const MAX_TITLE_LABEL_LENGTH = 20;
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
  colorScale,
  height,
  title,
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

  const { xIsString, yIsString } = useMemo(() => {
    return {
      xIsString: x.length !== 0 && isString(x[0]),
      yIsString: y.length !== 0 && isString(y[0]),
    };
  }, [ x, y ]);

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

    if (values && valueRange && colorScale) {
      const rgb0 = str2rgba(colorScale[0].color);
      const rgb1 = str2rgba(colorScale[1].color);

      /*
       * There is an issue with plotly's typing for `marker.color`.
       * It also takes in type of `string[]` but currently it's typed as `string` only.
       * So we cast it to `unknown` then to a `string` as a workaround.
       */
      (trace.marker as Partial<PlotMarker>).color = values.map(value => {
        const distance = valueRange[0] === valueRange[1] ?
          0.5 : (value - valueRange[0]) / (valueRange[1] - valueRange[0]);
        const rgb = rgbaFromGradient(rgb0, rgb1, distance);
        return rgba2str(rgb);
      }) as unknown as string;

      trace.text = values.map(value => roundToPrecision(value).toString());
    }
    return trace;
  }, [ colorScale, valueLabel, values, valueRange, x, xLabel, y, yLabel ]);

  const chartLayout: Partial<Layout> = useMemo(() => {
    const layout = clone(plotlyLayout);
    if (title) {
      layout.title.text = title;
    } else if (xLabel && yLabel) {
      const xLabelTitle = truncate(xLabel, MAX_TITLE_LABEL_LENGTH);
      const yLabelTitle = truncate(yLabel, MAX_TITLE_LABEL_LENGTH);
      layout.title.text = `${yLabelTitle} (y) vs ${xLabelTitle} (x)`;
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

    Plotly.react(ref, [], plotlyLayout, plotlyConfig);

    return () => {
      Plotly.purge(ref);
    };
  }, []);

  // Redraw the chart when we detect changes in the data or a resize event.
  useEffect(() => {
    const throttleResize = throttle(DEFAULT_RESIZE_THROTTLE_TIME, () => {
      if (!chartRef.current || resize.width === 0 || resize.height === 0) return;
      const rect = chartRef.current.getBoundingClientRect();
      const layout = { ...chartLayout, height: rect.height, width: rect.width };
      if (xIsString) layout.xaxis = { ...layout.xaxis, tickangle: 90 };
      Plotly.react(chartRef.current, [ chartData ], layout, plotlyConfig);
    });

    throttleResize();
  }, [ chartData, chartLayout, resize, xIsString, yIsString ]);

  return <div id={id} ref={chartRef} />;
};

export default ScatterPlot;
