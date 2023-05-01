import React, { ReactNode } from 'react';

import css from './Columns.module.scss';

interface ColumnProps {
  children: ReactNode;
  align?: 'left' | 'center' | 'right';
}

interface ColumnsProps {
  children: ReactNode;
  gap?: number;
  header?: boolean;
}

export const Column: React.FC<ColumnProps> = ({ children, align = 'left' }: ColumnProps) => {
  return <div className={`${css[align]} ${css.column}`}>{children}</div>;
};

export const Columns: React.FC<ColumnsProps> = ({ children, gap, header }: ColumnsProps) => {
  const classNames = [css.columns];
  if (header) classNames.push(css.header);

  return (
    <div className={classNames.join(' ')} style={{ gap }}>
      {children}
    </div>
  );
};
