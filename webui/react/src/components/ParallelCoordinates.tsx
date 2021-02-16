import React, { useEffect, useMemo, useRef, useState } from 'react';
import { throttle } from 'throttle-debounce';

import useResize, { DEFAULT_RESIZE_THROTTLE_TIME } from 'hooks/useResize';
import Plotly, { Layout, PlotData } from 'Plotly';
import { ExperimentHyperParamType, Point, Primitive, Range } from 'types';
import { clone, isBoolean, isNumber } from 'utils/data';
import { generateAlphaNumeric, truncate } from 'utils/string';

export type NumRange = [ number, number ];

export enum DimensionType {
  Categorical = 'categorical',
  Scalar = 'scalar',
}

/*
 * `colors` - list of numbers between 0.0 and 1.0
 */
interface Props {
  colorScale: ColorScale
  colorScaleKey?: string;
  data: Record<string, Primitive[]>;
  dimensions: Dimension[];
  id?: string;
  onHover?: (lineIndex: number, point: Point) => void;
  onUnhover?: () => void;
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

export type ColorScale = [ number, string ][];

export const dimensionTypeMap: Record<ExperimentHyperParamType, DimensionType> = {
  [ExperimentHyperParamType.Categorical]: DimensionType.Categorical,
  [ExperimentHyperParamType.Constant]: DimensionType.Scalar,
  [ExperimentHyperParamType.Double]: DimensionType.Scalar,
  [ExperimentHyperParamType.Int]: DimensionType.Scalar,
  [ExperimentHyperParamType.Log]: DimensionType.Scalar,
};

const MAX_LABEL_LENGTH = 20;
const CONSTRAINT_REMOVE_THRESHOLD = 1e-9;

const plotlyLayout: Partial<Layout> = {
  height: 450,
  margin: { b: 32, t: 120 },
  paper_bgcolor: 'transparent',
};
const plotlyConfig: Partial<Plotly.Config> = {
  displayModeBar: false,
  responsive: true,
};

const ParallelCoordinates: React.FC<Props> = ({
  colorScale,
  colorScaleKey,
  data,
  dimensions,
  onHover,
  onUnhover,
  ...props
}: Props) => {
  const chartRef = useRef<HTMLDivElement>(null);
  const colorBarRef = useRef<HTMLDivElement>(null);
  const resize = useResize(chartRef);
  const [ id ] = useState(props.id ? props.id : generateAlphaNumeric());
  const [ chartState, setChartState ] = useState<ChartState>({});

  const colorValues = useMemo(() => {
    if (!colorScaleKey || !Array.isArray(data[colorScaleKey])) return undefined;
    return data[colorScaleKey]
      .map(value => isBoolean(value) ? value.toString() : value) as (number | string)[];
  }, [ colorScaleKey, data ]);

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
        const label = truncate(key, MAX_LABEL_LENGTH);
        const hpDimension: Record<string, unknown> = {
          label,
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
      labelangle: -45,
      line: {
        cmax: colorScale.last().scale,
        cmin: colorScale.first().scale,
        color: colorValues,
        colorbar: { title: colorScaleKey, titleside: 'right' },
        colorscale: colorScale.map((cs, index) => ([ index, cs.color ])),
      },
      type: 'parcoords',
    };
  }, [
    chartState,
    colorScale,
    colorScaleKey,
    colorValues,
    data,
    sortedDimensions,
  ]);

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

  // General color scale legend
  useEffect(() => {
    const ref = colorBarRef.current;
    if (!ref) return;

    const scales = colorScale.map(scale => scale[1]);
    if (smallerIsBetter) scales.reverse();

    const gradient = scales.join(', ');
    ref.style.backgroundImage = `linear-gradient(90deg, ${gradient})`;
  }, [ colorScale, smallerIsBetter ]);

  return <div id={id} ref={chartRef} />;
};

export default ParallelCoordinates;
