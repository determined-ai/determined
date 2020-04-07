import { Dropdown, Menu } from 'antd';
import { ClickParam } from 'antd/es/menu';
import React, { useCallback } from 'react';
import styled from 'styled-components';
import { theme } from 'styled-tools';

import Icon from 'components/Icon';
import Experiments from 'contexts/ActiveExperiments';
import { archiveExperiment, killTask } from 'services/api';
import { Experiment, RecentTask, RunState, TaskType } from 'types';
import Logger from 'utils/Logger';
import { isTaskKillable, terminalRunStates } from 'utils/types';

const logger = new Logger('TaskActionDropdown');

interface Props {
  task: RecentTask;
}

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
        case 'toggleArchive':
          await archiveExp();
          break;
      }
    } catch (e) {
      logger.error(`failed to perform the requested ${params.key} action.`);
      logger.error(e);
    }
    // TODO show loading indicator when we have a button component that supports it.
  };

  const stopPropagation = useCallback((e: React.MouseEvent) => e.stopPropagation(), []);

  const menu = (
    <Menu onClick={handleMenuClick}>
      {isKillable && <Menu.Item key="kill">Kill</Menu.Item>}
      {isArchivable && <Menu.Item key="toggleArchive">Archive</Menu.Item>}
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
