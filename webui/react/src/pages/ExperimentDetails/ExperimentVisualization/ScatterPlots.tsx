import React, { useMemo } from 'react';

import Grid, { GridMode } from 'components/Grid';
import ScatterPlot from 'components/ScatterPlot';
import { ShirtSize } from 'themes';

const ScatterPlots: React.FC = () => {
  const x = useMemo(() => ([ 1, 2, 3, 4, 5, 6, 7, 8, 9, 10 ]), []);
  const y = useMemo(() => new Array(10).fill(null).map(() => Math.random()), []);
  return (
    <Grid gap={ShirtSize.big} mode={GridMode.AutoFill}>
      {new Array(100).fill(null).map((_, index) => (
        <ScatterPlot key={index} values={y} x={x} y={y} />
      ))}
    </Grid>
  );
};

export default ScatterPlots;
