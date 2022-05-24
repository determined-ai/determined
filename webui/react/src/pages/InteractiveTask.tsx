import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';

import TaskBar from 'components/TaskBar';
import { StoreAction, useStoreDispatch } from 'contexts/Store';
import { CommandType } from 'types';

import css from './InteractiveTask.module.scss';
import TaskLogs from './TaskLogs';

interface Params {
  taskId: string;
  taskName: string;
  taskResourcePool: string;
  taskType: CommandType
  taskUrl: string;
}

enum PageView {
  IFRAME= 'Iframe',
  TASK_LOGS = 'Task Logs'
}

export const InteractiveTask: React.FC = () => {

  const [ pageView, setPageView ] = useState<PageView>(PageView.IFRAME);
  const { taskId, taskName, taskResourcePool, taskUrl, taskType } = useParams<Params>();

  const storeDispatch = useStoreDispatch();
  useEffect(() => {
    storeDispatch({ type: StoreAction.HideUIChrome });
    return () => storeDispatch({ type: StoreAction.ShowUIChrome });
  }, [ storeDispatch ]);

  return (
    <div className={css.base}>
      <div className={css.barContainer}>
        <TaskBar
          handleViewLogsClick={() => setPageView(PageView.TASK_LOGS)}
          id={taskId}
          name={taskName}
          resourcePool={taskResourcePool}
          type={taskType}
        />
      </div>
      <div className={css.contentContainer}>
        {pageView === PageView.IFRAME && (
          <iframe
            allowFullScreen
            src={decodeURIComponent(taskUrl)}
            title="Interactive Task"
          />
        )}
        {pageView === PageView.TASK_LOGS && (
          <TaskLogs
            headerComponent={<div />}
            taskId={taskId}
            taskType={taskType}
            onCloseLogs={() => setPageView(PageView.IFRAME)}
          />
        )
        }
      </div>
    </div>
  );
};

export default InteractiveTask;
