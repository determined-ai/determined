import Button from 'hew/Button';
import Dropdown, { MenuItem } from 'hew/Dropdown';
import Icon from 'hew/Icon';
import { useModal } from 'hew/Modal';
import { useToast } from 'hew/Toast';
import useConfirm from 'hew/useConfirm';
import React, { useMemo } from 'react';
import { useNavigate } from 'react-router-dom';

import css from 'components/ActionDropdown/ActionDropdown.module.scss';
import TaskConnectModalComponent, { TaskConnectField } from 'components/TaskConnectModal';
import usePermissions from 'hooks/usePermissions';
import { paths, serverAddress } from 'routes/utils';
import { killTask } from 'services/api';
import { TaskAction as Action, CommandState, CommandTask, CommandType, DetailedUser } from 'types';
import { copyToClipboard } from 'utils/dom';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';
import { capitalize } from 'utils/string';
import { isTaskKillable } from 'utils/task';

interface Props {
  children?: React.ReactNode;
  curUser?: DetailedUser;
  onComplete?: (action?: Action) => void;
  onVisibleChange?: (visible: boolean) => void;
  task: CommandTask;
}

const TaskActionDropdown: React.FC<Props> = ({ task, onComplete, children }: Props) => {
  const { canModifyWorkspaceNSC } = usePermissions();
  const { openToast } = useToast();
  const TaskConnectModal = useModal(TaskConnectModalComponent);

  const isConnectable = (task: CommandTask): boolean => {
    const connectableTaskTypes: CommandType[] = [CommandType.JupyterLab, CommandType.Shell];
    return connectableTaskTypes.includes(task.type) && task.state === CommandState.Running;
  };

  const confirm = useConfirm();

  const taskConnectFields: TaskConnectField[] = useMemo(() => {
    switch (task.type) {
      case CommandType.JupyterLab:
        return [
          {
            label: 'Connect to notebook in VSCode using the remote Jupyter server address:',
            value: `${serverAddress()}${task.serviceAddress}`,
          },
        ];
      case CommandType.Shell:
        return [
          {
            label: 'Start an interactive SSH session in the terminal:',
            value: `det shell open ${task.id}`,
          },
        ];
      default:
        return [];
    }
  }, [task]);

  const menuItems: MenuItem[] = useMemo(() => {
    const items: MenuItem[] = [
      {
        key: Action.ViewLogs,
        label: 'View Logs',
      },
      {
        key: Action.CopyTaskID,
        label: 'Copy Task ID',
      },
    ];
    if (isTaskKillable(task, canModifyWorkspaceNSC({ workspace: { id: task.workspaceId } }))) {
      items.push({ key: Action.Kill, label: 'Kill' });
    }
    if (isConnectable(task)) {
      items.push({ key: Action.Connect, label: 'Connect' });
    }
    return items;
  }, [task, canModifyWorkspaceNSC]);

  const navigate = useNavigate();

  const handleDropdown = async (key: string) => {
    try {
      switch (key) {
        case Action.Connect:
          TaskConnectModal.open();
          break;
        case Action.Kill:
          confirm({
            content: 'Are you sure you want to kill this task?',
            danger: true,
            okText: 'Kill',
            onConfirm: async () => {
              await killTask(task);
              onComplete?.(key);
            },
            onError: handleError,
            title: 'Confirm Task Kill',
          });
          break;
        case Action.ViewLogs:
          onComplete?.(key);
          navigate(paths.taskLogs(task));
          break;
        case Action.CopyTaskID:
          await copyToClipboard(task.id);
          openToast({
            severity: 'Confirm',
            title: 'Task ID has been copied to clipboard.',
          });
          break;
      }
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: `Unable to ${key} task ${task.id}.`,
        publicSubject: `${capitalize(key)} failed.`,
        silent: false,
        type: ErrorType.Server,
      });
    }
    // TODO show loading indicator when we have a button component that supports it.
  };
  return children ? (
    <Dropdown isContextMenu menu={menuItems} onClick={handleDropdown}>
      {children}
    </Dropdown>
  ) : (
    <div className={css.base} title="Open actions menu">
      <Dropdown menu={menuItems} placement="bottomRight" onClick={handleDropdown}>
        <Button
          icon={<Icon name="overflow-vertical" size="small" title="Action menu" />}
          type="text"
        />
      </Dropdown>
      <TaskConnectModal.Component fields={taskConnectFields} title={`Connect to ${task.name}`} />
    </div>
  );
};

export default TaskActionDropdown;
