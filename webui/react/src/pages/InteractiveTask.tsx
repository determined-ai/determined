
import React, { useEffect } from 'react';
import { StoreAction, useStoreDispatch } from 'contexts/Store';
import { useParams } from 'react-router-dom';

import TaskBar from 'components/TaskBar';


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

  console.log( taskId, taskName, taskResourcePool, taskUrl );
  



  return (
    <div>
    <TaskBar taskName={taskName} resourcePool={taskResourcePool} />
    <iframe
      allowFullScreen
      height="100%"
      src={decodeURIComponent(taskUrl)}
      title="Interactive Task"
      width="100%"
    />
    </div>
  );
};

export default InteractiveTask;
