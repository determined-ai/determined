import React, { CSSProperties, ReactNode } from 'react';

import css from './Columns.module.scss';

interface ColumnProps {
  children?: ReactNode;
  align?: 'left' | 'center' | 'right';
}

interface ColumnsProps {
  children?: ReactNode;
  gap?: 0 | 8 | 16;
  header?: boolean;
}

export const Column: React.FC<ColumnProps> = ({ children, align = 'left' }: ColumnProps) => {
  return <div className={`${css[align]} ${css.column}`}>{children}</div>;
};

export const Columns: React.FC<ColumnsProps> = ({ children, gap = 8, header }: ColumnsProps) => {
  const cssVars = {
    '--gap': gap + 'px',
    '--margin-bottom': header ? '16px' : 0,
  };

  return (
    <div className={css.columns} style={cssVars as CSSProperties}>
      {children}
    </div>
  );
};
