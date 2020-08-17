import { Select, Space } from 'antd';
import { SelectValue } from 'antd/es/select';
import Plotly, { PlotData, PlotlyHTMLElement, PlotRelayoutEvent } from 'plotly.js/lib/core';
import React, { useCallback, useEffect, useState } from 'react';

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
  xaxis: [ number | undefined, number | undefined ];
  yaxis: [ number | undefined, number | undefined ];
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
  margin: { b: 50, l: 50, pad: 6, r: 10, t: 10 },
  xaxis: {
    hoverformat: '',
    rangemode: 'tozero',
  },
  yaxis: { type: Scale.Linear },
};

const defaultConfig: Partial<Plotly.Config> = {
  displayModeBar: false,
  responsive: true,
};

const MetricChart: React.FC<Props> = (props: Props) => {
  const id = props.id ? props.id : generateAlphaNumeric();
  const [ scale, setScale ] = useState<Scale>(Scale.Linear);
  const [ range, setRange ] = useState<Range>({
    xaxis: [ undefined, undefined ],
    yaxis: [ undefined, undefined ],
  });
  const [ isRendered, setIsRendered ] = useState(false);

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
      chart.on('plotly_relayout', (event: PlotRelayoutEvent) => {
        /*
         * Preserve the zoom and pan range. When new data comes in
         * the re-rendering of the plot will render the same zoom level.
         */
        setRange({
          xaxis: [ event['xaxis.range[0]'], event['xaxis.range[1]'] ],
          yaxis: [ event['yaxis.range[0]'], event['yaxis.range[1]'] ],
        });
      });
      setIsRendered(true);
    }
  }, [ isRendered ]);

  useEffect(() => {
    const layout = clone(defaultLayout);
    layout.xaxis.title = props.xLabel;
    layout.yaxis.title = props.yLabel;
    layout.yaxis.type = scale;

    if (range.xaxis[0] != null && range.xaxis[1] != null &&
        range.yaxis[0] != null && range.yaxis[1] != null) {
      layout.xaxis.range = range.xaxis;
      layout.yaxis.range = range.yaxis;
    }

    renderPlot(id, props.data || [], layout);
  }, [ id, props.data, props.xLabel, props.yLabel, renderPlot, range, scale ]);

  const handleScaleSelect = useCallback((newValue: SelectValue) => {
    setScale(newValue as Scale);
  }, []);

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
        <div id={id} />
      </div>
    </Section>
  );
};

export default MetricChart;
