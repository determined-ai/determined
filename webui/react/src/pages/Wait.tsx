import queryString from 'query-string';
import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { w3cwebsocket as W3CWebSocket } from 'websocket';

import Page from 'components/Page';
import { IndicatorUnpositioned } from 'components/Spinner';
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
  const { taskId, taskType } = useParams<Params>();
  const [ waitStatus, setWaitStatus ] = useState<WaitStatus>();
  const { eventUrl, serviceAddr }: Queries = queryString.parse(location.search);

  const taskTypeCap = capitalize(taskType);

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
      if (msg.snapshot) {
        const state = msg.snapshot.state;
        if (state === 'RUNNING' && msg.snapshot.is_ready) {
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

  let message = `Waiting for ${taskTypeCap}`;
  if ((!eventUrl || !serviceAddr)) {
    message = 'Missing required parameters.';
  }
  if (waitStatus && terminalCommandStates.has(waitStatus.state)) {
    message = `${taskTypeCap} has been terminated.`;
  }

  return (
    <Page id="wait">
      <div className={css.base}>
        <div className={css.content}>
          <p>Service State: {waitStatus?.state}</p>
          <p>{message}</p>
          <IndicatorUnpositioned />
        </div>
      </div>
    </Page>
  );
};

export default Wait;
