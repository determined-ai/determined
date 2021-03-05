import React, { useEffect, useMemo, useRef, useState } from 'react';
import { throttle } from 'throttle-debounce';

import useResize, { DEFAULT_RESIZE_THROTTLE_TIME } from 'hooks/useResize';
import Plotly, { Layout, PlotData } from 'Plotly';
import { ExperimentHyperParamType, Primitive, Range } from 'types';
import { ColorScale } from 'utils/color';
import { clone, isBoolean, isNumber } from 'utils/data';
import { generateAlphaNumeric, truncate } from 'utils/string';

import css from './ParallelCoordinates.module.scss';

export type NumRange = [ number, number ];

export enum DimensionType {
  Categorical = 'categorical',
  Scalar = 'scalar',
}

/*
 * `colors` - list of numbers between 0.0 and 1.0
 */
interface Props {
  colorScale: ColorScale[];
  colorScaleKey?: string;
  data: Record<string, Primitive[]>;
  dimensions: Dimension[];
  id?: string;
  onFilter?: (constraints: Record<string, Constraint>) => void;
}

export interface Dimension {
  categories?: Primitive[];
  label: string;
  range?: Range;
  type: DimensionType,
}

export interface Constraint {
  range: Range;
  values?: Primitive[];
}

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
  onFilter,
  ...props
}: Props) => {
  const chartRef = useRef<HTMLDivElement>(null);
  const resize = useResize(chartRef);
  const [ id ] = useState(props.id ? props.id : generateAlphaNumeric());
  const [ constraints, setConstraints ] = useState<Record<string, Constraint>>({});

  const colorValues = useMemo(() => {
    if (!colorScaleKey || !Array.isArray(data[colorScaleKey])) return undefined;
    return data[colorScaleKey]
      .map(value => isBoolean(value) ? value.toString() : value) as (number | string)[];
  }, [ colorScaleKey, data ]);

  const chartData: Partial<PlotData> = useMemo(() => {
    const chartDimensions = dimensions
      .map(dimension => {
        if (!dimension) return;

        const key = dimension.label;
        const label = truncate(key, MAX_LABEL_LENGTH);
        const hpDimension: Record<string, unknown> = {
          label,
          multiselect: false,
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

        if (constraints[dimension.label] != null) {
          hpDimension.constraintrange = clone(constraints[dimension.label].range);
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
    colorScale,
    colorScaleKey,
    colorValues,
    constraints,
    data,
    dimensions,
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

      // Check for user applied filter on a dimension
      const regex = /^dimensions\[(\d+)\]\.constraintrange/i;
      const matches = keys[0].match(regex);
      if (Array.isArray(matches) && matches.length === 2) {
        const range: Range = data[0][keys[0]] ? data[0][keys[0]][0] : undefined;
        const dimIndex = parseInt(matches[1]);
        const dim = dimensions[dimIndex];
        const dimKey = dim.label;
        const constraint: Constraint = { range };

        // Translate constraints back to categorical values.
        if (dim.categories && range && isNumber(range[0]) && isNumber(range[1])) {
          // Create a list of acceptable categorical values.
          const minIndex = Math.round((Math.ceil(range[0]) - 1) / 2);
          const maxIndex = Math.round((Math.floor(range[1]) - 1) / 2);
          const values = dim.categories.slice(minIndex, maxIndex + 1);
          constraint.values = values;
        }

        setConstraints(prev => {
          const newConstraints = clone(prev);

          if (range == null) {
            delete newConstraints[dimKey];
          } else if (isNumber(range[0]) && isNumber(range[1]) &&
              Math.abs(range[0] - range[1]) > CONSTRAINT_REMOVE_THRESHOLD) {
            newConstraints[dimKey] = constraint;
          } else if (range[0] !== range[1]) {
            newConstraints[dimKey] = constraint;
          }

          return newConstraints;
        });
      }
    });

    return () => {
      if (ref) Plotly.purge(ref);
    };
    /* eslint-disable-next-line react-hooks/exhaustive-deps */
  }, [ chartData, onFilter ]);

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

  // Send back user created filters
  useEffect(() => {
    if (onFilter) onFilter(constraints);
  }, [ constraints, onFilter ]);

  return (
    <div className={css.base}>
      <div className={css.note}>
        Click and drag along the axes to create filters.
        Click on existing filters to remove them.
      </div>
      <div id={id} ref={chartRef} />
    </div>
  );
};

export default ParallelCoordinates;
