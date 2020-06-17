import { Dropdown, Menu } from 'antd';
import { ClickParam } from 'antd/es/menu';
import React from 'react';

import Icon from 'components/Icon';
import Experiments from 'contexts/Experiments';
import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import { archiveExperiment, killTask, setExperimentState } from 'services/api';
import { AnyTask, Experiment, RunState } from 'types';
import { capitalize } from 'utils/string';
import { isExperimentTask } from 'utils/task';
import { cancellableRunStates, isTaskKillable, terminalRunStates } from 'utils/types';

import css from './TaskActionDropdown.module.scss';

interface Props {
  task: AnyTask;
}

const stopPropagation = (e: React.MouseEvent): void => e.stopPropagation();

const TaskActionDropdown: React.FC<Props> = ({ task }: Props) => {
  const isExperiment = isExperimentTask(task);
  const isArchivable = isExperiment && terminalRunStates.includes(task.state as RunState);
  const isKillable = isTaskKillable(task);
  const isPausable = isExperiment
    && task.state === RunState.Active;
  const isResumable = isExperiment
    && task.state === RunState.Paused;
  const isCancelable = isExperiment
    && cancellableRunStates.includes(task.state as RunState);

  if (!isArchivable && !isKillable) return (<div />);

  const experimentsResponse = Experiments.useStateContext();
  const setExperiments = Experiments.useActionContext();

  // update the local state of a single experiment.
  // TODO refactor to contexts.
  const updateExperimentLocally = (updater: (arg0: Experiment) => Experiment): void => {
    if (experimentsResponse.data) {
      const experiments = experimentsResponse.data
        .map(exp => exp.id.toString() === task.id ? updater(exp) : exp);
      setExperiments({
        type: Experiments.ActionType.Set,
        value: { ...experimentsResponse, data: experiments },
      });
    }
  };

  const handleMenuClick = async (params: ClickParam): Promise<void> => {
    params.domEvent.stopPropagation();
    try {
      switch (params.key) { // Cases should match menu items.
        case 'activate':
          await setExperimentState({
            experimentId: parseInt(task.id),
            state: RunState.Active,
          });
          updateExperimentLocally(exp => ({ ...exp, state: RunState.Active }));
          break;
        case 'archive':
          if (!isExperimentTask(task)) break;
          await archiveExperiment(parseInt(task.id), !task.archived);
          updateExperimentLocally(exp => ({ ...exp, archived: true }));
          break;
        case 'cancel':
          await setExperimentState({
            experimentId: parseInt(task.id),
            state: RunState.StoppingCanceled,
          });
          updateExperimentLocally(exp => ({ ...exp, state: RunState.StoppingCanceled }));
          break;
        case 'kill':
          await killTask(task);
          if (isExperiment) {
            // We don't provide immediate updates for command types yet.
            updateExperimentLocally(exp => ({ ...exp, state: RunState.StoppingCanceled }));
          }
          break;
        case 'pause':
          await setExperimentState({
            experimentId: parseInt(task.id),
            state: RunState.Paused,
          });
          updateExperimentLocally(exp => ({ ...exp, state: RunState.Paused }));
          break;
      }
    } catch (e) {
      handleError({
        error: e,
        level: ErrorLevel.Error,
        message: e.message,
        publicMessage: `Unable to ${params.key} task ${task.id}.`,
        publicSubject: `${capitalize(params.key)} failed.`,
        silent: false,
        type: ErrorType.Server,
      });
    }
    // TODO show loading indicator when we have a button component that supports it.
  };

  const menu = (
    <Menu onClick={handleMenuClick}>
      {isResumable && <Menu.Item key="activate">Activate</Menu.Item>}
      {isPausable && <Menu.Item key="pause">Pause</Menu.Item>}
      {isArchivable && <Menu.Item key="archive">Archive</Menu.Item>}
      {isCancelable && <Menu.Item key="cancel">Cancel</Menu.Item>}
      {isKillable && <Menu.Item key="kill">Kill</Menu.Item>}
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
