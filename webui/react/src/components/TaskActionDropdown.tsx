import { Dropdown, Menu } from 'antd';
import { ClickParam } from 'antd/es/menu';
import React from 'react';

import Icon from 'components/Icon';
import Experiments from 'contexts/ActiveExperiments';
import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import { archiveExperiment, killTask } from 'services/api';
import { Experiment, RecentTask, RunState, TaskType } from 'types';
import { capitalize } from 'utils/string';
import { isTaskKillable, terminalRunStates } from 'utils/types';

import css from './TaskActionDropdown.module.scss';

interface Props {
  task: RecentTask;
}

const stopPropagation = (e: React.MouseEvent): void => e.stopPropagation();

const TaskActionDropdown: React.FC<Props> = (props: Props) => {
  const isExperiment = props.task.type === TaskType.Experiment;
  const isArchivable = isExperiment && terminalRunStates.includes(props.task.state as RunState);
  const isKillable = isTaskKillable(props.task);

  if (!isArchivable && !isKillable) return (<div />);

  const experimentsResponse = Experiments.useStateContext();
  const setExperiments = Experiments.useActionContext();

  const archiveExp = async (): Promise<void> => {
    await archiveExperiment(parseInt(props.task.id), !props.task.archived);
    const localUpdate = (exp: Experiment): Experiment => {
      return { ...exp, archived: !props.task.archived };
    };
    if (experimentsResponse.data) {
      const updatedExperiments = experimentsResponse.data
        .map(exp => exp.id.toString() === props.task.id ? localUpdate(exp) : exp);
      setExperiments({
        type: Experiments.ActionType.Set,
        value: { ...experimentsResponse, data: updatedExperiments },
      });
    }
  };

  const handleMenuClick = async (params: ClickParam): Promise<void> => {
    params.domEvent.stopPropagation();
    try {
      switch (params.key) { // Cases should match menu items.
        case 'kill':
          await killTask(props.task);
          break;
        case 'archive':
          await archiveExp();
          break;
      }
    } catch (e) {
      handleError({
        error: e,
        level: ErrorLevel.Error,
        message: e.message,
        publicMessage: `Failed to ${params.key} task ${props.task.id}.`,
        publicSubject: `${capitalize(params.key)} failed.`,
        silent: false,
        type: ErrorType.Server,
      });
    }
    // TODO show loading indicator when we have a button component that supports it.
  };

  const menu = (
    <Menu onClick={handleMenuClick}>
      {isKillable && <Menu.Item key="kill">Kill</Menu.Item>}
      {isArchivable && <Menu.Item key="archive">Archive</Menu.Item>}
    </Menu>
  );

  return (
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
