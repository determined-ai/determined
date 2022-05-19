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

enum PageViews {
  IFRAME= 'Iframe',
  TASK_LOGS = 'Task Logs'
}

export const InteractiveTask: React.FC = () => {

  const [pageView, setPageView] = useState<PageViews>(PageViews.IFRAME)
  const { taskId, taskName, taskResourcePool, taskUrl, taskType } = useParams<Params>();

  const storeDispatch = useStoreDispatch();
  useEffect(() => {
    storeDispatch({ type: StoreAction.HideUIChrome });
    return () => storeDispatch({ type: StoreAction.ShowUIChrome });
  }, [ storeDispatch ]);


    return (
    <div className={css.base}>
      <div className={css.barContainer}>
        <TaskBar id={taskId} name={taskName} resourcePool={taskResourcePool} type={taskType} handleViewLogsClick={() => setPageView(PageViews.TASK_LOGS)}/>
      </div>
      {pageView == PageViews.IFRAME && (
         <div className={css.frameContainer}>
         <iframe
           allowFullScreen
           src={decodeURIComponent(taskUrl)}
           title="Interactive Task"
         />
       </div>
      )}
      {pageView == PageViews.TASK_LOGS && (
      <div className={css.contentContainer}>
        <TaskLogs taskId={taskId} taskType={taskType} onCloseLogs={() => setPageView(PageViews.IFRAME)}/>
      </div>
      )
      }
    </div>
  );
};

export default InteractiveTask;
