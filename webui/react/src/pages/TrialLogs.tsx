import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Modal } from 'antd';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { useParams } from 'react-router';
import { throttle } from 'throttle-debounce';

import LogViewer, { LogViewerHandles, TAIL_SIZE } from 'components/LogViewer';
import Message from 'components/Message';
import Page from 'components/Page';
import Spinner from 'components/Spinner';
import UI from 'contexts/UI';
import handleError, { ErrorType } from 'ErrorHandler';
import useRestApi from 'hooks/useRestApi';
import { detExperimentsStreamingApi, getTrialDetails } from 'services/api';
import * as DetSwagger from 'services/api-ts-sdk';
import { consumeStream } from 'services/apiBuilder';
import { jsonToTrialLog } from 'services/decoder';
import { TrialDetailsParams } from 'services/types';
import { Log, TrialDetails } from 'types';
import { downloadTrialLogs } from 'utils/browser';

interface Params {
  trialId: string;
}

const THROTTLE_TIME = 500;

const TrialLogs: React.FC = () => {
  const { trialId } = useParams<Params>();
  const id = parseInt(trialId);
  const title = `Trial ${id} Logs`;
  const setUI = UI.useActionContext();
  const logsRef = useRef<LogViewerHandles>(null);
  const [ offset, setOffset ] = useState(-TAIL_SIZE);
  const [ oldestId, setOldestId ] = useState(Number.MAX_SAFE_INTEGER);
  const [ oldestReached, setOldestReached ] = useState(false);
  const [ isDownloading, setIsDownloading ] = useState(false);
  const [ isLoading, setIsLoading ] = useState(true);
  const [ isIdInvalid, setIsIdInvalid ] = useState(false);
  const [ trial ] = useRestApi<TrialDetailsParams, TrialDetails>(getTrialDetails, { id });

  const handleScrollToTop = useCallback(() => {
    if (oldestReached) return;

    let buffer: Log[] = [];

    consumeStream<DetSwagger.V1TrialLogsResponse>(
      detExperimentsStreamingApi.determinedTrialLogs(id, offset - TAIL_SIZE, TAIL_SIZE),
      event => buffer.push(jsonToTrialLog(event)),
    ).then(() => {
      if (!logsRef.current) return;
      if (buffer.length === 0 || buffer[0].id === oldestId) {
        setOldestReached(true);
      } else {
        logsRef.current.addLogs(buffer, true);
        setOldestId(buffer[0].id);
        setOffset(prevOffset => prevOffset - TAIL_SIZE);
      }
      buffer = [];
    });
  }, [ id, offset, oldestId, oldestReached ]);

  const handleDownloadConfirm = useCallback(async () => {
    setIsDownloading(true);

    try {
      await downloadTrialLogs(id);
    } catch (e) {
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
    }

    setIsDownloading(false);
  }, [ id ]);

  const handleDownloadLogs = useCallback(() => {
    Modal.confirm({
      content: <div>
        We recommend using the Determined CLI to download trial logs:
        <code className="block">
          det trial logs {id} &gt; experiment_{trial.data?.experimentId}_trial_{trialId}_logs.txt
        </code>
      </div>,
      icon: <ExclamationCircleOutlined />,
      okText: 'Proceed to Download',
      onOk: handleDownloadConfirm,
      title: `Confirm Download for Trial ${id} Logs`,
      width: 640,
    });
  }, [ handleDownloadConfirm, id, trial.data, trialId ]);

  useEffect(() => setUI({ type: UI.ActionType.HideChrome }), [ setUI ]);

  useEffect(() => {
    if (trial.errorCount > 0 && !trial.isLoading) setIsIdInvalid(true);
  }, [ trial ]);

  useEffect(() => {
    if (!trial.hasLoaded) return;

    let buffer: Log[] = [];
    const throttleFunc = throttle(THROTTLE_TIME, () => {
      if (!logsRef.current) return;
      logsRef.current.addLogs(buffer);
      buffer = [];
      setIsLoading(false);
    });

    consumeStream<DetSwagger.V1TrialLogsResponse>(
      detExperimentsStreamingApi.determinedTrialLogs(id, -TAIL_SIZE, 0, true),
      event => {
        buffer.push(jsonToTrialLog(event));
        throttleFunc();
      },
    );

    return (): void => throttleFunc.cancel();
  }, [ id, trial.hasLoaded ]);

  if (isIdInvalid) {
    return (
      <Page id="page-error-message">
        <Message>Unable to find Trial {trialId}</Message>
      </Page>
    );
  }

  return (
    <Page id="trial-logs" maxHeight>
      <LogViewer
        disableLevel
        isDownloading={isDownloading}
        isLoading={isLoading}
        noWrap
        ref={logsRef}
        title={title}
        onDownload={handleDownloadLogs}
        onScrollToTop={handleScrollToTop} />
      {isLoading && <Spinner fullPage opaque />}
    </Page>
  );
};

export default TrialLogs;
