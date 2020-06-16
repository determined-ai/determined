import React from 'react';

import Page from 'components/Page';
import TaskTable from 'components/TaskTable';
import { Commands, Notebooks, Shells, Tensorboards } from 'contexts/Commands';
import { commandToTask } from 'utils/types';

const TaskList: React.FC = () => {
  const commands = Commands.useStateContext();
  const notebooks = Notebooks.useStateContext();
  const shells = Shells.useStateContext();
  const tensorboards = Tensorboards.useStateContext();

  const sources = [
    commands,
    notebooks,
    shells,
    tensorboards,
  ];

  const loadedTasks = sources
    .filter(src => src.data !== undefined)
    .map(src => src.data || [])
    .reduce((acc, cur) => [ ...acc, ...cur ], [])
    .map(commandToTask);

  const hasLoaded = sources.find(src => src.hasLoaded);

  // TODO select and batch operation:
  // https://ant.design/components/table/#components-table-demo-row-selection-and-operation
  return (
    <Page title="Tasks">
      <TaskTable tasks={hasLoaded ? loadedTasks : undefined} />
    </Page>
  );
};

export default TaskList;
