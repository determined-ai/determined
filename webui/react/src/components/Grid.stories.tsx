import React from 'react';
import styled from 'styled-components';

import Grid from './Grid';

export default {
  component: Grid,
  title: 'Grid',
};

const GridElement = styled.div`
  background-color: #666;
  border: 1px black solid;
  height: 5rem;
`;

const GridElements: React.ReactNodeArray =
  new Array(27).fill(0).map((_, idx) => <GridElement key={idx} />);

export const Default = (): React.ReactNode => <Grid>{GridElements}</Grid>;
export const SmallCells = (): React.ReactNode => <Grid minItemWidth={10}>{GridElements}</Grid>;
export const BigCells = (): React.ReactNode => <Grid minItemWidth={30}>{GridElements}</Grid>;
export const NoGap = (): React.ReactNode => <Grid gap={0}>{GridElements}</Grid>;
