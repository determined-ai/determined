import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Modal } from 'antd';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { useHistory, useParams } from 'react-router';
import { throttle } from 'throttle-debounce';

import LogViewer, { LogViewerHandles, TAIL_SIZE } from 'components/LogViewer';
import Message, { MessageType } from 'components/Message';
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

import TrialLogFilters, { TrialLogFiltersInterface } from './TrialLogs/TrialLogFilters';

interface Params {
  experimentId?: string;
  trialId: string;
}

const THROTTLE_TIME = 500;

const TrialLogs: React.FC = () => {
  const { experimentId: experimentIdParam, trialId: trialIdParam } = useParams<Params>();
  const history = useHistory();
  const trialId = parseInt(trialIdParam);
  const logsRef = useRef<LogViewerHandles>(null);
  const [ offset, setOffset ] = useState(-TAIL_SIZE);
  const [ oldestId, setOldestId ] = useState(Number.MAX_SAFE_INTEGER);
  const [ oldestReached, setOldestReached ] = useState(false);
  const [ isDownloading, setIsDownloading ] = useState(false);
  const [ isLoading, setIsLoading ] = useState(true);
  const [ downloadModal, setDownloadModal ] = useState<{ destroy: () => void }>();
  const [ trial ] = useRestApi<TrialDetailsParams, TrialDetails>(getTrialDetails, { id: trialId });
  const [ filter, setFilter ] = useState<TrialLogFiltersInterface>({});
  const [ historyCanceler ] = useState(new AbortController());

  const title = `Trial ${trialId} Logs`;
  const experimentId = trial.data?.experimentId;

  const handleScrollToTop = useCallback(() => {
    if (oldestReached) return;

    let buffer: Log[] = [];

    consumeStream<V1TrialLogsResponse>(
      detApi.StreamingExperiments.determinedTrialLogs(
        trialId,
        offset - TAIL_SIZE,
        TAIL_SIZE,
        false,
        filter.agentIds,
        filter.containerIds,
        filter.rankIds,
        filter.levels,
        filter.stdtypes,
        filter.sources,
        filter.timestampBefore ? filter.timestampBefore.toDate() : undefined,
        filter.timestampAfter ? filter.timestampAfter.toDate() : undefined,
        undefined,
        { signal: historyCanceler.signal },
      ),
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
  }, [ filter, historyCanceler, offset, oldestId, oldestReached, trialId ]);

  const handleDownloadConfirm = useCallback(async () => {
    if (downloadModal) {
      downloadModal.destroy();
      setDownloadModal(undefined);
    }

    setIsDownloading(true);

    try {
      await downloadTrialLogs(trialId);
    } catch (e) {
      handleError({
        error: e,
        message: 'trial log download failed.',
        publicMessage: `
          Failed to download trial ${trialId} logs.
          If the problem persists please try our CLI "det trial logs ${trialId}"
        `,
        publicSubject: 'Download Failed',
        type: ErrorType.Ui,
      });
    }

    setIsDownloading(false);
  }, [ downloadModal, trialId ]);

  const handleDownloadLogs = useCallback(() => {
    const modal = Modal.confirm({
      content: <div>
        We recommend using the Determined CLI to download trial logs:
        <code className="block">
          det trial logs {trialId} &gt; experiment_{experimentId}_trial_{trialId}_logs.txt
        </code>
      </div>,
      icon: <ExclamationCircleOutlined />,
      okText: 'Proceed to Download',
      onOk: handleDownloadConfirm,
      title: `Confirm Download for Trial ${trialId} Logs`,
      width: 640,
    });
    setDownloadModal(modal);
  }, [ experimentId, handleDownloadConfirm, trialId ]);

  useEffect(() => {
    // Experiment id does not exist in route, reroute to the one with it
    if (!experimentIdParam && experimentId) {
      history.replace(`/experiments/${experimentId}/trials/${trialId}/logs`);
    }
  }, [ experimentId, experimentIdParam, history, trialId ]);

  useEffect(() => {
    if (!trial.hasLoaded) return;

    let buffer: Log[] = [];
    const canceler = new AbortController();
    const throttleFunc = throttle(THROTTLE_TIME, () => {
      if (!logsRef.current) return;
      logsRef.current.addLogs(buffer);
      buffer = [];
      setIsLoading(false);
    });

    consumeStream<V1TrialLogsResponse>(
      detApi.StreamingExperiments.determinedTrialLogs(
        trialId,
        -TAIL_SIZE,
        0,
        true,
        filter.agentIds,
        filter.containerIds,
        filter.rankIds,
        filter.levels,
        filter.stdtypes,
        filter.sources,
        filter.timestampBefore ? filter.timestampBefore.toDate() : undefined,
        filter.timestampAfter ? filter.timestampAfter.toDate() : undefined,
        undefined,
        { signal: canceler.signal },
      ),
      event => {
        buffer.push(jsonToTrialLog(event));
        throttleFunc();
      },
    );

    return () => {
      canceler.abort();
      throttleFunc.cancel();
    };
  }, [ filter, trial.hasLoaded, trialId ]);

  // Clean up all running api calls before unmounting.
  useEffect(() => {
    return () => historyCanceler.abort();
  }, [ historyCanceler ]);

  if (trial.errorCount > 0 && !trial.isLoading) {
    return <Message title={`Unable to find Trial ${trialId}`} type={MessageType.Warning} />;
  }

  const experimentDetailPath = `/experiments/${trial.data?.experimentId}`;
  const trialDetailPath = `${experimentDetailPath}/trials/${trialId}`;

  const onFilterChange = (newFilters: TrialLogFiltersInterface) => {
    setFilter(newFilters);
    if (logsRef.current) {
      logsRef.current.clearLogs();
    }
  };

  return (
    <LogViewer
      disableLevel
      filterOptions={<TrialLogFilters
        filter={filter}
        trialId={trialId}
        onChange={onFilterChange}
      />}
      isDownloading={isDownloading}
      isLoading={isLoading}
      noWrap
      pageProps={{
        breadcrumb: [
          { breadcrumbName: 'Experiments', path: '/experiments' },
          {
            breadcrumbName: `Experiment ${trial.data?.experimentId}`,
            path: experimentDetailPath,
          },
          {
            breadcrumbName: `Trial ${trialId}`,
            path: trialDetailPath,
          },
          { breadcrumbName: 'Logs', path: '#' },
        ],
        title,
      }}
      ref={logsRef}
      onDownload={handleDownloadLogs}
      onScrollToTop={handleScrollToTop} />
  );
};

export default TrialLogs;
