import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Modal } from 'antd';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import LogViewerCore, { FetchConfig, FetchType } from 'components/LogViewerCore';
import { useStore } from 'contexts/Store';
import useSettings from 'hooks/useSettings';
import TrialLogFilters, { Filters } from 'pages/TrialDetails/Logs/TrialLogFilters';
import { serverAddress } from 'routes/utils';
import { detApi } from 'services/apiConfig';
import { jsonToTrialLog } from 'services/decoder';
import { consumeStream } from 'services/utils';
import { ExperimentBase, TrialDetails } from 'types';
import { downloadTrialLogs } from 'utils/browser';
import handleError, { ErrorType } from 'utils/error';

import css from './TrialDetailsLogs.module.scss';
import settingsConfig, { Settings } from './TrialDetailsLogs.settings';

export interface Props {
  experiment: ExperimentBase;
  trial: TrialDetails;
}

type OrderBy = 'ORDER_BY_UNSPECIFIED' | 'ORDER_BY_ASC' | 'ORDER_BY_DESC';

const TrialDetailsLogs: React.FC<Props> = ({ experiment, trial }: Props) => {
  const { ui } = useStore();
  const [ filterOptions, setFilterOptions ] = useState<Filters>({});
  const [ downloadModal, setDownloadModal ] = useState<{ destroy: () => void }>();

  const {
    resetSettings,
    settings,
    updateSettings,
  } = useSettings<Settings>(settingsConfig);

  const filterValues: Filters = useMemo(() => ({
    agentIds: settings.agentId,
    containerIds: settings.containerId,
    levels: settings.level,
    rankIds: settings.rankId,
  }), [ settings ]);

  const handleFilterChange = useCallback((filters: Filters) => {
    updateSettings({
      agentId: filters.agentIds,
      containerId: filters.containerIds,
      level: filters.levels,
      rankId: filters.rankIds,
    });
  }, [ updateSettings ]);

  const handleFilterReset = useCallback(() => resetSettings(), [ resetSettings ]);

  const handleDownloadConfirm = useCallback(async () => {
    if (downloadModal) {
      downloadModal.destroy();
      setDownloadModal(undefined);
    }

    try {
      await downloadTrialLogs(trial.id);
    } catch (e) {
      handleError(e, {
        publicMessage: `
          Failed to download trial ${trial.id} logs.
          If the problem persists please try our CLI "det trial logs ${trial.id}"
        `,
        publicSubject: 'Trial log download failed.',
        type: ErrorType.Ui,
      });
    }
  }, [ downloadModal, trial.id ]);

  const handleDownloadLogs = useCallback(() => {
    const modal = Modal.confirm({
      content: (
        <div>
          We recommend using the Determined CLI to download trial logs:
          <code className="block">
            det -m {serverAddress()} trial logs {trial.id} &gt;
            experiment_{experiment.id}_trial_{trial.id}_logs.txt
          </code>
        </div>
      ),
      icon: <ExclamationCircleOutlined />,
      okText: 'Proceed to Download',
      onOk: handleDownloadConfirm,
      title: `Confirm Download for Trial ${trial.id} Logs`,
      width: 640,
    });
    setDownloadModal(modal);
  }, [ experiment.id, handleDownloadConfirm, trial.id ]);

  const handleFetch = useCallback((config: FetchConfig, type: FetchType) => {
    const options = {
      follow: false,
      limit: config.limit,
      orderBy: 'ORDER_BY_UNSPECIFIED',
      timestampAfter: '',
      timestampBefore: '',
    };

    if (type === FetchType.Initial) {
      options.orderBy = config.isNewestFirst ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC';
    } else if (type === FetchType.Newer) {
      options.orderBy = 'ORDER_BY_ASC';
      if (config.offsetLog?.time) options.timestampAfter = config.offsetLog.time;
    } else if (type === FetchType.Older) {
      options.orderBy = 'ORDER_BY_DESC';
      if (config.offsetLog?.time) options.timestampBefore = config.offsetLog.time;
    } else if (type === FetchType.Stream) {
      options.follow = true;
      options.limit = 0;
      options.orderBy = 'ORDER_BY_ASC';
      options.timestampAfter = new Date().toISOString();
    }

    return detApi.StreamingExperiments.trialLogs(
      trial.id,
      options.limit,
      options.follow,
      settings.agentId,
      settings.containerId,
      settings.rankId,
      settings.level,
      undefined,
      undefined,
      options.timestampBefore ? new Date(options.timestampBefore) : undefined,
      options.timestampAfter ? new Date(options.timestampAfter) : undefined,
      options.orderBy as OrderBy,
      { signal: config.canceler.signal },
    );
  }, [ settings, trial.id ]);

  useEffect(() => {
    if (ui.isPageHidden) return;

    const canceler = new AbortController();

    consumeStream(
      detApi.StreamingExperiments.trialLogsFields(
        trial.id,
        true,
        { signal: canceler.signal },
      ),
      event => setFilterOptions(event as Filters),
    );

    return () => canceler.abort();
  }, [ trial.id, ui.isPageHidden ]);

  const trialLogFilters = (
    <div className={css.filters}>
      <TrialLogFilters
        options={filterOptions}
        values={filterValues}
        onChange={handleFilterChange}
        onReset={handleFilterReset}
      />
    </div>
  );

  return (
    <div className={css.base}>
      <LogViewerCore
        decoder={jsonToTrialLog}
        sortKey="time"
        title={trialLogFilters}
        onDownload={handleDownloadLogs}
        onFetch={handleFetch}
      />
    </div>
  );
};

export default TrialDetailsLogs;
