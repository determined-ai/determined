import React, { useEffect, useState } from 'react';
import { Helmet } from 'react-helmet-async';
import { useParams, useSearchParams } from 'react-router-dom';

import TaskBar from 'components/TaskBar';
import { getTask } from 'services/api';
import useUI from 'shared/contexts/stores/UI';
import { ValueOf } from 'shared/types';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { CommandState, CommandType } from 'types';
import handleError, { handleWarning } from 'utils/error';

import css from './InteractiveTask.module.scss';
import TaskLogs from './TaskLogs';

type Params = {
  taskId: string;
  taskName: string;
  taskResourcePool: string;
  taskType: CommandType;
  taskUrl: string;
};

const PageView = {
  IFRAME: 'Iframe',
  TASK_LOGS: 'Task Logs',
} as const;

type PageView = ValueOf<typeof PageView>;

const DEFAULT_PAGE_TITLE = 'Tasks - Determined';

const getTitleState = (commandState?: CommandState, taskName?: string): string => {
  if (!commandState) {
    return DEFAULT_PAGE_TITLE;
  }
  const commandStateTitleMap = {
    [CommandState.Waiting]: 'Waiting',
    [CommandState.Pulling]: 'Pulling',
    [CommandState.Queued]: 'Queued',
    [CommandState.Running]: taskName || DEFAULT_PAGE_TITLE,
    [CommandState.Terminating]: 'Terminating',
    [CommandState.Terminated]: 'Terminated',
    [CommandState.Starting]: 'Starting',
  };
  const title = commandStateTitleMap[commandState];
  if (commandState !== CommandState.Terminated && commandState !== CommandState.Running) {
    return title + '...';
  }
  return title;
};

export const InteractiveTask: React.FC = () => {
  const [pageView, setPageView] = useState<PageView>(PageView.IFRAME);
  const {
    taskId: tId,
    taskName: tName,
    taskResourcePool: tResourcePool,
    taskType: tType,
    taskUrl: tUrl,
  } = useParams<Params>();
  const [taskState, setTaskState] = useState<CommandState>();
  const [searchParams] = useSearchParams();
  const currentSlotsExceeded = searchParams.get('currentSlotsExceeded');
  const { actions: uiActions, ui } = useUI();

  const slotsExceeded = currentSlotsExceeded ? currentSlotsExceeded === 'true' : false;

  const taskId = tId ?? '';
  const taskName = tName ?? '';
  const taskResourcePool = tResourcePool ?? '';
  const taskType = tType as CommandType;
  const taskUrl = tUrl ?? '';

  useEffect(() => {
    uiActions.hideChrome();
    return uiActions.showChrome;
  }, [uiActions]);

  useEffect(() => {
    if (slotsExceeded) {
      handleWarning({
        level: ErrorLevel.Warn,
        publicMessage:
          'The requested job requires more slots than currently available. You may need to increase cluster resources in order for the job to run.',
        publicSubject: 'Current Slots Exceeded',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [slotsExceeded]);

  useEffect(() => {
    const queryTask = setInterval(async () => {
      try {
        const response = await getTask({ taskId });
        if (response?.allocations?.length) {
          const lastRunState = response.allocations[0]?.state;
          setTaskState(lastRunState);
          if (lastRunState === CommandState.Terminated) {
            clearInterval(queryTask);
          }
        }
      } catch (e) {
        handleError(e, {
          publicMessage: 'failed querying for command state',
          silent: true,
        });
        clearInterval(queryTask);
      }
    }, 2000);
    return () => clearInterval(queryTask);
  }, [taskId]);

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
            <iframe allowFullScreen src={decodeURIComponent(taskUrl)} title="Interactive Task" />
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
