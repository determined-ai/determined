import Plotly, { Data, Layout } from 'plotly.js-basic-dist';
import React, { useMemo, useState } from 'react';
import createPlotlyComponent from 'react-plotly.js/factory';

import { getStateColor, lightTheme } from 'themes';
import { CommandState, CommonProps, Resource, ResourceState } from 'types';
import { clone } from 'utils/data';

const Plot = createPlotlyComponent(Plotly);
interface Props extends CommonProps {
  title: string;
  resources?: Resource[];
}

export interface PlotInfo {
  data: Data[];
  layout: Partial<Layout>;
}

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
    if (rsState === ResourceState.Free) {
      colors.push(lightTheme.colors.states.inactive);
    } else {
      colors.push(getStateColor(rsState as CommandState));
    }
  });

  const data: Data[] = [ {
    hole: 0.5,
    hoverinfo: 'skip',
    labels,
    marker: {
      colors,
    },
    textinfo: 'label+value',
    type: 'pie',
    values,
  } ];

  if (values.filter(v => v !== 0).length === 0) {
    data[0] = {
      ...data[0],
      labels: [ `no ${title} available` ],
      marker: {
        colors: [ lightTheme.colors.monochrome[13] ],
      },
      textinfo: 'label',
      values: [ 1 ],
    };
  }

  return { data,
    layout: {
      annotations: [
        {
          font: {
            size: 20,
          },
          showarrow: false,
          text: `${title} (${resources.length})`,
          x: 0.5,
          y: 0.5,
        },
      ],
      hovermode: false,
      showlegend: false,
    } };
};

const SlotChart: React.FC<Props> = ({ title, resources, ...rest }: Props) => {
  const [ oldPlotInfo, setOldPlotInfo ] = useState<PlotInfo | null>(null);

  const plotInfo = useMemo(() => {
    const newPlotInfo = genPlotInfo(title, resources || []);
    if (JSON.stringify(newPlotInfo) === JSON.stringify(oldPlotInfo)) return oldPlotInfo;
    setOldPlotInfo(newPlotInfo);
    return newPlotInfo;
  }, [ oldPlotInfo, resources, title ]);

  if (plotInfo === null) return <React.Fragment />;

  return (
    <Plot
      {...rest}
      config={{ displaylogo: false, displayModeBar: false, responsive: false }}
      data={plotInfo.data}
      layout={plotInfo.layout}
    />
  );
};

export default SlotChart;
