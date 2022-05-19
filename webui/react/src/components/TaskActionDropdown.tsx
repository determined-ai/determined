import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Dropdown, Menu, Modal } from 'antd';
import { MenuInfo } from 'rc-menu/lib/interface';
import React, { PropsWithChildren, useCallback } from 'react';

import Icon from 'components/Icon';

import useModalExperimentDelete from 'hooks/useModal/useModalExperimentDelete';
import { paths } from 'routes/utils';
import {
  killTask,
} from 'services/api';
import { capitalize } from 'shared/utils/string';
import {
  ExperimentAction as Action, AnyTask, CommandTask, DetailedUser,
} from 'types';
import handleError from 'utils/error';
import { isTaskKillable } from 'utils/task';

import { ErrorLevel, ErrorType } from '../shared/utils/error';

import css from './ActionDropdown.module.scss';
import Link from './Link';

interface Props {
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
}: PropsWithChildren<Props>) => {

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

  const menuItems: React.ReactNode[] = [];

  if (isKillable) menuItems.push(<Menu.Item key={Action.Kill}>Kill</Menu.Item>);

  menuItems.push(
    <Menu.Item key={Action.ViewLogs}>
      <Link path={paths.taskLogs(task as CommandTask)}>View Logs</Link>
    </Menu.Item>,
  );

  const menu = <Menu onClick={handleMenuClick}>{menuItems}</Menu>;

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
