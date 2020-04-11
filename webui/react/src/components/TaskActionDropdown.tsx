import { Dropdown, Menu } from 'antd';
import { ClickParam } from 'antd/es/menu';
import React, { useCallback } from 'react';
import styled from 'styled-components';
import { theme } from 'styled-tools';

import Icon from 'components/Icon';
import Experiments from 'contexts/ActiveExperiments';
import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import { archiveExperiment, killTask } from 'services/api';
import { Experiment, RecentTask, RunState, TaskType } from 'types';
import { capitalize } from 'utils/string';
import { isTaskKillable, terminalRunStates } from 'utils/types';

interface Props {
  task: RecentTask;
}

const TaskActionDropdown: React.FC<Props> = (props: Props) => {
  const isExperiment = props.task.type === TaskType.Experiment;
  const isArchivable = isExperiment && terminalRunStates.includes(props.task.state as RunState);
  const isKillable = isTaskKillable(props.task);

  const stopPropagation = useCallback((e: React.MouseEvent) => e.stopPropagation(), []);

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
      }, false);
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
    <div title="Open actions menu" onClick={stopPropagation}>
      <Dropdown overlay={menu} placement="bottomRight" trigger={[ 'click' ]}>
        <TransparentButton onClick={stopPropagation}>
          <Icon name="overflow-vertical" />
        </TransparentButton>
      </Dropdown>
    </div>
  );
};

const TransparentButton = styled.button`
  background: none;
  border: none;
  color: ${theme('colors.monochrome.8')};
  cursor: pointer;
  font: inherit;
  outline: inherit;
  padding: 0;
  &:hover { color: ${theme('colors.core.action')}; }
`;

export default TaskActionDropdown;
