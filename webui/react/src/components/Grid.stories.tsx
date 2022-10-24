import { Meta, Story } from '@storybook/react';
import React from 'react';

import { ShirtSize } from 'themes';

import Grid, { GridMode } from './Grid';

export default {
  argTypes: {
    gap: { control: { options: ShirtSize, type: 'inline-radio' } },
    gridElements: { control: { max: 50, min: 0, step: 1, type: 'range' } },
    mode: { control: { options: GridMode, type: 'inline-radio' } },
  },
  component: Grid,
  parameters: { layout: 'padded' },
  title: 'Determined/Grid',
} as Meta<typeof Grid>;

const GridElement: React.FC = () => {
  const style = {
    backgroundColor: '#666',
    border: '1px black solid',
    height: '50px',
  };
  return <div style={style} />;
};

const GridElements: React.ReactNodeArray = new Array(27)
  .fill(0)
  .map((_, idx) => <GridElement key={idx} />);

type GridProps = React.ComponentProps<typeof Grid>;

export const Default: Story<GridProps & { gridElements: number }> = ({ gridElements, ...args }) => (
  <Grid {...args}>
    {new Array(gridElements).fill(0).map((_, idx) => (
      <GridElement key={idx} />
    ))}
  </Grid>
);

export const SmallCells = (): React.ReactNode => (
  <Grid gap={ShirtSize.Large} minItemWidth={100}>
    {GridElements}
  </Grid>
);

export const BigCells = (): React.ReactNode => (
  <Grid gap={ShirtSize.Large} minItemWidth={300}>
    {GridElements}
  </Grid>
);

Default.args = {
  gap: ShirtSize.Large,
  gridElements: 27,
  minItemWidth: 240,
  mode: GridMode.AutoFit,
};
