import { Select } from 'antd';
import React, { useCallback, useMemo, useState } from 'react';

import Button from 'components/kit/Button';

import css from './TableBatch.module.scss';

export interface Action<T> {
  bulk?: boolean;
  disabled?: boolean;
  label: T;
  value: T;
}

interface Props<T> {
  actions?: Action<T>[];
  ids?: string[];
  onAction?: (action: T) => void;
  onChangeSelectionMode?: () => void;
  onClear?: () => void;
  selectAllMatching?: boolean;
  selectedRowCount?: number;
}

const defaultProps = {
  ids: [],
  selectedRowCount: 0,
};

function TableBatch<T extends string>({
  actions: _actions,

  selectedRowCount,
  selectAllMatching,
  onAction,
  onClear,
  onChangeSelectionMode,
}: Props<T>): React.ReactElement {
  const [action, setAction] = useState<T | ''>();
  const classes = [css.base];
  const selectCount = selectedRowCount || 0;

  const message = selectAllMatching
    ? 'Apply batch operations to all matching items'
    : selectCount === 0
    ? 'Select Items to Apply Actions'
    : `Apply batch operations to ${selectCount}` + ` item${selectCount === 1 ? '' : 's'}`;

  const actions = useMemo(
    () =>
      _actions?.map((a) => ({
        ...a,
        disabled: a.disabled || selectedRowCount === 0 || (!a.bulk && selectAllMatching),
      })),
    [_actions, selectAllMatching, selectedRowCount],
  );

  if (selectCount > 0) classes.push(css.show);

  const handleAction = useCallback(
    (action?: string) => {
      /*
       * This succession setting of action to an empty string
       * followed by `undefined` is required to guarantee clearing
       * out of the selection value. Using a state `value` prop and
       * setting the state to `undefined` did not work.
       */
      setAction('');
      setTimeout(() => setAction(undefined), 100);

      if (action) onAction?.(action as T);
    },
    [onAction],
  );

  const handleClear = useCallback(() => {
    if (onClear) onClear();
  }, [onClear]);

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
        <div className={css.selectMode}>
          {onChangeSelectionMode && (
            <Button onClick={onChangeSelectionMode}>
              {selectAllMatching ? 'Individual Selection' : 'Select All Matching'}
            </Button>
          )}
        </div>
        {!selectAllMatching && (
          <div className={css.clear}>
            <Button onClick={handleClear}>Clear</Button>
          </div>
        )}
      </div>
    </div>
  );
}

TableBatch.defaultProps = defaultProps;

export default TableBatch;
