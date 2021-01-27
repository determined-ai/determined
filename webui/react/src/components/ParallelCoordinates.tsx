import React, { useEffect, useMemo, useRef, useState } from 'react';

import Plotly, { Layout, PlotData, PlotType } from 'Plotly';
import { ExperimentHyperParamType, ExperimentHyperParamValue } from 'types';
import { clone } from 'utils/data';
import { generateAlphaNumeric } from 'utils/string';

export type Range = [ number, number ];

enum DimensionType {
  Categorical = 'categorical',
  Scalar = 'scalar',
}

interface Props {
  colors: number[];
  data: Record<string, ExperimentHyperParamValue[]>;
  dimensions: ConfigDimension[];
  id?: string;
  lineIds: number[];
}

interface Dimension {
  constraintrange?: Range;
  label: string;
  range: Range;
  ticktext?: string[];
  tickvals?: number[];
  values: number[];
}

interface Trace {
  dimensions: Dimension[];
  line: { color: number[] };
  type: PlotType,
}

export interface ConfigDimension {
  categories?: string[];
  constraintrange?: Range;
  label: string;
  range?: Range;
  type: DimensionType,
  valueRange?: Range;
}

interface Config {
  dimensionCount: number;
  dimensions: ConfigDimension[];
  trialCount: number;
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

function generateDimension(config: ConfigDimension, count: number): Dimension {
  const dimension: Dimension = {
    label: config.label,
    range: [ 0, 1 ],
    values: [],
  };

  for (let i = 0; i < count; i++) {
    if (config.type === DimensionType.Categorical && config.categories) {
      const categoryCount = config.categories.length;
      const index = Math.floor(Math.random() * categoryCount) * 2 + 1;
      dimension.values.push(index);
      dimension.tickvals = config.categories.map((_, index) => index * 2 + 1);
      dimension.ticktext = clone(config.categories);
      dimension.range = [ 0, categoryCount * 2 ];
    } else if (config.type === DimensionType.Scalar && config.valueRange) {
      const minValue = config.valueRange[0];
      const maxValue = config.valueRange[1];
      const value = Math.random() * (maxValue - minValue) + minValue;
      dimension.values.push(value);
      dimension.range = config.range || [ 0, 1 ];
    }
  }

  return dimension;
}

function generateTrace(config: Config): Partial<PlotData> {
  const trace: Trace = {
    dimensions: [],
    line: { color: [] },
    type: 'parcoords',
  };

  (trace.line.color as number[]) = new Array(config.trialCount).fill(0)
    .map(() => Math.random() * 100);

  (trace.dimensions as Dimension[]) = new Array(config.dimensionCount).fill(null).map(() => {
    const index = Math.floor(Math.random() * config.dimensions.length);
    const dimensionConfig = config.dimensions[index];
    return generateDimension(dimensionConfig, config.trialCount);
  });

  return trace as Partial<PlotData>;
}

const chartConfig: Config = {
  dimensionCount: 10,
  dimensions: [
    {
      label: 'learning_rate',
      range: [ 0, 1 ],
      type: DimensionType.Scalar,
      valueRange: [ 0, 1 ],
    },
    {
      categories: [ '0.009', '0.09', '0.9' ],
      label: 'momentum',
      type: DimensionType.Categorical,
    },
    {
      categories: [ '0.999', '0.099', '0.009' ],
      label: 'rms_prop',
      type: DimensionType.Categorical,
    },
    {
      categories: [ 'false', 'true' ],
      label: 'horizontal_flip',
      type: DimensionType.Categorical,
    },
    {
      label: 'dropout_1',
      range: [ 0, 1 ],
      type: DimensionType.Scalar,
      valueRange: [ 0, 1 ],
    },
    {
      label: 'dropout_2',
      range: [ 0, 1 ],
      type: DimensionType.Scalar,
      valueRange: [ 0, 1 ],
    },
    {
      categories: [ '16', '32', '64' ],
      label: 'global_batch_size',
      type: DimensionType.Categorical,
    },
    {
      label: 'width_shift_range',
      range: [ -1, 1 ],
      type: DimensionType.Scalar,
      valueRange: [ -1, 1 ],
    },
    {
      label: 'height_shift_range',
      range: [ 0, 0.2 ],
      type: DimensionType.Scalar,
      valueRange: [ 0.05, 0.15 ],
    },
    {
      label: '#_of_hidden_layers',
      range: [ 1, 20 ],
      type: DimensionType.Scalar,
      valueRange: [ 3, 16 ],
    },
    {
      label: '#_of_hidden_units',
      range: [ 10, 50 ],
      type: DimensionType.Scalar,
      valueRange: [ 20, 40 ],
    },
  ],
  trialCount: 1000,
};

// const ADD_COUNT = 5;

const ParallelCoordinates: React.FC<Props> = (props: Props) => {
  const chartRef = useRef<HTMLDivElement>(null);
  const [ id ] = useState(props.id ? props.id : generateAlphaNumeric());
  // const [ data ] = useState<Partial<PlotData>[]>([ generateTrace(chartConfig) ]);

  // const addDataPoint = useCallback(() => {
  //   const trace = data[0] as Trace;

  //   for (let i = 0; i < ADD_COUNT; i++) {
  //     const colorIndex = Math.floor(Math.random() * trace.line.color.length);
  //     trace.line.color.push(trace.line.color[colorIndex]);

  //     for (const dimension of trace.dimensions) {
  //       const index = Math.floor(Math.random() * dimension.values.length);
  //       dimension.values.push(dimension.values[index]);
  //     }
  //   }
  // }, []);

  // usePolling(addDataPoint, { delay: 5 });
  const data: Partial<PlotData> = useMemo(() => {
    const dimensions = props.dimensions.map(dim => {
      const hpDimension = { label: dim.label };
      return hpDimension;
    });
    return {
      dimensions,
      line: { color: props.colors },
      type: 'parcoords',
    };
  }, [ props.colors, props.dimensions ]);

  useEffect(() => {
    if (!chartRef.current) return;
    console.log('data', data);
    // Plotly.react(chartRef.current, data, plotlyLayout, plotlyConfig);
  }, [ chartRef, data ]);

  return <div id={id} ref={chartRef} />;
};

export default ParallelCoordinates;
