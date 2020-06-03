import React from 'react';

import Page from 'components/Page';
import TaskTable from 'components/TasksTable';
import { Commands, Notebooks, Shells, Tensorboards } from 'contexts/Commands';
import Experiments from 'contexts/Experiments';
import { commandToTask, experimentToTask } from 'utils/types';

const TaskList: React.FC = () => {
  const commands = Commands.useStateContext();
  const notebooks = Notebooks.useStateContext();
  const shells = Shells.useStateContext();
  const tensorboards = Tensorboards.useStateContext();
  const experiments = Experiments.useStateContext();

  const genericCommands = [
    ...(commands.data || []),
    ...(notebooks.data || []),
    ...(shells.data || []),
    ...(tensorboards.data || []),
  ];

  const loadedTasks = [
    ...(experiments.data || []).map(experimentToTask),
    ...genericCommands.map(commandToTask),
  ];

  // TODO select and batch operation:
  // https://ant.design/components/table/#components-table-demo-row-selection-and-operation
  return (
    <Page title="Tasks">
      <TaskTable tasks={loadedTasks} />
    </Page>
  );
};

export default TaskList;
