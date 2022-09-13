import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Dropdown, Menu, Modal } from 'antd';
import type { MenuProps } from 'antd';
import { MenuInfo } from 'rc-menu/lib/interface';
import React from 'react';

import { paths } from 'routes/utils';
import {
  killTask,
} from 'services/api';
import css from 'shared/components/ActionDropdown/ActionDropdown.module.scss';
import Icon from 'shared/components/Icon/Icon';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { capitalize } from 'shared/utils/string';
import {
  ExperimentAction as Action, AnyTask, CommandTask, DetailedUser,
} from 'types';
import handleError from 'utils/error';
import { isTaskKillable } from 'utils/task';

import Link from './Link';

interface Props {
  children?: React.ReactNode;
  curUser?: DetailedUser;
  onComplete?: (action?: Action) => void;
  onVisibleChange?: (visible: boolean) => void;
  task: AnyTask;
}

const stopPropagation = (e: React.MouseEvent): void => e.stopPropagation();

const TaskActionDropdown: React.FC<Props> = ({
  task,
  onComplete,
  onVisibleChange,
  children,
}: Props) => {

  const isKillable = isTaskKillable(task);

  const handleMenuClick = (params: MenuInfo): void => {
    params.domEvent.stopPropagation();
    try {
      const action = params.key as Action;
      switch (action) { // Cases should match menu items.
        case Action.Kill:
          Modal.confirm({
            content: `
              Are you sure you want to kill
              this task?
            `,
            icon: <ExclamationCircleOutlined />,
            okText: 'Kill',
            onOk: async () => {
              await killTask(task as CommandTask);
              onComplete?.(action);
            },
            title: 'Confirm Task Kill',
          });
          break;
        case Action.ViewLogs:
          break;
      }
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: `Unable to ${params.key} task ${task.id}.`,
        publicSubject: `${capitalize(params.key.toString())} failed.`,
        silent: false,
        type: ErrorType.Server,
      });
    } finally {
      onVisibleChange?.(false);
    }
    // TODO show loading indicator when we have a button component that supports it.
  };

  const menuItems: MenuProps['items'] = [];

  if (isKillable) menuItems.push({ key: Action.Kill, label: 'Kill' });

  menuItems.push({
    key: Action.ViewLogs,
    label: <Link path={paths.taskLogs(task as CommandTask)}>View Logs</Link>,
  });

  const menu = <Menu items={menuItems} onClick={handleMenuClick} />;

  return children ? (
    <Dropdown
      overlay={menu}
      placement="bottomLeft"
      trigger={[ 'contextMenu' ]}
      onVisibleChange={onVisibleChange}>
      {children}
    </Dropdown>
  ) : (
    <div className={css.base} title="Open actions menu" onClick={stopPropagation}>
      <Dropdown overlay={menu} placement="bottomRight" trigger={[ 'click' ]}>
        <button onClick={stopPropagation}>
          <Icon name="overflow-vertical" />
        </button>
      </Dropdown>
    </div>
  );
};

export default TaskActionDropdown;
