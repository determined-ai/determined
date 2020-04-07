import React, { PropsWithChildren } from 'react';
import styled from 'styled-components';
import { prop } from 'styled-tools';

export enum GridMode {
  AutoFill = 'auto-fill', // will squeeze as many items into a given space and minimum size
  AutoFit = 'auto-fit', // auto-fill but also stretch to fit the entire available space.
}

interface Props {
  gap?: number; // TODO turn this into a string to use theme sizes.
  minItemWidth?: number;
  mode?: GridMode;
}

const defaultProps = {
  gap: 0.6,
  minItemWidth: 24,
  mode: GridMode.AutoFit,
};

const Grid: React.FC<Props> = (props: PropsWithChildren<Props>) => {
  return <Base {...props} data-test="grid">{props.children}</Base>;
};

const Base = styled.div<Props>`
  display: grid;
  grid-gap: ${prop('gap', defaultProps.gap)}rem;
  grid-template-columns:
    repeat(
      ${prop('mode', defaultProps.mode)},
      minmax(${prop('minItemWidth', defaultProps.minItemWidth)}rem, 1fr)
    );
`;

Grid.defaultProps = defaultProps;

export default Grid;
