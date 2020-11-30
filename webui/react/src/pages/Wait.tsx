import queryString from 'query-string';
import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';

import Message from 'components/Message';
import Page from 'components/Page';
import Spinner from 'components/Spinner';
import handleError, { ErrorType } from 'ErrorHandler';
import { serverAddress } from 'routes/utils';
import { capitalize } from 'utils/string';
import { terminalCommandStates } from 'utils/types';
import { waitCommandReady, WaitStatus } from 'wait';

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

  useEffect(() => {
    if (!eventUrl || !serviceAddr) return;
    waitCommandReady(eventUrl)
      .then((status) => {
        setWaitStatus(status);
        if (status.isReady) window.location.assign(serverAddress(serviceAddr));
      })
      .catch((err: Error) => {
        handleError({
          error: err,
          message: 'failed while waiting for command to be ready',
          silent: false,
          type: ErrorType.Server,
        });
      });
  }, [ eventUrl, serviceAddr ]);

  let message: React.ReactNode;
  if ((!eventUrl || !serviceAddr)) {
    message = <Message title='Missing required parameters.' />;
  }
  if (waitStatus && terminalCommandStates.has(waitStatus.state)) {
    message = <Message title={`${taskTypeCap} has been terminated.`} />;
  }

  return (
    <Page id="wait" title={`Waiting for ${taskTypeCap} ${taskId}`}>
      {message ?
        message :
        <Spinner />
      }
    </Page>
  );
};

export default Wait;
