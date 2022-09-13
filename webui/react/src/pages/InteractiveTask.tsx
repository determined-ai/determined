import React, { useEffect, useState } from 'react';
import { Helmet } from 'react-helmet-async';
import { useParams } from 'react-router-dom';

import TaskBar from 'components/TaskBar';
import { useStore, useStoreDispatch } from 'contexts/Store';
import { getTask } from 'services/api';
import { StoreActionUI } from 'shared/contexts/UIStore';
import { CommandState, CommandType } from 'types';
import handleError from 'utils/error';

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

const DEFAULT_PAGE_TITLE = 'Tasks - Determined';

const getTitleState = (commandState?: CommandState, taskName?: string): string => {
  if (!commandState){
    return DEFAULT_PAGE_TITLE;
  }
  const commandStateTitleMap = {
    [CommandState.Pending]: 'Pending',
    [CommandState.Assigned]: 'Assigned',
    [CommandState.Pulling]: 'Pulling',
    [CommandState.Running]: taskName || DEFAULT_PAGE_TITLE,
    [CommandState.Terminating]: 'Terminating',
    [CommandState.Terminated]: 'Terminated',
    [CommandState.Starting]: 'Starting',
  };
  const title = commandStateTitleMap[commandState];
  if (commandState !== CommandState.Terminated && commandState !== CommandState.Running){
    return title + '...';
  }
  return title;

};

export const InteractiveTask: React.FC = () => {

  const [ pageView, setPageView ] = useState<PageView>(PageView.IFRAME);
  const { taskId, taskName, taskResourcePool, taskUrl, taskType } = useParams<Params>();
  const [ taskState, setTaskState ] = useState<CommandState>();
  const storeDispatch = useStoreDispatch();
  const { ui } = useStore();

  useEffect(() => {
    storeDispatch({ type: StoreActionUI.HideUIChrome });
    return () => storeDispatch({ type: StoreActionUI.ShowUIChrome });
  }, [ storeDispatch ]);

  useEffect(() => {
    const queryTask = setInterval(async () => {
      try {
        const response = await getTask({ taskId });
        if (response?.allocations?.length) {
          const lastRunState = response.allocations[0]?.state;
          setTaskState(lastRunState);
          if (lastRunState === CommandState.Terminated){
            clearInterval(queryTask);
          }
        }
      } catch (e) {
        handleError(
          {
            error: e,
            message: 'failed querying for command state',
            silent: true,
          },
        );
        clearInterval(queryTask);
      }
    }, 2000);
    return () => clearInterval(queryTask);
  }, [ taskId ]);

  const title = ui.isPageHidden ? getTitleState(taskState, taskName) : taskName;

  return (
    <>
      <Helmet defer={false}>
        <title>{title}</title>
      </Helmet>
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
          )}
        </div>
      </div>
    </>
  );
};

export default InteractiveTask;
