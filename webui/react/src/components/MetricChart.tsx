import { Select, Space } from 'antd';
import { SelectValue } from 'antd/es/select';
import Plotly, { PlotData, PlotlyHTMLElement, PlotRelayoutEvent } from 'plotly.js/lib/core';
import React, { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react';

import useResize from 'hooks/useResize';
import { clone } from 'utils/data';
import { capitalize, generateAlphaNumeric } from 'utils/string';

import css from './MetricChart.module.scss';
import Section from './Section';
import SelectFilter from './SelectFilter';

const { Option } = Select;

interface Props {
  data: Partial<PlotData>[];
  id?: string;
  options?: React.ReactNode;
  title: string;
  xLabel: string;
  yLabel: string;
}

interface Range {
  xaxis: [ number, number ];
  yaxis: [ number, number ];
}

enum Scale {
  Linear = 'linear',
  Log = 'log',
}

type PlotArguments = [
  string,
  Partial<PlotData>[],
  Partial<Plotly.Layout>,
  Partial<Plotly.Config>,
];

const defaultLayout: Partial<Plotly.Layout> = {
  height: 400,
  hovermode: 'x unified',
  legend: { xanchor: 'right' },
  margin: { b: 50, l: 50, pad: 6, r: 10, t: 10 },
  showlegend: true,
  xaxis: { hoverformat: '' },
  yaxis: { type: Scale.Linear },
};

const defaultConfig: Partial<Plotly.Config> = {
  displayModeBar: false,
  doubleClick: false,
  responsive: true,
};

const PADDING_PERCENT = 0.1;

const MetricChart: React.FC<Props> = (props: Props) => {
  const chartRef = useRef<HTMLDivElement>(null);
  const [ id ] = useState(props.id ? props.id : generateAlphaNumeric());
  const [ scale, setScale ] = useState<Scale>(Scale.Linear);
  const [ range, setRange ] = useState<Range>();
  const [ maxRange, setMaxRange ] = useState<Range>();
  const [ isRendered, setIsRendered ] = useState(false);
  const [ isZoomed, setIsZoomed ] = useState(false);
  const resize = useResize(chartRef);

  const handleDoubleClick = useCallback(() => {
    setRange(clone(maxRange));
    setIsZoomed(false);
  }, [ maxRange ]);

  const handleRelayout = useCallback((event: PlotRelayoutEvent) => {
    // Brute force check to keep Typescript happy.
    if (event['xaxis.range[0]'] != null && event['xaxis.range[1]'] != null &&
        event['yaxis.range[0]'] != null && event['yaxis.range[1]'] != null) {
      /*
      * Preserve the zoom and pan range. When new data comes in
      * the re-rendering of the plot will render the same zoom level.
      */
      setRange({
        xaxis: [ event['xaxis.range[0]'], event['xaxis.range[1]'] ],
        yaxis: [ event['yaxis.range[0]'], event['yaxis.range[1]'] ],
      });
      setIsZoomed(true);
    }
  }, []);

  const renderPlot = useCallback(async (
    elementId: string,
    plotData: Partial<PlotData>[],
    plotLayout?: Partial<Plotly.Layout>,
    plotConfig?: Partial<Plotly.Config>,
  ) => {
    const layout = plotLayout || defaultLayout;
    const config = plotConfig || defaultConfig;
    const args: PlotArguments = [ elementId, plotData, layout, config ];

    if (isRendered) {
      await Plotly.react.apply(null, args);
    } else {
      const chart: PlotlyHTMLElement = await Plotly.newPlot.apply(null, args);
      chart.on('plotly_doubleclick', handleDoubleClick);
      chart.on('plotly_relayout', handleRelayout);
      chart.on('plotly_legendclick', () => false);
      setIsRendered(true);
    }
  }, [ handleDoubleClick, handleRelayout, isRendered ]);

  const handleScaleSelect = useCallback((newValue: SelectValue) => setScale(newValue as Scale), []);

  useEffect(() => {
    let xMin = Number.POSITIVE_INFINITY;
    let xMax = Number.NEGATIVE_INFINITY;
    let yMin = Number.POSITIVE_INFINITY;
    let yMax = Number.NEGATIVE_INFINITY;

    // Figure out the ranges based on the provided data.
    props.data.forEach(data => {
      (data.y as number[] || []).forEach(y => {
        if (y < yMin) yMin = y;
        if (y > yMax) yMax = y;
      });
      (data.x as number[] || []).forEach(x => {
        if (x < xMin) xMin = x;
        if (x > xMax) xMax = x;
      });
    });

    // Add padding to the ranges.
    const [ xPad, yPad ] = [ (xMax - xMin) * PADDING_PERCENT, (yMax - yMin) * PADDING_PERCENT ];
    const [ xMinEdge, xMaxEdge ] = [ xMin - xPad, xMax + xPad ];
    const [ yMinEdge, yMaxEdge ] = [ yMin - yPad, yMax + yPad ];
    const newMaxRange: Range = {
      xaxis: [ Math.max(0, xMinEdge), xMaxEdge ],
      yaxis: [ yMinEdge < 0 ? yMinEdge : Math.max(0, yMinEdge), yMaxEdge ],
    };

    setMaxRange(newMaxRange);

    if (!isZoomed) setRange(clone(newMaxRange));
  }, [ isZoomed, props.data ]);

  useEffect(() => {
    const layout = clone(defaultLayout);
    const maxRangeCopy = clone(maxRange);
    layout.xaxis.title = props.xLabel;
    layout.yaxis.title = props.yLabel;
    layout.yaxis.type = scale;
    layout.xaxis.range = range ? range.xaxis : (maxRangeCopy ? maxRangeCopy.xaxis : undefined);
    layout.yaxis.range = range ? range.yaxis : (maxRangeCopy ? maxRangeCopy.yaxis : undefined);

    renderPlot(id, props.data || [], layout);
  }, [ id, maxRange, props.data, props.xLabel, props.yLabel, range, renderPlot, scale ]);

  /*
   * Dynamcially swapping out chart handlers is needed otherwise
   * referenced data such as `maxRange` within the handlers will be stale.
   */
  useEffect(() => {
    const chart = chartRef.current as unknown as PlotlyHTMLElement;
    if (!chart || !chart.removeAllListeners) return;

    chart.removeAllListeners('plotly_doubleclick');
    chart.removeAllListeners('plotly_relayout');
    chart.on('plotly_doubleclick', handleDoubleClick);
    chart.on('plotly_relayout', handleRelayout);

    return () => {
      chart.removeAllListeners('plotly_legendclick');
      chart.removeAllListeners('plotly_doubleclick');
      chart.removeAllListeners('plotly_relayout');
    };
  }, [ handleDoubleClick, handleRelayout, id ]);

  useLayoutEffect(() => {
    if (!chartRef.current) return;
    Plotly.Plots.resize(chartRef.current);
  }, [ resize ]);

  const chartOptions = (
    <Space size="small">
      {props.options}
      <SelectFilter
        enableSearchFilter={false}
        label="Scale"
        showSearch={false}
        value={scale}
        onSelect={handleScaleSelect}>
        {Object.values(Scale).map(scale => (
          <Option key={scale} value={scale}>{capitalize(scale)}</Option>
        ))}
      </SelectFilter>
    </Space>
  );

  return (
    <Section bodyBorder maxHeight options={chartOptions} title={props.title}>
      <div className={css.base}>
        <div id={id} ref={chartRef} />
      </div>
    </Section>
  );
};

export default MetricChart;
