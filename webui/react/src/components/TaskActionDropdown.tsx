import { Dropdown, Menu } from 'antd';
import { ClickParam } from 'antd/es/menu';
import React from 'react';

import Icon from 'components/Icon';
import Experiments from 'contexts/ActiveExperiments';
import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import { archiveExperiment, killTask, setExperimentState } from 'services/api';
import { Experiment, RecentTask, RunState, TaskType } from 'types';
import { capitalize } from 'utils/string';
import { cancellableRunStates, isTaskKillable, terminalRunStates } from 'utils/types';

import css from './TaskActionDropdown.module.scss';

interface Props {
  task: RecentTask;
}

const stopPropagation = (e: React.MouseEvent): void => e.stopPropagation();

const TaskActionDropdown: React.FC<Props> = ({ task }: Props) => {
  const isExperiment = task.type === TaskType.Experiment;
  const isArchivable = isExperiment && terminalRunStates.includes(task.state as RunState);
  const isKillable = isTaskKillable(task);
  const isPausable = task.type === TaskType.Experiment
    && task.state === RunState.Active;
  const isResumable = task.type === TaskType.Experiment
    && task.state === RunState.Paused;
  const isCancelable = task.type === TaskType.Experiment
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
        case 'kill':
          await killTask(task);
          break;
        case 'archive':
          await archiveExperiment(parseInt(task.id), !task.archived);
          await updateExperimentLocally(exp => ({ ...exp, archived: true }));
          break;
        case 'cancel':
          await setExperimentState({
            experimentId: parseInt(task.id),
            state: RunState.StoppingCanceled,
          });
          await updateExperimentLocally(exp => ({ ...exp, state: RunState.StoppingCanceled }));
          break;
        case 'pause':
          await setExperimentState({
            experimentId: parseInt(task.id),
            state: RunState.Paused,
          });
          await updateExperimentLocally(exp => ({ ...exp, state: RunState.Paused }));
          break;
        case 'activate':
          await setExperimentState({
            experimentId: parseInt(task.id),
            state: RunState.Active,
          });
          await updateExperimentLocally(exp => ({ ...exp, state: RunState.Active }));
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
      {isKillable && <Menu.Item key="kill">Kill</Menu.Item>}
      {isArchivable && <Menu.Item key="archive">Archive</Menu.Item>}
      {isPausable && <Menu.Item key="pause">Pause</Menu.Item>}
      {isResumable && <Menu.Item key="activate">Activate</Menu.Item>}
      {isCancelable && <Menu.Item key="cancel">Cancel</Menu.Item>}
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
