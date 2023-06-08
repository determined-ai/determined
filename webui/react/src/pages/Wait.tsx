import React, { useEffect, useState } from 'react';
import { useParams, useSearchParams } from 'react-router-dom';

import Badge, { BadgeType } from 'components/Badge';
import PageMessage from 'components/PageMessage';
import Spinner from 'components/Spinner/Spinner';
import { terminalCommandStates } from 'constants/states';
import { serverAddress } from 'routes/utils';
import { getTask } from 'services/api';
import useUI from 'stores/contexts/UI';
import { CommandState } from 'types';
import { ErrorType } from 'utils/error';
import handleError from 'utils/error';
import { capitalize } from 'utils/string';
import { WaitStatus } from 'utils/wait';

import css from './Wait.module.scss';

type Params = {
  taskId: string;
  taskType: string;
};

const Wait: React.FC = () => {
  const {
    actions: { showChrome, hideChrome },
  } = useUI();
  const [searchParams] = useSearchParams();
  const { taskType } = useParams<Params>();
  const [waitStatus, setWaitStatus] = useState<WaitStatus>();
  const serviceAddr = searchParams.get('serviceAddr');

  const capitalizedTaskType = capitalize(taskType ?? '');
  const isLoading = !waitStatus || !terminalCommandStates.has(waitStatus.state);

  let message = `Waiting for ${capitalizedTaskType} ...`;
  if (!serviceAddr) {
    message = 'Missing required parameters.';
  } else if (waitStatus && terminalCommandStates.has(waitStatus.state)) {
    message = `${capitalizedTaskType} has been terminated.`;
  } else if (
    capitalizedTaskType === 'Tensor-board' &&
    waitStatus &&
    waitStatus?.state === CommandState.Waiting
  ) {
    message = `Waiting for ${capitalizedTaskType} metrics step to be completed.`;
  }

  useEffect(() => {
    hideChrome();
    return showChrome;
  }, [hideChrome, showChrome]);

  const handleTaskError = (e: Error) => {
    handleError(e, {
      publicMessage:
        'Failed while waiting for command to be ready. This may be caused by not having related permissions',
      silent: false,
      type: ErrorType.Server,
    });
  };

  useEffect(() => {
    if (!serviceAddr) return;
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
        if (CommandState.Terminated === lastRun.state) {
          clearInterval(ival);
        } else if (lastRun.isReady) {
          clearInterval(ival);
          window.location.assign(serverAddress(serviceAddr));
        }
        // TODO: use task.endTime to determine if the task is terminated.
        setWaitStatus(lastRun);
      } catch (e) {
        handleTaskError(e as Error);
      }
    }, 1000);
  }, [serviceAddr]);

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
