import Checkbox from 'hew/Checkbox';
import CodeSample from 'hew/CodeSample';
import Input from 'hew/Input';
import Message from 'hew/Message';
import Spinner from 'hew/Spinner';
import Tooltip from 'hew/Tooltip';
import useConfirm from 'hew/useConfirm';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { throttle } from 'throttle-debounce';

import useUI from 'components/ThemeProvider';
import useFeature from 'hooks/useFeature';
import { useSettings } from 'hooks/useSettings';
import { DateString, decode, optional } from 'ioTypes';
import { serverAddress } from 'routes/utils';
import { detApi } from 'services/apiConfig';
import { mapV1LogsResponse } from 'services/decoder';
import { readStream } from 'services/utils';
import { ExperimentBase, TrialDetails, TrialLog } from 'types';
import { downloadTrialLogs } from 'utils/browser';
import handleError, { ErrorType } from 'utils/error';
import mergeAbortControllers from 'utils/mergeAbortControllers';

import LogViewer, { FetchConfig, FetchDirection, FetchType } from './LogViewer';
import LogViewerSelect, { Filters } from './LogViewerSelect';
import { Settings, settingsConfigForTrial } from './LogViewerSelect.settings';
import css from './TrialDetailsLogs.module.scss';

export interface Props {
  experiment: ExperimentBase;
  trial?: TrialDetails;
}

type OrderBy = 'ORDER_BY_UNSPECIFIED' | 'ORDER_BY_ASC' | 'ORDER_BY_DESC';

const TrialDetailsLogs: React.FC<Props> = ({ experiment, trial }: Props) => {
  const { ui } = useUI();
  const [filterOptions, setFilterOptions] = useState<Filters>({});
  const [searchOn, setSearchOn] = useState<boolean>(false);
  const [searchInput, setSearchInput] = useState<string>('');
  const [searchResults, setSearchResults] = useState<TrialLog[]>([]);
  const [selectedLog, setSelectedLog] = useState<TrialLog>();
  const confirm = useConfirm();
  const canceler = useRef(new AbortController());
  const f_flat_runs = useFeature().isOn('flat_runs');

  const trialSettingsConfig = useMemo(() => settingsConfigForTrial(trial?.id || -1), [trial?.id]);
  const { resetSettings, settings, updateSettings } = useSettings<Settings>(trialSettingsConfig);

  const filterValues: Filters = useMemo(
    () => ({
      agentIds: settings.agentId,
      containerIds: settings.containerId,
      enableRegex: settings.enableRegex,
      levels: settings.level,
      rankIds: settings.rankId,
      searchText: settings.searchText,
    }),
    [settings],
  );

  useEffect(() => {
    settings.searchText?.length && setSearchOn(true);
  }, [settings.searchText]);

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
        enableRegex: filters.enableRegex,
        level: filters.levels,
        rankId: filters.rankIds,
        searchText: filters.searchText,
      });
    },
    [updateSettings],
  );

  const handleFilterReset = useCallback(() => {
    resetSettings();
    setSearchResults([]);
    setSearchInput('');
    setSelectedLog(undefined);
  }, [resetSettings]);

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
    (config: FetchConfig, type: FetchType, searchText?: string) => {
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
        searchText,
        settings.enableRegex,
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
      searchOn={searchOn}
      showSearch={false}
      values={filterValues}
      onChange={handleFilterChange}
      onClickSearch={() => setSearchOn((prev) => !prev)}
      onReset={handleFilterReset}
    />
  );

  const throttledChangeSearch = useMemo(
    () =>
      throttle(
        500,
        (s: string) => {
          updateSettings({ ...settings, searchText: s });
        },
        { noLeading: true },
      ),
    [updateSettings, settings],
  );

  useEffect(() => {
    return () => {
      throttledChangeSearch.cancel();
    };
  }, [throttledChangeSearch]);

  const onSearchChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setSearchInput(e.target.value);
      throttledChangeSearch(e.target.value);
      if (!e.target.value) {
        setSearchResults([]);
        setSelectedLog(undefined);
      }
    },
    [throttledChangeSearch],
  );

  const processSearchResult = useCallback(
    (log: TrialLog) => {
      const key = settings.searchText;

      if (!key) return;
      const content = log.log;
      if (!content) return;
      const i = content.indexOf(key);
      if (i < 0) return;
      const numOfChar = 18;
      const j = i + key.length;
      let start = Math.max(i - numOfChar, 0);
      let end = Math.min(j + numOfChar, content.length);
      if (start > 0 && end === content.length) {
        start = Math.max(start - (numOfChar - end + j), 0);
      } else if (start === 0 && end < content.length) {
        end = Math.min(end + numOfChar - i + start, content.length);
      }
      const formated: TrialLog = {
        ...log,
        log: `${start > 0 ? '...' : ''}${content.slice(start, i)}<span style="background-color: #E7F7FF">${content.slice(i, j)}</span>${content.slice(j, end)}${end < content.length ? '...' : ''}`,
      };

      setSearchResults((prev) => [...prev, formated]);
    },
    [settings.searchText, setSearchResults],
  );

  useEffect(() => {
    if (settings.searchText) {
      setSearchResults([]);
      const newCanceler = new AbortController();
      canceler.current = newCanceler;
      readStream(
        handleFetch(
          {
            canceler: canceler.current,
            fetchDirection: FetchDirection.Newer,
            limit: 0,
          },
          FetchType.Initial,
          settings.searchText,
        ),
        (log) => processSearchResult(mapV1LogsResponse(log)),
      );
    }
    return () => canceler.current.abort();
  }, [settings.searchText, handleFetch, processSearchResult]);

  return (
    <div className={css.base}>
      {searchOn && (
        <div className={css.search}>
          <div className={css.header}>
            <Input
              allowClear
              placeholder="Search Logs..."
              value={searchInput || settings.searchText}
              onChange={onSearchChange}
            />
          </div>
          <Checkbox
            checked={settings.enableRegex}
            onChange={(e) => handleFilterChange({ enableRegex: e.target.checked })}>
            Regex
          </Checkbox>
          <div className={css.logContainer}>
            {searchResults.length > 0 ? (
              searchResults.map((log) => (
                <div key={log.id}>
                  <Tooltip content={log.message}>
                    <div
                      className={
                        selectedLog?.id === log.id ? [css.log, css.selected].join(' ') : css.log
                      }
                      dangerouslySetInnerHTML={{ __html: log.log! }}
                      onClick={() => setSelectedLog(log)}
                    />
                  </Tooltip>
                </div>
              ))
            ) : (
              <Message icon="warning" title="No logs to show. " />
            )}
          </div>
        </div>
      )}
      <Spinner conditionalRender spinning={!trial}>
        <LogViewer
          decoder={mapV1LogsResponse}
          selectedLog={selectedLog}
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
