import queryString from 'query-string';
import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';

import Badge, { BadgeType } from 'components/Badge';
import PageMessage from 'components/PageMessage';
import Spinner from 'components/Spinner';
import { terminalCommandStates } from 'constants/states';
import { StoreAction, useStoreDispatch } from 'contexts/Store';
import { serverAddress } from 'routes/utils';
import { getTask } from 'services/api';
import { capitalize } from 'shared/utils/string';
import { CommandState } from 'types';
import handleError from 'utils/error';
import { WaitStatus } from 'wait';

import { ErrorType } from '../shared/utils/error';

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

  const handleTaskError = (err: Error) => {
    handleError({
      error: err,
      message: 'failed while waiting for command to be ready',
      silent: false,
      type: ErrorType.Server,
    });
  };

  useEffect(() => {
    if (!eventUrl || !serviceAddr) return;
    const taskId = (serviceAddr.match(/[0-f-]+/) || ' ')[0];
    const ival = setInterval(async () => {
      try {
        const response = await getTask({ taskId });
        if (!response?.allocations?.length) {
          return;
        }
        const lastRun = response.allocations[0];
        if (!lastRun) {
          return;
        }
        if ([ CommandState.Terminated ].includes(lastRun.state)) {
          clearInterval(ival);
        } else if (lastRun.isReady) {
          clearInterval(ival);
          window.location.assign(serverAddress(serviceAddr));
        }
        setWaitStatus(lastRun);
      } catch (e) {
        handleTaskError(e);
      }
    }, 1000);
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
