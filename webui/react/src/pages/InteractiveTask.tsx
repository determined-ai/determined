import { not } from 'fp-ts/lib/Predicate';
import queryString from 'query-string';
import PageMessage from 'components/PageMessage';
import React, {useEffect} from 'react';
import { StoreAction, useStore, useStoreDispatch } from 'contexts/Store';

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
        <PageMessage title={"Task not found"}>
        <div>
          <div>Missing Task Url</div>
        </div>
      </PageMessage>
      )
    } 
    return (
        <iframe 
        src={decodeURIComponent(taskUrl)}
        width="100%"
        height="100%"
        title="Interactive Task"
        allowFullScreen
        >
        </iframe>
    )
  }
  
export default InteractiveTask;
