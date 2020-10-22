import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Modal } from 'antd';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { useParams } from 'react-router';
import { throttle } from 'throttle-debounce';

import LogViewer, { LogViewerHandles, TAIL_SIZE } from 'components/LogViewer';
import Message from 'components/Message';
import UI from 'contexts/UI';
import handleError, { ErrorType } from 'ErrorHandler';
import useRestApi from 'hooks/useRestApi';
import { getTrialDetails } from 'services/api';
import { V1TrialLogsResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { jsonToTrialLog } from 'services/decoder';
import { TrialDetailsParams } from 'services/types';
import { consumeStream } from 'services/utils';
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
  const [ downloadModal, setDownloadModal ] = useState<{ destroy: () => void }>();
  const [ trial ] = useRestApi<TrialDetailsParams, TrialDetails>(getTrialDetails, { id });

  const handleScrollToTop = useCallback(() => {
    if (oldestReached) return;

    let buffer: Log[] = [];

    consumeStream<V1TrialLogsResponse>(
      detApi.StreamingExperiments.determinedTrialLogs(id, offset - TAIL_SIZE, TAIL_SIZE),
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
    if (downloadModal) {
      downloadModal.destroy();
      setDownloadModal(undefined);
    }

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
  }, [ downloadModal, id ]);

  const handleDownloadLogs = useCallback(() => {
    const modal = Modal.confirm({
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
    setDownloadModal(modal);
  }, [ handleDownloadConfirm, id, trial.data, trialId ]);

  useEffect(() => setUI({ type: UI.ActionType.HideChrome }), [ setUI ]);

  useEffect(() => {
    if (!trial.hasLoaded) return;

    let buffer: Log[] = [];
    const throttleFunc = throttle(THROTTLE_TIME, () => {
      if (!logsRef.current) return;
      logsRef.current.addLogs(buffer);
      buffer = [];
      setIsLoading(false);
    });

    consumeStream<V1TrialLogsResponse>(
      detApi.StreamingExperiments.determinedTrialLogs(id, -TAIL_SIZE, 0, true),
      event => {
        buffer.push(jsonToTrialLog(event));
        throttleFunc();
      },
    );

    return (): void => throttleFunc.cancel();
  }, [ id, trial.hasLoaded ]);

  if (trial.errorCount > 0 && !trial.isLoading) {
    return <Message title={`Unable to find Trial ${trialId}`} />;
  }

  return (
    <LogViewer
      disableLevel
      isDownloading={isDownloading}
      isLoading={isLoading}
      noWrap
      ref={logsRef}
      title={title}
      onDownload={handleDownloadLogs}
      onScrollToTop={handleScrollToTop} />
  );
};

export default TrialLogs;
