import React, { useRef } from 'react';

import Page from 'components/Page';
import TaskList from 'components/TaskList';

const TaskListPage: React.FC = () => {
  const pageRef = useRef<HTMLElement>(null);

  return (
    <Page containerRef={pageRef} id="tasks" title="Tasks">
      <TaskList />
    </Page>
  );
};

export default TaskListPage;
