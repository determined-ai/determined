import queryString from 'query-string';
import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { w3cwebsocket as W3CWebSocket } from 'websocket';

import Badge, { BadgeType } from 'components/Badge';
import PageMessage from 'components/PageMessage';
import Spinner from 'components/Spinner';
import { StoreAction, useStoreDispatch } from 'contexts/Store';
import handleError, { ErrorType } from 'ErrorHandler';
import { serverAddress } from 'routes/utils';
import { CommandState } from 'types';
import { capitalize } from 'utils/string';
import { terminalCommandStates } from 'utils/types';
import { createWsUrl, WaitStatus } from 'wait';

import css from './Wait.module.scss';

interface Params {
  taskId: string;
  taskType: string;
}

interface Queries {
  eventUrl?: string;
  serviceAddr?: string;
}

const Wait: React.FC = () => {
  const storeDispatch = useStoreDispatch();
  const { taskType } = useParams<Params>();
  const [ waitStatus, setWaitStatus ] = useState<WaitStatus>();
  const { eventUrl, serviceAddr }: Queries = queryString.parse(location.search);

  const capitalizedTaskType = capitalize(taskType);
  const isLoading = !waitStatus || !terminalCommandStates.has(waitStatus.state);

  let message = `Waiting for ${capitalizedTaskType} ...`;
  if (!eventUrl || !serviceAddr) {
    message = 'Missing required parameters.';
  } else if (waitStatus && terminalCommandStates.has(waitStatus.state)) {
    message = `${capitalizedTaskType} has been terminated.`;
  }

  useEffect(() => {
    storeDispatch({ type: StoreAction.HideUIChrome });
    return () => storeDispatch({ type: StoreAction.ShowUIChrome });
  }, [ storeDispatch ]);

  const handleWsError = (err: Error) => {
    handleError({
      error: err,
      message: 'failed while waiting for command to be ready',
      silent: false,
      type: ErrorType.Server,
    });
  };

  useEffect(() => {
    if (!eventUrl || !serviceAddr) return;

    const url = createWsUrl(eventUrl);
    const client = new W3CWebSocket(url);

    client.onmessage = (messageEvent) => {
      if (typeof messageEvent.data !== 'string') return;
      const msg = JSON.parse(messageEvent.data);
      if (msg.state) {
        const state = msg.state;
        if (state === CommandState.Running && msg.is_ready) {
          setWaitStatus({ isReady: true, state: CommandState.Running });
          client.close();
          window.location.assign(serverAddress(serviceAddr));
        } else if (terminalCommandStates.has(state)) {
          setWaitStatus({ isReady: false, state });
          client.close();
        }
        setWaitStatus({ isReady: false, state });
      }
    };

    // client.onclose = handleWsError;
    client.onerror = handleWsError;
  }, [ eventUrl, serviceAddr ]);

  return (
    <PageMessage title={capitalizedTaskType}>
      <div className={css.base}>
        <div className={css.message}>{message}</div>
        {waitStatus && (
          <div className={css.state}>
            <Badge state={waitStatus?.state} type={BadgeType.State} />
          </div>
        )}
        <Spinner spinning={isLoading} />
      </div>
    </PageMessage>
  );
};

export default Wait;
