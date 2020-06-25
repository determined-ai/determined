import React, { PropsWithChildren } from 'react';

import css from './TableBatch.module.scss';

interface Props {
  ids?: string[];
  message: string;
  show?: boolean;
}

const defaultProps = {
  ids: [],
  show: true,
};

const TableBatch: React.FC<Props> = (props: PropsWithChildren<Props>) => {
  const classes = [ css.base ];

  if (props.show) classes.push(css.show);

  return (
    <div className={classes.join(' ')}>
      <div className={css.container}>
        <div className={css.actions}>{props.children}</div>
        <div className={css.message}>{props.message}</div>
      </div>
    </div>
  );
};

TableBatch.defaultProps = defaultProps;

export default TableBatch;
