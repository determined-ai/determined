import { ExclamationCircleOutlined } from '@ant-design/icons';
import type { DropDownProps, MenuProps } from 'antd';
import { Dropdown } from 'antd';
import React, { useCallback, useMemo } from 'react';

import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';
import { killTask } from 'services/api';
import Icon from 'shared/components/Icon/Icon';
import { ValueOf } from 'shared/types';
import { routeToReactUrl } from 'shared/utils/routes';
import { CommandTask, CommandType } from 'types';
import handleError from 'utils/error';

import { useModal } from './kit/Modal';
import css from './TaskBar.module.scss';
import TaskKillModalComponent from './TaskKillModal';
interface Props {
  handleViewLogsClick: () => void;
  id: string;
  name: string;
  resourcePool: string;
  type: CommandType;
}

export const TaskBar: React.FC<Props> = ({
  handleViewLogsClick,
  id,
  name,
  resourcePool,
  type,
}: Props) => {
  const TaskKillModal = useModal(TaskKillModalComponent);
  const { canModifyWorkspaceNSC } = usePermissions();
  const task = useMemo(() => {
    return { id, name, resourcePool, type } as CommandTask;
  }, [id, name, resourcePool, type]);

  const onKill = useCallback(() => {
    routeToReactUrl(paths.taskList());
  }, []);

  const dropdownOverlay: DropDownProps['menu'] = useMemo(() => {
    const MenuKey = {
      Kill: 'kill',
      ViewLogs: 'viewLogs',
    } as const;

    const funcs = {
      [MenuKey.Kill]: () => {
        TaskKillModal.open();
      },
      [MenuKey.ViewLogs]: () => {
        handleViewLogsClick();
      },
    };

    const onItemClick: MenuProps['onClick'] = (e) => {
      funcs[e.key as ValueOf<typeof MenuKey>]();
    };

    const menuItems: MenuProps['items'] = [
      {
        disabled: !canModifyWorkspaceNSC({ workspace: { id: task.workspaceId } }),
        key: MenuKey.Kill,
        label: 'Kill',
      },
      { key: MenuKey.ViewLogs, label: 'View Logs' },
    ];

    return { items: menuItems, onClick: onItemClick };
  }, [task, handleViewLogsClick, canModifyWorkspaceNSC, TaskKillModal]);

  return (
    <div className={css.base}>
      <div className={css.barContent}>
        <span>{name}</span>
        <span>&#8212;</span>
        <Dropdown menu={dropdownOverlay} placement="bottomRight" trigger={['click']}>
          <div className={css.dropdownTrigger} data-testid="task-action-dropdown-trigger">
            <span className={css.dropdownTrigger}>{resourcePool}</span>
            <Icon name="arrow-down" size="tiny" />
          </div>
        </Dropdown>
      </div>
      <TaskKillModal.Component task={task} onKill={onKill} />
    </div>
  );
};

export default TaskBar;
