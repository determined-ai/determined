import React, { useCallback, useEffect, useRef, useState } from 'react';
import { useParams } from 'react-router';
import { debounce } from 'throttle-debounce';

import LogViewer, { LogViewerHandles } from 'components/LogViewer';
import Page from 'components/Page';
import UI from 'contexts/UI';
import handleError, { ErrorType } from 'ErrorHandler';
import usePolling from 'hooks/usePolling';
import { useRestApiSimple } from 'hooks/useRestApi';
import { getTrialLogs } from 'services/api';
import * as DetSwagger from 'services/api-ts-sdk';
import { consumeStream, experimentsApi } from 'services/apiBuilder';
import { jsonToTrialLog } from 'services/decoder';
import { Log } from 'types';
import { downloadTrialLogs } from 'utils/browser';

interface Params {
  trialId: string;
}

const TAIL_SIZE = 50;
const DEBOUNCE_TIME = 1000;

const TrialLogs: React.FC = () => {
  const { trialId } = useParams<Params>();
  const id = parseInt(trialId);
  const title = `Trial ${id} Logs`;
  const setUI = UI.useActionContext();
  const logsRef = useRef<LogViewerHandles>(null);
  const [ offset, setOffset ] = useState(-TAIL_SIZE);
  const [ oldestId, setOldestId ] = useState(Number.MAX_SAFE_INTEGER);
  const [ oldestReached, setOldestReached ] = useState(false);

  const handleScrollToTop = useCallback(() => {
    console.log('HANLDE SCROLL TO TOP');
    if (oldestReached) return;

    let buffer: Log[] = [];

    consumeStream<DetSwagger.V1TrialLogsResponse>(
      experimentsApi.determinedTrialLogs(id, offset - TAIL_SIZE, TAIL_SIZE),
      event => buffer.push(jsonToTrialLog(event)),
    ).then(() => {
      if (!logsRef.current) return;
      if (buffer.length === 0 || buffer[0].id === oldestId) {
        setOldestReached(true);
      } else {
        logsRef.current?.addLogs(buffer, true);
        setOldestId(buffer[0].id);
        setOffset(prevOffset => prevOffset - TAIL_SIZE);
      }
      buffer = [];
    });
  }, [ id, offset, oldestId, oldestReached ]);

  useEffect(() => setUI({ type: UI.ActionType.HideChrome }), [ setUI ]);

  useEffect(() => {
    let buffer: Log[] = [];
    const debounceFunc = debounce(DEBOUNCE_TIME, () => {
      if (!logsRef.current) return;
      console.log('new logs', buffer);
      logsRef.current?.addLogs(buffer);
      buffer = [];
    });

    consumeStream<DetSwagger.V1TrialLogsResponse>(
      experimentsApi.determinedTrialLogs(id, -TAIL_SIZE, 0, true),
      event => {
        console.log('single event', event);
        buffer.push(jsonToTrialLog(event));
        debounceFunc();
      },
    ).then(() => console.log('finished new log stream'));

    return (): void => debounceFunc.cancel();
  }, [ id ]);

  const downloadLogs = useCallback(() => {
    return downloadTrialLogs(id).catch(e => {
      handleError({
        error: e,
        message: 'trial log download failed.',
        publicMessage: `
        Failed to download trial ${id} logs.
        If the problem persists please try our CLI "det trial logs ${id}"
      `,
        publicSubject: 'Download Failed',
        type: ErrorType.Ui,
      });
    });
  }, [ id ]);

  return (
    <Page hideTitle maxHeight title={title}>
      <LogViewer
        disableLevel
        disableLineNumber
        isLoading={pollingLogsResponse.isLoading}
        noWrap
        ref={logsRef}
        title={title}
        onDownload={downloadLogs}
        onScrollToTop={handleScrollToTop} />
    </Page>
  );
};

export default TrialLogs;
