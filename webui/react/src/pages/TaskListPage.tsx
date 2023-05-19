import React, { useRef } from 'react';

import Page from 'components/Page';
import TaskList from 'components/TaskList';
import { paths } from 'routes/utils';

const TaskListPage: React.FC = () => {
  const pageRef = useRef<HTMLElement>(null);

  return (
    <Page
      breadcrumb={[
        {
          breadcrumbName: 'Tasks',
          path: paths.taskList(),
        },
      ]}
      containerRef={pageRef}
      id="tasks"
      title="Tasks">
      <TaskList />
    </Page>
  );
};

export default TaskListPage;
