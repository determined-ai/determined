import Button from 'hew/Button';
import Checkbox from 'hew/Checkbox';
import ClipboardButton from 'hew/ClipboardButton';
import CodeSample from 'hew/CodeSample';
import Icon from 'hew/Icon';
import Input from 'hew/Input';
import { RecordKey } from 'hew/internal/types';
import LogViewerEntry, { MAX_DATETIME_LENGTH } from 'hew/LogViewer/LogViewerEntry';
import LogViewerSelect, { Filters } from 'hew/LogViewer/LogViewerSelect';
import { Settings, settingsConfigForTrial } from 'hew/LogViewer/LogViewerSelect.settings';
import Message from 'hew/Message';
import Row from 'hew/Row';
import Spinner from 'hew/Spinner';
import SplitPane, { Pane } from 'hew/SplitPane';
import useConfirm from 'hew/useConfirm';
import _ from 'lodash';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import screenfull from 'screenfull';
import { sprintf } from 'sprintf-js';
import { throttle } from 'throttle-debounce';

import useUI from 'components/ThemeProvider';
import useFeature from 'hooks/useFeature';
import { useSettings } from 'hooks/useSettings';
import { DateString, decode, optional } from 'ioTypes';
import { serverAddress } from 'routes/utils';
import { detApi } from 'services/apiConfig';
import { mapV1LogsResponse } from 'services/decoder';
import { readStream } from 'services/utils';
import { ExperimentBase, Log, TrialDetails, TrialLog } from 'types';
import { downloadTrialLogs } from 'utils/browser';
import handleError, { ErrorType } from 'utils/error';
import mergeAbortControllers from 'utils/mergeAbortControllers';
import { pluralizer } from 'utils/string';

import LogViewer, {
  FetchConfig,
  FetchDirection,
  FetchType,
  formatLogEntry,
  ViewerLog,
} from './LogViewer';
import css from './TrialDetailsLogs.module.scss';

export interface Props {
  experiment: ExperimentBase;
  trial?: TrialDetails;
}

type OrderBy = 'ORDER_BY_UNSPECIFIED' | 'ORDER_BY_ASC' | 'ORDER_BY_DESC';

const INITIAL_SEARCH_WIDTH = 420;

const TrialDetailsLogs: React.FC<Props> = ({ experiment, trial }: Props) => {
  const { ui } = useUI();
  const [filterOptions, setFilterOptions] = useState<Filters>({});
  const [logs, setLogs] = useState<ViewerLog[]>([]);
  const [searchOn, setSearchOn] = useState<boolean>(false);
  const [logViewerOn, setLogViewerOn] = useState<boolean>(true);
  const [searchInput, setSearchInput] = useState<string>('');
  const [searchResults, setSearchResults] = useState<TrialLog[]>([]);
  const [selectedLog, setSelectedLog] = useState<ViewerLog>();
  const [searchWidth, setSearchWidth] = useState(INITIAL_SEARCH_WIDTH);
  const confirm = useConfirm();
  const canceler = useRef(new AbortController());
  const container = useRef<HTMLDivElement>(null);
  const logsRef = useRef<HTMLDivElement>(null);
  const f_flat_runs = useFeature().isOn('flat_runs');

  const local = useRef({
    idSet: new Set<RecordKey>([]),
    isScrollReady: false as boolean,
  });

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
    (config: FetchConfig, type: FetchType, searchText?: string, enableRegex?: boolean) => {
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
        enableRegex,
        { signal },
      );
    },
    [settings.agentId, settings.containerId, settings.rankId, settings.level, trial?.id],
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
      showSearch={false}
      values={filterValues}
      onChange={handleFilterChange}
      onReset={handleFilterReset}
    />
  );

  const debouncedChangeSearch = useMemo(
    () =>
      throttle(
        500,
        (s: string) => {
          updateSettings({ searchText: s });
        },
        { noLeading: true },
      ),
    [updateSettings],
  );

  useEffect(() => {
    return () => {
      debouncedChangeSearch.cancel();
      // kinda gross but we want this to run only on unmount
      setSearchInput((s) => {
        updateSettings({ searchText: s });
        return s;
      });
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [debouncedChangeSearch]);

  const onSearchChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setSearchInput(e.target.value);
      debouncedChangeSearch(e.target.value);
      if (!e.target.value) {
        setSearchResults([]);
        setSelectedLog(undefined);
      }
    },
    [debouncedChangeSearch],
  );

  const formattedSearchResults = useMemo(() => {
    const key = settings.searchText;

    if (!key) return [];
    const formatted: ViewerLog[] = [];
    _.uniqBy(searchResults, (l) => l.id).forEach((l) => {
      const content = l.log;
      if (!content) return;
      if (settings.enableRegex) {
        try {
          new RegExp(key);
        } catch {
          return;
        }
      }

      const logEntry = formatLogEntry(l);

      const i = settings.enableRegex ? content.match(`${key}`)?.index : content.indexOf(key);
      if (_.isUndefined(i) || i < 0) return;
      const keyLen = settings.enableRegex ? content.match(`${key}`)?.[0].length || 0 : key.length;
      const j = i + keyLen;
      formatted.push({
        ...logEntry,
        message: `${content.slice(0, i)}<span class=${css.key}>${content.slice(i, j)}</span>${content.slice(j)}`,
      });
    });
    return formatted;
  }, [searchResults, settings.searchText, settings.enableRegex]);

  useEffect(() => {
    if (settings.searchText) {
      setSearchResults([]);
      setSelectedLog(undefined);
      readStream(
        handleFetch(
          {
            canceler: canceler.current,
            fetchDirection: FetchDirection.Newer,
            limit: 0,
          },
          FetchType.Initial,
          settings.searchText,
          settings.enableRegex,
        ),
        (log) => setSearchResults((prev) => [...prev, mapV1LogsResponse(log)]),
      );
    }
  }, [settings.searchText, handleFetch, settings.enableRegex, canceler]);

  const processLogs = useCallback((newLogs: Log[]) => {
    return newLogs
      .filter((log) => {
        const isDuplicate = local.current.idSet.has(log.id);
        const isTqdm = log.message.includes('\r');
        local.current.idSet.add(log.id);
        return !isDuplicate && !isTqdm;
      })
      .map((log) => formatLogEntry(log));
  }, []);

  const onSelectLog = useCallback(
    async (logEntry: ViewerLog) => {
      setSelectedLog(logEntry);
      setLogViewerOn(true);
      const index = logs.findIndex((l) => l.id === logEntry.id);
      if (index > -1) {
        // Selected log is already fetched. Just need to scroll to the place.
        return;
      }
      local.current = {
        idSet: new Set<RecordKey>([]),
        isScrollReady: true,
      };
      const bufferBefore: TrialLog[] = [];
      const bufferAfter: TrialLog[] = [];

      await readStream(
        handleFetch(
          {
            canceler: canceler.current,
            fetchDirection: FetchDirection.Older,
            limit: 100,
            offsetLog: logEntry,
          },
          FetchType.Older,
        ),
        (log) => bufferBefore.push(mapV1LogsResponse(log)),
      );
      await readStream(
        handleFetch(
          {
            canceler: canceler.current,
            fetchDirection: FetchDirection.Newer,
            limit: 100,
            offsetLog: logEntry,
          },
          FetchType.Newer,
        ),
        (log) => bufferAfter.push(mapV1LogsResponse(log)),
      );
      setLogs([...processLogs(bufferBefore.reverse()), ...processLogs(bufferAfter)]);
    },
    [handleFetch, logs, processLogs],
  );

  const renderSearch = useCallback(() => {
    const height = container.current?.getBoundingClientRect().height || 0;
    return (
      <div className={css.search} style={{ height: `${height}px` }}>
        <Checkbox
          checked={settings.enableRegex}
          onChange={(e) => updateSettings({ enableRegex: e.target.checked })}>
          Regex
        </Checkbox>
        <div className={css.logContainer}>
          {formattedSearchResults.length > 0 ? (
            formattedSearchResults.map((logEntry) => (
              <div
                className={css.log}
                key={logEntry.id}
                onClick={() => {
                  onSelectLog(logEntry);
                }}>
                <LogViewerEntry
                  formattedTime={logEntry.formattedTime}
                  htmlMessage={true}
                  key={logEntry.id}
                  level={logEntry.level}
                  message={logEntry.message}
                  style={{
                    backgroundColor:
                      logEntry.id === selectedLog?.id ? 'var(--theme-ix-active)' : 'transparent',
                  }}
                />
              </div>
            ))
          ) : (
            <Message icon="warning" title="No logs to show. " />
          )}
        </div>
      </div>
    );
  }, [
    settings.enableRegex,
    formattedSearchResults,
    container,
    selectedLog,
    updateSettings,
    onSelectLog,
  ]);

  const formatClipboardHeader = (log: Log): string => {
    const logEntry = formatLogEntry(log);
    const format = `%${MAX_DATETIME_LENGTH - 1}s `;
    const level = `<${logEntry.level || ''}>`;
    return sprintf(`%-9s ${format}`, level, logEntry.formattedTime);
  };

  const clipboardCopiedMessage = useMemo(() => {
    const linesLabel = pluralizer(logs.length, 'entry', 'entries');
    return `Copied ${logs.length} ${linesLabel}!`;
  }, [logs]);

  const getClipboardContent = useCallback(() => {
    return logs.map((log) => `${formatClipboardHeader(log)}${log.message || ''}`).join('\n');
  }, [logs]);

  const handleFullScreen = useCallback(() => {
    if (logsRef.current && screenfull.isEnabled) screenfull.toggle();
  }, []);

  const rightButtons = (
    <Row>
      <ClipboardButton copiedMessage={clipboardCopiedMessage} getContent={getClipboardContent} />
      <Button
        aria-label="Toggle Fullscreen Mode"
        icon={<Icon name="fullscreen" showTooltip title="Toggle Fullscreen Mode" />}
        onClick={handleFullScreen}
      />
      <Button
        aria-label="Download Logs"
        icon={<Icon name="download" showTooltip title="Download Logs" />}
        onClick={handleDownloadLogs}
      />
    </Row>
  );

  return (
    <div className={css.base} ref={container}>
      <Spinner conditionalRender spinning={!trial}>
        <div className={css.header}>
          <div className={css.filters}>
            <Input
              allowClear
              placeholder="Search Logs..."
              value={searchInput || settings.searchText}
              width={240}
              onChange={onSearchChange}
            />
            <Button
              type={searchOn ? 'primary' : 'default'}
              onClick={() => {
                setSearchOn((prev) => !prev);
                searchOn && setLogViewerOn(true);
              }}>
              <Icon name="search" showTooltip title={`${searchOn ? 'Close' : 'Open'} Search`} />
            </Button>
            <Button
              type={logViewerOn ? 'primary' : 'default'}
              onClick={() => searchOn && setLogViewerOn((prev) => !prev)}>
              <Icon
                name="list"
                showTooltip
                title={searchOn ? `${logViewerOn ? 'Close' : 'Open'} Logs` : ''}
              />
            </Button>
            {logFilters}
          </div>
          {rightButtons}
        </div>
        <SplitPane
          hidePane={searchOn && logViewerOn ? undefined : searchOn ? Pane.Right : Pane.Left}
          initialWidth={searchWidth || INITIAL_SEARCH_WIDTH}
          leftPane={renderSearch()}
          minimumWidths={{ left: 300, right: 300 }}
          rightPane={
            <LogViewer
              decoder={mapV1LogsResponse}
              local={local}
              logs={logs}
              logsRef={logsRef}
              selectedLog={selectedLog}
              serverAddress={serverAddress}
              setLogs={setLogs}
              onError={handleError}
              onFetch={handleFetch}
            />
          }
          onChange={(w) => setSearchWidth(w)}
        />
      </Spinner>
    </div>
  );
};

export default TrialDetailsLogs;
