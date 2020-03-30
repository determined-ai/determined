import React, { PropsWithChildren } from 'react';
import styled from 'styled-components';
import { prop } from 'styled-tools';

import { PropsWithTheme, ShirtSize } from 'themes';

export enum GridMode {
  AutoFill = 'auto-fill', // will squeeze as many items into a given space and minimum size
  AutoFit = 'auto-fit', // auto-fill but also stretch to fit the entire available space.
}

interface Props {
  gap?: ShirtSize;
  minItemWidth?: number;
  mode?: GridMode;
}

const defaultProps = {
  minItemWidth: 24,
  mode: GridMode.AutoFit,
};

const Grid: React.FC<Props> = (props: PropsWithChildren<Props>) => {
  return <Base {...props} data-test="grid">{props.children}</Base>;
};

const getGap = (props: PropsWithTheme<Props>): string => {
  return props.gap ? `grid-gap: ${props.theme.sizes.layout[props.gap]};` : '';
};

const Base = styled.div<Props>`
  display: grid;
  grid-template-columns:
    repeat(
      ${prop('mode', defaultProps.mode)},
      minmax(${prop('minItemWidth', defaultProps.minItemWidth)}rem, 1fr)
    );
  ${getGap}
`;

Grid.defaultProps = defaultProps;

export default Grid;
