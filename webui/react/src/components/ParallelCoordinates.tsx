import React, { useEffect, useMemo, useState } from 'react';

import Plotly, { Layout, PlotData } from 'Plotly';
import { ExperimentHyperParamType, Primitive, Range } from 'types';
import { generateAlphaNumeric } from 'utils/string';

export type NumRange = [ number, number ];

enum DimensionType {
  Categorical = 'categorical',
  Scalar = 'scalar',
}

interface Props {
  colors: number[];
  data: Record<string, Primitive[]>;
  dimensions: Dimension[];
  id?: string;
  lineIds: number[];
}

export interface Dimension {
  categories?: Primitive[];
  label: string;
  range?: Range;
  type: DimensionType,
}

export const dimensionTypeMap: Record<ExperimentHyperParamType, DimensionType> = {
  [ExperimentHyperParamType.Categorical]: DimensionType.Categorical,
  [ExperimentHyperParamType.Constant]: DimensionType.Scalar,
  [ExperimentHyperParamType.Double]: DimensionType.Scalar,
  [ExperimentHyperParamType.Int]: DimensionType.Scalar,
  [ExperimentHyperParamType.Log]: DimensionType.Scalar,
};

const plotlyLayout: Partial<Layout> = { paper_bgcolor: 'transparent' };
const plotlyConfig: Partial<Plotly.Config> = {
  displayModeBar: false,
  responsive: true,
};

const ParallelCoordinates: React.FC<Props> = (props: Props) => {
  const [ id ] = useState(props.id ? props.id : generateAlphaNumeric());

  const data: Partial<PlotData> = useMemo(() => {
    const dimensions = props.dimensions.map(dimension => {
      const key = dimension.label;
      const hpDimension: Record<string, unknown> = {
        label: key,
        range: dimension.range,
        values: props.data[key],
      };

      if (dimension.categories) {
        const { map, ticktext, tickvals } = dimension.categories.reduce((acc, category, index) => {
          const key = category.toString();
          const value = index * 2 + 1;
          acc.map[key] = value;
          acc.ticktext.push(key);
          acc.tickvals.push(value);
          return acc;
        }, {
          map: {} as Record<string, number>,
          ticktext: [] as string[],
          tickvals: [] as number[],
        });
        hpDimension.range = [ 0, dimension.categories.length * 2 ];
        hpDimension.ticktext = ticktext;
        hpDimension.tickvals = tickvals;
        hpDimension.values = props.data[key].map(value => map[value.toString()]);
      }

      return hpDimension;
    });

    return {
      dimensions,
      line: { color: props.colors },
      type: 'parcoords',
    };
  }, [ props.colors, props.data, props.dimensions ]);

  useEffect(() => {
    Plotly.react(id, [ data ], plotlyLayout, plotlyConfig);

    return () => Plotly.purge(id);
  }, [ id, data ]);

  return <div id={id} />;
};

export default ParallelCoordinates;
