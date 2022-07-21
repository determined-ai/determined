import React, { useEffect, useState } from 'react';
import { Helmet } from 'react-helmet-async';
import { useParams } from 'react-router-dom';

import TaskBar from 'components/TaskBar';
import { StoreAction, useStore, useStoreDispatch } from 'contexts/Store';
import { CommandState, CommandType } from 'types';

import css from './InteractiveTask.module.scss';
import TaskLogs from './TaskLogs';

type Params = {
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

const getTitleState = (commandState?: CommandState): string => {
  if (!commandState){
    return DEFAULT_PAGE_TITLE;
  }
  const commandStateTitleMap = {
    [CommandState.Pending]: 'Pending',
    [CommandState.Assigned]: 'Assigned',
    [CommandState.Pulling]: 'Pulling',
    [CommandState.Running]: DEFAULT_PAGE_TITLE,
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

  const handleMessage = (event: MessageEvent) => {
    const messageFromSameOrigin = window.location.origin === event.origin;
    if (event?.data?.commandState && messageFromSameOrigin) {
      const commandState = event.data.commandState as CommandState;
      setTaskState(commandState);
    }
  };

  useEffect(() => {
    storeDispatch({ type: StoreAction.HideUIChrome });
    return () => storeDispatch({ type: StoreAction.ShowUIChrome });
  }, [ storeDispatch ]);

  useEffect(() => {
    window.addEventListener('message', handleMessage);

    return () => {
      window.removeEventListener('message', handleMessage);
    };
  }, [ storeDispatch ]);

  const title = ui.isPageHidden ? getTitleState(taskState) : DEFAULT_PAGE_TITLE;

  return (
    <>
      <Helmet defer={false}>
        <title>{title}</title>
      </Helmet>
      <div className={css.base}>
        <div className={css.barContainer}>
          <TaskBar
            handleViewLogsClick={() => setPageView(PageView.TASK_LOGS)}
            id={taskId!}
            name={taskName!}
            resourcePool={taskResourcePool!}
            type={taskType!}
          />
        </div>
        <div className={css.contentContainer}>
          {pageView === PageView.IFRAME && (
            <iframe
              allowFullScreen
              src={decodeURIComponent(taskUrl!)}
              title="Interactive Task"
            />
          )}
          {pageView === PageView.TASK_LOGS && (
            <TaskLogs
              headerComponent={<div />}
              taskId={taskId!}
              taskType={taskType!}
              onCloseLogs={() => setPageView(PageView.IFRAME)}
            />
          )}
        </div>
      </div>
    </>
  );
};

export default InteractiveTask;
