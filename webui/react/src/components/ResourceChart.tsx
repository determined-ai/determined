import { PlotData } from 'plotly.js/lib/core';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import Plotly, { Data } from 'Plotly';
import { getStateColor, lightTheme } from 'themes';
import { CommandState, CommonProps, Resource, ResourceState } from 'types';
import { clone } from 'utils/data';

import { generateAlphaNumeric } from '../utils/string';

interface Props extends CommonProps {
  title: string;
  resources?: Resource[];
}

export interface PlotInfo {
  data: Data[];
  layout: Partial<Plotly.Layout>;
  config: Partial<Plotly.Config>
}

type PlotArguments = [
  string,
  Partial<PlotData>[],
  Partial<Plotly.Layout>,
  Partial<Plotly.Config>,
];

type Tally = Record<ResourceState, number>;

const initialTally = Object.values(ResourceState).reduce((acc, key) => {
  acc[key as unknown as ResourceState] = 0;
  return acc;
}, {} as Tally);

const genPlotInfo = (title: string, resources: Resource[]): PlotInfo | null => {
  const tally = clone(initialTally) as Tally;

  resources.forEach(resource => {
    const state = resource.container ? resource.container.state : ResourceState.Free;
    tally[state] += 1;
  });

  const labels: string[] = [];
  const values: number[] = [];
  const colors: string[] = [];
  Object.entries(tally).forEach(([ rsState, rsValue ]) => {
    if (rsValue === 0) return;
    labels.push(rsState);
    values.push(rsValue);
    colors.push(getStateColor(rsState as CommandState));
  });

  const data: Data[] = [ {
    hole: 0.5,
    hoverinfo: 'skip',
    labels,
    marker: { colors },
    textinfo: 'label+value',
    type: 'pie',
    values,
  } ];

  if (values.filter(v => v !== 0).length === 0) {
    data[0] = {
      ...data[0],
      labels: [ `no ${title} available` ],
      marker: { colors: [ lightTheme.colors.monochrome[13] ] },
      textinfo: 'label',
      values: [ 1 ],
    };
  }

  return {
    config: { displayModeBar: false },
    data,
    layout: {
      annotations: [
        {
          font: { size: 20 },
          showarrow: false,
          text: `${title} (${resources.length})`,
          x: 0.5,
          y: 0.5,
        },
      ],
      hovermode: false,
      showlegend: false,
    },
  };
};

const SlotChart: React.FC<Props> = (props: Props) => {
  const chartRef = useRef<HTMLDivElement>(null);
  const [ id ] = useState(generateAlphaNumeric());
  const [ oldPlotInfo, setOldPlotInfo ] = useState<PlotInfo | null>(null);

  const plotInfo = useMemo(() => {
    const newPlotInfo = genPlotInfo(props.title, props.resources || []);
    if (JSON.stringify(newPlotInfo) === JSON.stringify(oldPlotInfo)) return oldPlotInfo;
    setOldPlotInfo(newPlotInfo);
    return newPlotInfo;
  }, [ oldPlotInfo, props.resources, props.title ]);

  const renderPlot = useCallback(async (
    elementId: string,
    pInfo: PlotInfo,
  ) => {
    const args: PlotArguments = [ elementId, pInfo.data, pInfo.layout, pInfo.config ];
    await Plotly.react.apply(null, args);
  }, [ id, plotInfo, props.resources ]);

  if (plotInfo === null) return <React.Fragment />;

  useEffect(() => {
    renderPlot(id, plotInfo);
  }, [ id, plotInfo, props.resources ]);

  return (
    // <div className={css.base}>
    <div>
      <div id={id} ref={chartRef} />
    </div>
  );
};

export default SlotChart;
