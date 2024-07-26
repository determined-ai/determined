import CodeSample from 'hew/CodeSample';
import LogViewer, { FetchConfig, FetchDirection, FetchType } from 'hew/LogViewer/LogViewer';
import LogViewerSelect, { Filters } from 'hew/LogViewer/LogViewerSelect';
import { Settings, settingsConfigForTrial } from 'hew/LogViewer/LogViewerSelect.settings';
import Spinner from 'hew/Spinner';
import useConfirm from 'hew/useConfirm';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import useUI from 'components/ThemeProvider';
import useFeature from 'hooks/useFeature';
import { useSettings } from 'hooks/useSettings';
import { DateString, decode, optional } from 'ioTypes';
import { serverAddress } from 'routes/utils';
import { detApi } from 'services/apiConfig';
import { mapV1LogsResponse } from 'services/decoder';
import { readStream } from 'services/utils';
import { ExperimentBase, TrialDetails } from 'types';
import { downloadTrialLogs } from 'utils/browser';
import handleError, { ErrorType } from 'utils/error';
import mergeAbortControllers from 'utils/mergeAbortControllers';

import css from './TrialDetailsLogs.module.scss';

export interface Props {
  experiment: ExperimentBase;
  trial?: TrialDetails;
}

type OrderBy = 'ORDER_BY_UNSPECIFIED' | 'ORDER_BY_ASC' | 'ORDER_BY_DESC';

const TrialDetailsLogs: React.FC<Props> = ({ experiment, trial }: Props) => {
  const { ui } = useUI();
  const [filterOptions, setFilterOptions] = useState<Filters>({});
  const confirm = useConfirm();
  const canceler = useRef(new AbortController());
  const f_flat_runs = useFeature().isOn('flat_runs');

  const trialSettingsConfig = useMemo(() => settingsConfigForTrial(trial?.id || -1), [trial?.id]);
  const { resetSettings, settings, updateSettings } = useSettings<Settings>(trialSettingsConfig);

  const filterValues: Filters = useMemo(
    () => ({
      agentIds: settings.agentId,
      containerIds: settings.containerId,
      levels: settings.level,
      rankIds: settings.rankId,
      searchText: settings.searchText,
    }),
    [settings],
  );

  const handleFilterChange = useCallback(
    (filters: Filters) => {
      // request should have already been canceled when resetSettings updated
      // the settings hash
      if (Object.keys(filters).length === 0) return;

      canceler.current.abort();
      const newCanceler = new AbortController();
      canceler.current = newCanceler;

      updateSettings({
        agentId: filters.agentIds,
        containerId: filters.containerIds,
        level: filters.levels,
        rankId: filters.rankIds,
        searchText: filters.searchText,
      });
    },
    [updateSettings],
  );

  const handleFilterReset = useCallback(() => resetSettings(), [resetSettings]);

  const handleDownloadConfirm = useCallback(async () => {
    if (!trial?.id) return;

    try {
      await downloadTrialLogs(trial.id);
    } catch (e) {
      handleError(e, {
        publicMessage: `
          Failed to download ${f_flat_runs ? 'run' : 'trial'} ${trial.id} logs.
          If the problem persists please try our CLI "det trial logs ${trial.id}"
        `,
        publicSubject: `${f_flat_runs ? 'Run' : 'Trial'} log download failed.`,
        type: ErrorType.Ui,
      });
    }
  }, [f_flat_runs, trial?.id]);

  const handleDownloadLogs = useCallback(() => {
    if (!trial?.id) return;

    const code =
      `det -m ${serverAddress()} trial logs ${trial.id} > ` +
      `experiment_${experiment.id}_trial_${trial.id}_logs.txt`;
    confirm({
      content: (
        <div className={css.downloadConfirm}>
          <p>We recommend using the Determined CLI to download trial logs:</p>
          <CodeSample text={code} />
        </div>
      ),
      okText: 'Proceed to Download',
      onConfirm: handleDownloadConfirm,
      onError: handleError,
      size: 'medium',
      title: `Confirm Download for ${f_flat_runs ? 'Run' : 'Trial'} ${trial.id} Logs`,
    });
  }, [confirm, experiment.id, f_flat_runs, handleDownloadConfirm, trial?.id]);

  const handleFetch = useCallback(
    (config: FetchConfig, type: FetchType) => {
      const { signal } = mergeAbortControllers(config.canceler, canceler.current);

      const options = {
        follow: false,
        limit: config.limit,
        orderBy: 'ORDER_BY_UNSPECIFIED',
        timestampAfter: undefined as Date | string | undefined,
        timestampBefore: undefined as Date | string | undefined,
      };

      if (type === FetchType.Initial) {
        options.orderBy =
          config.fetchDirection === FetchDirection.Older ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC';
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
        trial?.id ?? 0,
        options.limit,
        options.follow,
        settings.agentId,
        settings.containerId,
        settings.rankId,
        settings.level,
        undefined,
        undefined,
        decode(optional(DateString), options.timestampBefore),
        decode(optional(DateString), options.timestampAfter),
        options.orderBy as OrderBy,
        settings.searchText,
        { signal },
      );
    },
    [settings, trial?.id],
  );

  useEffect(() => {
    if (ui.isPageHidden) return;
    if (!trial?.id) return;

    const fieldCanceler = new AbortController();

    const newCanceler = new AbortController();
    canceler.current = newCanceler;

    readStream(
      detApi.StreamingExperiments.trialLogsFields(trial.id, true, { signal: fieldCanceler.signal }),
      (event) => setFilterOptions(event as Filters),
    );

    return () => {
      canceler.current.abort();
      fieldCanceler.abort();
    };
  }, [trial?.id, ui.isPageHidden]);

  const logFilters = (
    <LogViewerSelect
      options={filterOptions}
      showSearch={true}
      values={filterValues}
      onChange={handleFilterChange}
      onReset={handleFilterReset}
    />
  );

  return (
    <div className={css.base}>
      <Spinner conditionalRender spinning={!trial}>
        <LogViewer
          decoder={mapV1LogsResponse}
          serverAddress={serverAddress}
          title={logFilters}
          onDownload={handleDownloadLogs}
          onError={handleError}
          onFetch={handleFetch}
        />
      </Spinner>
    </div>
  );
};

export default TrialDetailsLogs;
