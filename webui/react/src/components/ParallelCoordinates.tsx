import React, { useEffect, useMemo, useRef, useState } from 'react';

import Plotly, { Layout, PlotData } from 'Plotly';
import { ExperimentHyperParamType, Primitive, Range } from 'types';
import { clone, isNumber } from 'utils/data';
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

const CONSTRAINT_REMOVE_THRESHOLD = 0.000000001;

const plotlyLayout: Partial<Layout> = { paper_bgcolor: 'transparent' };
const plotlyConfig: Partial<Plotly.Config> = {
  displayModeBar: false,
  responsive: true,
};

const ParallelCoordinates: React.FC<Props> = ({
  colors,
  data,
  dimensions,
  lineIds,
  ...props
}: Props) => {
  const chartRef = useRef<HTMLDivElement>(null);
  const [ id ] = useState(props.id ? props.id : generateAlphaNumeric());
  const [ constraintRanges, setConstraintRanges ] = useState<Range[]>([]);

  const chartData: Partial<PlotData> = useMemo(() => {
    const chartDimensions = dimensions.map((dimension, index) => {
      const key = dimension.label;
      const hpDimension: Record<string, unknown> = {
        // constraintrange: dimension.range,
        label: key,
        range: dimension.range,
        values: data[key],
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
        hpDimension.values = data[key].map(value => map[value.toString()]);
      }

      if (constraintRanges[index] != null) {
        hpDimension.constraintrange = clone(constraintRanges[index]);
      }

      return hpDimension;
    });

    return {
      dimensions: chartDimensions,
      line: { color: colors },
      type: 'parcoords',
    };
  }, [ constraintRanges, colors, data, dimensions ]);

  useEffect(() => {
    const ref = chartRef.current;
    if (!ref) return;

    Plotly.react(ref, [ chartData ], plotlyLayout, plotlyConfig);

    /*
     * During filtering, save all the constraint ranges to reconstruct the
     * filter during a re-rendering of the chart.
     */
    (ref as unknown as Plotly.PlotlyHTMLElement).on('plotly_restyle', data => {
      if (!Array.isArray(data) || data.length < 1) return;

      const keys = Object.keys(data[0]);
      if (!Array.isArray(keys) || keys.length === 0) return;

      const regex = /^dimensions\[(\d+)\]\.constraintrange/i;
      const matches = keys[0].match(regex);
      if (!Array.isArray(matches) || matches.length !== 2) return;

      const constraint: Range = data[0][keys[0]][0];
      const hpIndex = parseInt(matches[1]);
      setConstraintRanges(prev => {
        const newRanges = clone(prev);
        newRanges[hpIndex] = null;
        if (constraint) {
          if (isNumber(constraint[0]) && isNumber(constraint[1]) &&
              Math.abs(constraint[0] - constraint[1]) > CONSTRAINT_REMOVE_THRESHOLD) {
            newRanges[hpIndex] = constraint;
          } else if (constraint[0] !== constraint[1]) {
            newRanges[hpIndex] = constraint;
          }
        }
        return newRanges;
      });
    });

    return () => {
      if (ref) Plotly.purge(ref);
    };
    /* eslint-disable-next-line react-hooks/exhaustive-deps */
  }, [ chartData ]);

  return <div id={id} ref={chartRef} />;
};

export default ParallelCoordinates;
