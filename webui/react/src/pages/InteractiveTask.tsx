import queryString from 'query-string';
import React, { useEffect } from 'react';

import PageMessage from 'components/PageMessage';
import { StoreAction, useStoreDispatch } from 'contexts/Store';

interface Queries {
  taskUrl?: string;
}

export const InteractiveTask: React.FC = () => {

  const storeDispatch = useStoreDispatch();

  useEffect(() => {
    storeDispatch({ type: StoreAction.HideUIChrome });
    return () => storeDispatch({ type: StoreAction.ShowUIChrome });
  }, [ storeDispatch ]);

  const { taskUrl }: Queries = queryString.parse(location.search);

  if (!taskUrl){
    return (
      <PageMessage title={'Task not found'}>
        <div>
          <div>Missing Task Url</div>
        </div>
      </PageMessage>
    );
  }
  return (
    <iframe
      allowFullScreen
      height="100%"
      src={decodeURIComponent(taskUrl)}
      title="Interactive Task"
      width="100%"
    />
  );
};

export default InteractiveTask;
