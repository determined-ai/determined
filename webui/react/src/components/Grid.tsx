import React from 'react';

import { ShirtSize } from 'themes';

import css from './Grid.module.scss';

export enum GridMode {
  AutoFill = 'auto-fill', // will squeeze as many items into a given space and minimum size
  AutoFit = 'auto-fit', // auto-fill but also stretch to fit the entire available space.
}

interface Props {
  border?: boolean;
  children: React.ReactNode;
  gap?: ShirtSize;
  minItemWidth?: number;
  mode?: GridMode | number;
}

const sizeMap = {
  [ShirtSize.small]: '4px',
  [ShirtSize.medium]: '8px',
  [ShirtSize.large]: '16px',
};

const Grid: React.FC<Props> = ({
  border,
  gap = ShirtSize.medium,
  minItemWidth = 240,
  mode = GridMode.AutoFit,
  children,
}: Props) => {
  const style = {
    gridGap: `calc(${sizeMap[gap]} + var(--theme-density) * 1px)`,
    gridTemplateColumns: `repeat(${mode}, minmax(${minItemWidth}px, 1fr))`,
  };
  const classes = [ css.base ];

  if (border) classes.push(css.border);

  return (
    <div className={classes.join(' ')} style={style}>{children}</div>
  );
};

export default Grid;
