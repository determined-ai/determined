import React, { useEffect, useMemo, useRef, useState } from 'react';

import Plotly, { Layout, PlotData } from 'Plotly';
import { ExperimentHyperParamType, Point, Primitive, Range } from 'types';
import { clone, isNumber } from 'utils/data';
import { generateAlphaNumeric } from 'utils/string';

export type NumRange = [ number, number ];

export enum DimensionType {
  Categorical = 'categorical',
  Scalar = 'scalar',
}

/*
 * `colors` - list of numbers between 0.0 and 1.0
 */
interface Props {
  colors: number[];
  data: Record<string, Primitive[]>;
  dimensions: Dimension[];
  id?: string;
  onHover?: (lineIndex: number, point: Point) => void;
  onUnhover?: () => void;
  smallerIsBetter?: boolean;
}

interface ChartState {
  constraintRanges?: Range[];
  dimensionOrder?: string[];
}

interface HoverEvent {
  clientX: number;
  clientY: number;
  curveNumber: number;
  dataIndex: number;
  x: number;
  y: number;
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
const COLOR_SCALE = [
  [ 0.0, 'rgba(238, 0, 0, 1.0)' ],
  [ 0.5, 'rgba(238, 238, 0, 1.0)' ],
  [ 1.0, 'rgba(0, 238, 0, 1.0)' ],
];
const COLOR_SCALE_INVERSE = [
  [ 0.0, 'rgba(0, 238, 0, 1.0)' ],
  [ 0.5, 'rgba(238, 238, 0, 1.0)' ],
  [ 1.0, 'rgba(238, 0, 0, 1.0)' ],
];

const plotlyLayout: Partial<Layout> = { paper_bgcolor: 'transparent' };
const plotlyConfig: Partial<Plotly.Config> = {
  displayModeBar: false,
  responsive: true,
};

const ParallelCoordinates: React.FC<Props> = ({
  colors,
  data,
  dimensions,
  onHover,
  onUnhover,
  smallerIsBetter,
  ...props
}: Props) => {
  const chartRef = useRef<HTMLDivElement>(null);
  const [ id ] = useState(props.id ? props.id : generateAlphaNumeric());
  const [ chartState, setChartState ] = useState<ChartState>({});

  const sortedDimensions: Dimension[] = useMemo(() => {
    const dimensionOrder = chartState.dimensionOrder;
    if (!dimensionOrder) return dimensions;

    const unorderedDimensionKeys: string[] = [];
    dimensions.forEach(dimension => {
      if (!dimensionOrder.includes(dimension.label)) unorderedDimensionKeys.push(dimension.label);
    });

    const dimensionMap = dimensions.reduce((acc, dimension) => {
      acc[dimension.label] = dimension;
      return acc;
    }, {} as Record<string, Dimension>);

    return [ ...dimensionOrder, ...unorderedDimensionKeys ].map(key => dimensionMap[key]);
  }, [ chartState, dimensions ]);

  const chartData: Partial<PlotData> = useMemo(() => {
    const chartDimensions = sortedDimensions
      .map((dimension, index) => {
        if (!dimension) return;

        const key = dimension.label;
        const hpDimension: Record<string, unknown> = {
          label: key,
          range: dimension.range,
          values: data[key],
        };

        if (dimension.categories) {
          const { map, ticktext, tickvals } = dimension.categories
            .reduce((acc, category, index) => {
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

        if (chartState.constraintRanges && chartState.constraintRanges[index] != null) {
          hpDimension.constraintrange = clone(chartState?.constraintRanges[index]);
        }

        return hpDimension;
      })
      .filter(dimension => dimension !== undefined);

    return {
      dimensions: chartDimensions,
      line: {
        color: colors,
        colorbar: {},
        colorscale: smallerIsBetter ? COLOR_SCALE_INVERSE : COLOR_SCALE,
      },
      type: 'parcoords',
    };
  }, [ chartState, colors, data, smallerIsBetter, sortedDimensions ]);

  useEffect(() => {
    const ref = chartRef.current;
    if (!ref) return;

    Plotly.react(ref, [ chartData ], plotlyLayout, plotlyConfig);

    const plotly = ref as unknown as Plotly.PlotlyHTMLElement;

    /*
     * During filtering or ordering, save all the constraint ranges and column order
     * to reconstruct the chart correctly when re-rendering the chart.
     */
    plotly.on('plotly_restyle', data => {
      if (!Array.isArray(data) || data.length < 1) return;

      const keys = Object.keys(data[0]);
      if (!Array.isArray(keys) || keys.length === 0) return;

      // Check for user applied dimension reorder
      if (keys[0] === 'dimensions') {
        /* eslint-disable @typescript-eslint/no-explicit-any */
        const dimensions = data[0][keys[0]][0] || [];
        const dimensionKeys = dimensions.map((dimension: any) => dimension.label);
        const constraints = dimensions.map((dimension: any) => dimension.constraintrange);
        /* eslint-enable @typescript-eslint/no-explicit-any */

        setChartState({ constraintRanges: constraints, dimensionOrder: dimensionKeys });
      }

      // Check for user applied filter on a dimension
      const regex = /^dimensions\[(\d+)\]\.constraintrange/i;
      const matches = keys[0].match(regex);
      if (Array.isArray(matches) && matches.length === 2) {
        const constraint: Range = data[0][keys[0]] ? data[0][keys[0]][0] : undefined;
        const hpIndex = parseInt(matches[1]);

        setChartState(prev => {
          const newChartState = clone(prev);
          newChartState.constraintRanges = newChartState.constraintRanges || [];
          newChartState.constraintRanges[hpIndex] = undefined;
          if (constraint) {
            if (isNumber(constraint[0]) && isNumber(constraint[1]) &&
              Math.abs(constraint[0] - constraint[1]) > CONSTRAINT_REMOVE_THRESHOLD) {
              newChartState.constraintRanges[hpIndex] = constraint;
            } else if (constraint[0] !== constraint[1]) {
              newChartState.constraintRanges[hpIndex] = constraint;
            }
          }
          return newChartState;
        });
      }
    });

    plotly.on('plotly_hover', data => {
      if (!onHover) return;

      const event = data as unknown as HoverEvent;
      const lineIndex = event.curveNumber;
      onHover(lineIndex, { x: event.x, y: event.y });
    });

    plotly.on('plotly_unhover', () => {
      if (onUnhover) onUnhover();
    });

    return () => {
      if (ref) Plotly.purge(ref);
    };
    /* eslint-disable-next-line react-hooks/exhaustive-deps */
  }, [ chartData ]);

  return <div id={id} ref={chartRef} />;
};

export default ParallelCoordinates;
