import React from 'react';

import Chart from 'components/Chart';
import Grid, { GridMode } from 'components/Grid';
import { ShirtSize } from 'themes';

const ScatterPlots: React.FC = () => {
  return (
    <Grid gap={ShirtSize.big} mode={GridMode.AutoFill}>
      {new Array(100).fill(null).map((_, index) => (
        <Chart key={index} />
      ))}
    </Grid>
  );
};

export default ScatterPlots;
