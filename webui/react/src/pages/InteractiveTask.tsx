import React, { useEffect } from 'react';
import { useParams } from 'react-router-dom';

import TaskBar from 'components/TaskBar';
import { StoreAction, useStoreDispatch } from 'contexts/Store';
import { CommandType } from 'types';

import css from './InteractiveTask.module.scss';

interface Params {
  taskId: string;
  taskName: string;
  taskResourcePool: string;
  taskType: CommandType
  taskUrl: string;
}

export const InteractiveTask: React.FC = () => {

  const { taskId, taskName, taskResourcePool, taskUrl, taskType } = useParams<Params>();

  const storeDispatch = useStoreDispatch();
  useEffect(() => {
    storeDispatch({ type: StoreAction.HideUIChrome });
    return () => storeDispatch({ type: StoreAction.ShowUIChrome });
  }, [ storeDispatch ]);

  return (
    <div className={css.base}>
      <div className={css.barContainer}>
        <TaskBar id={taskId} name={taskName} resourcePool={taskResourcePool} type={taskType} />
      </div>
      <div className={css.frameContainer}>
        <iframe
          allowFullScreen
          src={decodeURIComponent(taskUrl)}
          title="Interactive Task"
        />
      </div>
    </div>
  );
};

export default InteractiveTask;
