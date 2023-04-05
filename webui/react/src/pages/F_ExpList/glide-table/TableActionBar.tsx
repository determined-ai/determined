import { Menu, Space } from 'antd';
import { ItemType } from 'rc-menu/lib/interface';
import React, { useCallback, useMemo } from 'react';

import Dropdown from 'components/Dropdown';
import Button from 'components/kit/Button';
import Icon from 'shared/components/Icon';
import { ExperimentAction } from 'types';

import css from './TableActionBar.module.scss';

interface Action {
  disabled?: boolean;
  label: ExperimentAction;
}

const actionIcons: Partial<Record<ExperimentAction, string>> = {
  [ExperimentAction.Activate]: 'play',
  [ExperimentAction.Pause]: 'pause',
  [ExperimentAction.Cancel]: 'stop',
  [ExperimentAction.Archive]: 'archive',
  [ExperimentAction.Unarchive]: 'document',
  [ExperimentAction.Move]: 'workspaces',
  [ExperimentAction.OpenTensorBoard]: 'tensor-board',
  [ExperimentAction.Kill]: 'cancelled',
  [ExperimentAction.Delete]: 'error',
} as const;

interface Props {
  actions?: Action[];
  onAction: (action: string) => void;
  selectAll: boolean;
  selectedRowCount: number;
}

const TableActionBar: React.FC<Props> = ({
  actions = [],
  onAction,
  selectAll,
  selectedRowCount,
}) => {
  const handleAction = useCallback(
    ({ key }: { key: string }) => {
      onAction?.(key);
    },
    [onAction],
  );

  const editMenuItems: ItemType[] = useMemo(() => {
    return actions.map((action) => ({
      disabled: action.disabled,
      // The icon doesn't show up without being wrapped in a div.
      icon: (
        <div>
          <Icon name={actionIcons[action.label]} />
        </div>
      ),
      key: action.label,
      label: action.label,
    }));
  }, [actions]);

  return (
    <Space className={css.base}>
      {(selectAll || selectedRowCount > 0) && (
        <Dropdown content={<Menu items={editMenuItems} onClick={handleAction} />}>
          <Button icon={<Icon name="pencil" />}>
            Edit ({selectAll ? 'All' : selectedRowCount})
          </Button>
        </Dropdown>
      )}
    </Space>
  );
};

export default TableActionBar;
