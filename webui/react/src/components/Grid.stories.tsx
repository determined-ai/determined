import React from 'react';

import { ShirtSize } from 'themes';

import Grid from './Grid';

export default {
  component: Grid,
  parameters: { layout: 'padded' },
  title: 'Grid',
};

const GridElement: React.FC = () => {
  const style = {
    backgroundColor: '#666',
    border: 'px black solid',
    height: '5rem',
  };
  return <div style={style} />;
};

const GridElements: React.ReactNodeArray =
  new Array(27).fill(0).map((_, idx) => <GridElement key={idx} />);

export const Default = (): React.ReactNode => <Grid gap={ShirtSize.big}>{GridElements}</Grid>;

export const NoGap = (): React.ReactNode => <Grid>{GridElements}</Grid>;

export const SmallCells = (): React.ReactNode => (
  <Grid gap={ShirtSize.big} minItemWidth={10}>{GridElements}</Grid>
);

export const BigCells = (): React.ReactNode => (
  <Grid gap={ShirtSize.big} minItemWidth={30}>{GridElements}</Grid>
);
