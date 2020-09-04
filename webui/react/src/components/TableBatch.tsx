import React, { PropsWithChildren } from 'react';

import css from './TableBatch.module.scss';

interface Props {
  ids?: string[];
  selectedRowCount?: number;
}

const defaultProps = {
  ids: [],
  selectedRowCount: 0,
};

const TableBatch: React.FC<Props> = (props: PropsWithChildren<Props>) => {
  const classes = [ css.base ];
  const selectedRowCount = props.selectedRowCount || 0;

  const message = `Apply batch operations to ${selectedRowCount}`+
    ` item${selectedRowCount === 1 ? '' : 's'}`;

  if (selectedRowCount > 0) classes.push(css.show);

  return (
    <div className={classes.join(' ')}>
      <div className={css.container}>
        <div className={css.actions}>{props.children}</div>
        <div className={css.message}>{message}</div>
      </div>
    </div>
  );
};

TableBatch.defaultProps = defaultProps;

export default TableBatch;
