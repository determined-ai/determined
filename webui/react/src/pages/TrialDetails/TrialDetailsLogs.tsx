import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Modal } from 'antd';
import React, { useCallback, useState } from 'react';

import LogViewerTimestamp, { TAIL_SIZE } from 'components/LogViewerTimestamp';
import handleError, { ErrorType } from 'ErrorHandler';
import TrialLogFilters, { TrialLogFiltersInterface } from 'pages/TrialDetails/Logs/TrialLogFilters';
import { serverAddress } from 'routes/utils';
import { detApi } from 'services/apiConfig';
import { jsonToTrialLog } from 'services/decoder';
import { ExperimentBase, TrialDetails } from 'types';
import { downloadTrialLogs } from 'utils/browser';

export interface Props {
  experiment: ExperimentBase;
  trial: TrialDetails;
}

const TrialDetailsLogs: React.FC<Props> = ({ experiment, trial }: Props) => {
  const [ downloadModal, setDownloadModal ] = useState<{ destroy: () => void }>();

  const fetchLogAfter =
    useCallback((filters: TrialLogFiltersInterface, canceler: AbortController) => {
      return detApi.StreamingExperiments.determinedTrialLogs(
        trial.id,
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
    }, [ trial.id ]);

  const fetchLogBefore =
    useCallback((filters: TrialLogFiltersInterface, canceler: AbortController) => {
      return detApi.StreamingExperiments.determinedTrialLogs(
        trial.id,
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
    }, [ trial.id ]);

  const fetchLogFilter = useCallback((canceler: AbortController) => {
    return detApi.StreamingExperiments.determinedTrialLogsFields(
      trial.id,
      true,
      { signal: canceler.signal },
    );
  }, [ trial.id ]);

  const fetchLogTail =
    useCallback((filters: TrialLogFiltersInterface, canceler: AbortController) => {
      return detApi.StreamingExperiments.determinedTrialLogs(
        trial.id,
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
    }, [ trial.id ]);

  const handleDownloadConfirm = useCallback(async () => {
    if (downloadModal) {
      downloadModal.destroy();
      setDownloadModal(undefined);
    }

    try {
      await downloadTrialLogs(trial.id);
    } catch (e) {
      handleError({
        error: e,
        message: 'trial log download failed.',
        publicMessage: `
          Failed to download trial ${trial.id} logs.
          If the problem persists please try our CLI "det trial logs ${trial.id}"
        `,
        publicSubject: 'Download Failed',
        type: ErrorType.Ui,
      });
    }
  }, [ downloadModal, trial.id ]);

  const handleDownloadLogs = useCallback(() => {
    const modal = Modal.confirm({
      content: <div>
        We recommend using the Determined CLI to download trial logs:
        <code className="block">
          det -m {serverAddress()} trial logs {trial.id} &gt;
          experiment_{experiment.id}_trial_{trial.id}_logs.txt
        </code>
      </div>,
      icon: <ExclamationCircleOutlined />,
      okText: 'Proceed to Download',
      onOk: handleDownloadConfirm,
      title: `Confirm Download for Trial ${trial.id} Logs`,
      width: 640,
    });
    setDownloadModal(modal);
  }, [ experiment.id, handleDownloadConfirm, trial.id ]);

  return (
    <LogViewerTimestamp
      fetchToLogConverter={jsonToTrialLog}
      FilterComponent={TrialLogFilters}
      onDownloadClick={handleDownloadLogs}
      onFetchLogAfter={fetchLogAfter}
      onFetchLogBefore={fetchLogBefore}
      onFetchLogFilter={fetchLogFilter}
      onFetchLogTail={fetchLogTail}
    />
  );
};

export default TrialDetailsLogs;
