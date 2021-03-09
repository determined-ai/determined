import React, { PropsWithChildren } from 'react';

import { ShirtSize } from 'themes';

import css from './Grid.module.scss';

export enum GridMode {
  AutoFill = 'auto-fill', // will squeeze as many items into a given space and minimum size
  AutoFit = 'auto-fit', // auto-fill but also stretch to fit the entire available space.
}

interface Props {
  border?: boolean;
  gap?: ShirtSize;
  length?: number;
  minItemWidth?: number;
  mode?: GridMode;
}

const defaultProps = {
  minItemWidth: 24,
  mode: GridMode.AutoFit,
};

const Grid: React.FC<Props> = (props: PropsWithChildren<Props>) => {
  const classes = [ css.base ];
  const mode = props.mode || defaultProps.mode;
  const itemWidth = props.minItemWidth || defaultProps.minItemWidth;
  const length = props.length ? props.length : mode;
  const style = {
    gridGap: props.gap ? `var(--theme-sizes-layout-${props.gap})` : '',
    gridTemplateColumns: `repeat(${length}, minmax(${itemWidth}rem, 1fr))`,
  };

  if (props.border) classes.push(css.border);

  return (
    <div className={classes.join(' ')} style={style}>
      {props.children}</div>
  );
};

Grid.defaultProps = defaultProps;

export default Grid;
