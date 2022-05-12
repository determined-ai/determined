import React, { useEffect } from 'react';
import { useParams } from 'react-router-dom';

import TaskBar from 'components/TaskBar';
import { StoreAction, useStoreDispatch } from 'contexts/Store';

import css from './InteractiveTask.module.scss';

interface Params {
  taskId: string;
  taskUrl: string;
  taskResourcePool: string;
  taskName: string;
}

export const InteractiveTask: React.FC = () => {

  const { taskId, taskName, taskResourcePool, taskUrl } = useParams<Params>();
  const storeDispatch = useStoreDispatch();
  useEffect(() => {
    storeDispatch({ type: StoreAction.HideUIChrome });
    return () => storeDispatch({ type: StoreAction.ShowUIChrome });
  }, [ storeDispatch ]);

  console.log(taskId, taskName, taskResourcePool, taskUrl);

  return (
    <div className={css.base}>
      <div className={css.barContainer}>
        <TaskBar resourcePool={taskResourcePool} taskId={taskId} taskName={taskName} />
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
