import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Modal } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';
import { useHistory, useParams } from 'react-router';

import LogViewerTimestamp, { TAIL_SIZE } from 'components/LogViewerTimestamp';
import Message, { MessageType } from 'components/Message';
import Spinner from 'components/Spinner';
import handleError, { ErrorType } from 'ErrorHandler';
import useRestApi from 'hooks/useRestApi';
import { paths } from 'routes/utils';
import { getTrialDetails } from 'services/api';
import { detApi } from 'services/apiConfig';
import { jsonToTrialLog } from 'services/decoder';
import { TrialDetailsParams } from 'services/types';
import { TrialDetails } from 'types';
import { downloadTrialLogs } from 'utils/browser';

import TrialLogFilters, { TrialLogFiltersInterface } from './TrialLogs/TrialLogFilters';

interface Params {
  experimentId?: string;
  trialId: string;
}

const TrialLogs: React.FC = () => {
  const { experimentId: experimentIdParam, trialId: trialIdParam } = useParams<Params>();
  const history = useHistory();
  const trialId = parseInt(trialIdParam);
  const [ downloadModal, setDownloadModal ] = useState<{ destroy: () => void }>();
  const [ trial ] = useRestApi<TrialDetailsParams, TrialDetails>(getTrialDetails, { id: trialId });

  const title = `Trial ${trialId} Logs`;
  const experimentId = trial.data?.experimentId;

  const fetchLogAfter =
    useCallback((filters: TrialLogFiltersInterface, canceler: AbortController) => {
      return detApi.StreamingExperiments.determinedTrialLogs(
        trialId,
        0,
        TAIL_SIZE,
        false,
        filters.agentIds,
        filters.containerIds,
        filters.rankIds,
        filters.levels,
        filters.stdtypes,
        filters.sources,
        filters.timestampBefore ? filters.timestampBefore.toDate() : undefined,
        filters.timestampAfter ? filters.timestampAfter.toDate() : undefined,
        'ORDER_BY_ASC',
        { signal: canceler.signal },
      );
    }, [ trialId ]);

  const fetchLogBefore =
    useCallback((filters: TrialLogFiltersInterface, canceler: AbortController) => {
      return detApi.StreamingExperiments.determinedTrialLogs(
        trialId,
        0,
        TAIL_SIZE,
        false,
        filters.agentIds,
        filters.containerIds,
        filters.rankIds,
        filters.levels,
        filters.stdtypes,
        filters.sources,
        filters.timestampBefore ? filters.timestampBefore.toDate() : undefined,
        filters.timestampAfter ? filters.timestampAfter.toDate() : undefined,
        'ORDER_BY_DESC',
        { signal: canceler.signal },
      );
    }, [ trialId ]);

  const fetchLogFilter = useCallback((canceler: AbortController) => {
    return detApi.StreamingExperiments.determinedTrialLogsFields(
      trialId,
      true,
      { signal: canceler.signal },
    );
  }, [ trialId ]);

  const fetchLogTail =
    useCallback((filters: TrialLogFiltersInterface, canceler: AbortController) => {
      return detApi.StreamingExperiments.determinedTrialLogs(
        trialId,
        0,
        0,
        true,
        filters.agentIds,
        filters.containerIds,
        filters.rankIds,
        filters.levels,
        filters.stdtypes,
        filters.sources,
        undefined,
        new Date(),
        'ORDER_BY_ASC',
        { signal: canceler.signal },
      );
    }, [ trialId ]);

  const handleDownloadConfirm = useCallback(async () => {
    if (downloadModal) {
      downloadModal.destroy();
      setDownloadModal(undefined);
    }

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
      history.replace(paths.trialLogs(trialId, experimentId));
    }
  }, [ experimentId, experimentIdParam, history, trialId ]);

  if (!experimentId || !trialId) {
    return <Spinner spinning={true} />;
  }

  if (trial.errorCount > 0 && !trial.isLoading) {
    return <Message title={`Unable to find Trial ${trialId}`} type={MessageType.Warning} />;
  }

  return (
    <LogViewerTimestamp
      fetchToLogConverter={jsonToTrialLog}
      FilterComponent={TrialLogFilters}
      pageProps={{
        breadcrumb: [
          { breadcrumbName: 'Experiments', path: paths.experimentList() },
          {
            breadcrumbName: `Experiment ${experimentId}`,
            path: paths.experimentDetails(experimentId),
          },
          {
            breadcrumbName: `Trial ${trialId}`,
            path: paths.trialDetails(trialId, experimentId),
          },
          { breadcrumbName: 'Logs', path: '#' },
        ],
        title,
      }}
      onDownloadClick={handleDownloadLogs}
      onFetchLogAfter={fetchLogAfter}
      onFetchLogBefore={fetchLogBefore}
      onFetchLogFilter={fetchLogFilter}
      onFetchLogTail={fetchLogTail}
    />
  );
};

export default TrialLogs;
