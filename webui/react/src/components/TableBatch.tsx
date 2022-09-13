import { Button, Select } from 'antd';
import React, { useCallback, useState } from 'react';

import css from './TableBatch.module.scss';

interface Action<T> {
  disabled?: boolean;
  label: T;
  value: T;
}

interface Props<T> {
  actions?: Action<T>[];
  ids?: string[];
  onAction?: (action: T) => void;
  onClear?: () => void;
  selectedRowCount?: number;
}

const defaultProps = {
  ids: [],
  selectedRowCount: 0,
};

function TableBatch<T extends string>({
  actions,
  selectedRowCount,
  onAction,
  onClear,
}: Props<T>):React.ReactElement {
  const [ action, setAction ] = useState<T | ''>();
  const classes = [ css.base ];
  const selectCount = selectedRowCount || 0;

  const message = `Apply batch operations to ${selectCount}` +
    ` item${selectCount === 1 ? '' : 's'}`;

  if (selectCount > 0) classes.push(css.show);

  const handleAction = useCallback((action?: string) => {
    /*
     * This succession setting of action to an empty string
     * followed by `undefined` is required to guarantee clearing
     * out of the selection value. Using a state `value` prop and
     * setting the state to `undefined` did not work.
     */
    setAction('');
    setTimeout(() => setAction(undefined), 100);

    if (action) onAction?.(action as T);
  }, [ onAction ]);

  const handleClear = useCallback(() => {
    if (onClear) onClear();
  }, [ onClear ]);

  return (
    <div className={classes.join(' ')}>
      <div className={css.container}>
        <div className={css.actions}>
          <Select
            options={actions}
            placeholder="Select an action..."
            value={action}
            onSelect={handleAction}
          />
        </div>
        <div className={css.message}>{message}</div>
        <div className={css.clear}>
          <Button onClick={handleClear}>Clear</Button>
        </div>
      </div>
    </div>
  );
}

TableBatch.defaultProps = defaultProps;

export default TableBatch;
