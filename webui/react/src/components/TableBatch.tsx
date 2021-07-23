import { Button } from 'antd';
import React, { PropsWithChildren } from 'react';

import css from './TableBatch.module.scss';

interface Props {
  ids?: string[];
  onClear?: () => void;
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
        <div className={css.actions}>
          <div className={css.message}>{message}</div>
          {props.onClear &&
            <Button onClick={props.onClear}>Clear Selected</Button>
          }
        </div>
      </div>
    </div>
  );
};

TableBatch.defaultProps = defaultProps;

export default TableBatch;
