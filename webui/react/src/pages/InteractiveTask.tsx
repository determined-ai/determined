import React, { useEffect } from 'react';
import { useParams } from 'react-router-dom';
import TaskBar from 'components/TaskBar';
import { StoreAction, useStoreDispatch } from 'contexts/Store';

import css from './InteractiveTask.module.scss';
import { CommandType } from 'types';

interface Params {
  taskId: string;
  taskUrl: string;
  taskResourcePool: string;
  taskName: string;
  taskType: CommandType
}

export const InteractiveTask: React.FC = () => {

  const { taskId, taskName, taskResourcePool, taskUrl, taskType } = useParams<Params>();

  const storeDispatch = useStoreDispatch();
  useEffect(() => {
    storeDispatch({ type: StoreAction.HideUIChrome });
    return () => storeDispatch({ type: StoreAction.ShowUIChrome });
  }, [ storeDispatch ]);

  console.log(taskId, taskType, taskName, taskResourcePool, taskUrl);

  return (
    <div className={css.base}>
      <div className={css.barContainer}>
        <TaskBar type={taskType} resourcePool={taskResourcePool} id={taskId} name={taskName} />
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
