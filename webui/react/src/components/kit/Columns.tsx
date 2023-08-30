import React, { CSSProperties, ReactNode } from 'react';

import css from 'components/kit/Columns.module.scss';

interface ColumnProps {
  children?: ReactNode;
  align?: 'left' | 'center' | 'right';
}

interface ColumnsProps {
  children?: ReactNode;
  gap?: 0 | 8 | 16;
  page?: boolean;
}

export const Column: React.FC<ColumnProps> = ({ children, align = 'left' }: ColumnProps) => {
  return <div className={`${css[align]} ${css.column}`}>{children}</div>;
};

export const Columns: React.FC<ColumnsProps> = ({ children, gap = 8, page }: ColumnsProps) => {
  const classes = [css.columns];
  if (page) classes.push(css.page);

  return (
    <div
      className={classes.join(' ')}
      style={
        {
          '--columns-gap': gap + 'px',
        } as CSSProperties
      }>
      {children}
    </div>
  );
};
