import { Button, notification, Space, Tooltip } from 'antd';
import dayjs, { Dayjs } from 'dayjs';
import React, {
  Reducer, RefObject,
  useCallback, useEffect, useLayoutEffect, useMemo, useReducer, useRef, useState,
} from 'react';
import { ListChildComponentProps, ListOnItemsRenderedProps, VariableSizeList } from 'react-window';
import screenfull from 'screenfull';
import { sprintf } from 'sprintf-js';
import { throttle } from 'throttle-debounce';

import Icon from 'components/Icon';
import useGetCharMeasureInContainer from 'hooks/useGetCharMeasureInContainer';
import useScroll from 'hooks/useScroll';
import { LogViewerTimestampFilterComponentProp } from 'pages/TrialDetails/Logs/TrialLogFilters';
import { FetchArgs } from 'services/api-ts-sdk';
import { consumeStream } from 'services/utils';
import { LogLevel, TrialLog } from 'types';
import { formatDatetime } from 'utils/date';
import { ansiToHtml, copyToClipboard, toPixel, toRem } from 'utils/dom';

import css from './LogViewer.module.scss';
import { LogStoreAction, LogStoreActionType, logStoreReducer, ViewerLog } from './LogViewer.store';
import LogViewerLevel, { ICON_WIDTH } from './LogViewerLevel';
import Section from './Section';

export interface LogViewerTimestampFilter {
  timestampAfter?: Dayjs,   // exclusive of the specified date time
  timestampBefore?: Dayjs,  // inclusive of the specified date time
}

interface Props {
  FilterComponent?: React.ComponentType<LogViewerTimestampFilterComponentProp>,
  fetchToLogConverter: (data: unknown) => TrialLog,
  onDownloadClick?: () => void;
  onFetchLogAfter: (filters: LogViewerTimestampFilter, canceler: AbortController) => FetchArgs;
  onFetchLogBefore: (filters: LogViewerTimestampFilter, canceler: AbortController) => FetchArgs;
  onFetchLogFilter: (canceler: AbortController) => FetchArgs;
  onFetchLogTail: (filters: LogViewerTimestampFilter, canceler: AbortController) => FetchArgs;
}

export interface ListMeasure {
  height: number;
  width: number;
}

export const TAIL_SIZE = 100;

// Format the datetime to...
const DATETIME_PREFIX = '[';
const DATETIME_SUFFIX = ']';
const DATETIME_FORMAT = `[${DATETIME_PREFIX}]YYYY-MM-DD HH:mm:ss${DATETIME_SUFFIX}`;

// Max datetime size: DATETIME_FORMAT (plus 1 for a space suffix)
const MAX_DATETIME_LENGTH = 23;

const THROTTLE_TIME = 500;

enum DIRECTIONS {
  TOP_TO_BOTTOM, // show oldest logs and infinite-scroll newest ones at the bottom
  BOTTOM_TO_TOP, // show newest logs and infinite-scroll oldest ones at the top
}

const formatClipboardHeader = (log: TrialLog): string => {
  const format = `%${MAX_DATETIME_LENGTH - 1}s `;
  const level = `<${log.level || ''}>`;
  const datetime = log.time ? formatDatetime(log.time, DATETIME_FORMAT) : '';
  return sprintf(`%-9s ${format}`, level, datetime);
};

const useGetListMeasure = (container: RefObject<HTMLDivElement>): ListMeasure => {
  const containerPaddingInPixel = useMemo(() => {
    return toPixel(
      getComputedStyle(document.documentElement)
        .getPropertyValue('--theme-sizes-layout-medium'),
    );
  }, []);
  const scroll = useScroll(container);

  return {
    height: Math.max(
      0,
      (scroll?.viewHeight || 0) - (parseInt(containerPaddingInPixel) * 2),
    ),
    width: Math.max(
      0,
      (scroll?.viewWidth || 0) - (parseInt(containerPaddingInPixel) * 2),
    ),
  };
};

const LogViewerTimestamp: React.FC<Props> = ({
  fetchToLogConverter,
  FilterComponent,
  onDownloadClick,
  onFetchLogAfter,
  onFetchLogBefore,
  onFetchLogFilter,
  onFetchLogTail,
}: Props) => {
  const baseRef = useRef<HTMLDivElement>(null);
  const container = useRef<HTMLDivElement>(null);
  const listRef = useRef<VariableSizeList>(null);

  const charMeasures = useGetCharMeasureInContainer(container);
  const listMeasure = useGetListMeasure(container);

  const dateTimeWidth = charMeasures.width * MAX_DATETIME_LENGTH;

  const [ direction, setDirection ] = useState(DIRECTIONS.BOTTOM_TO_TOP);
  const [ filter, setFilter ] = useState<LogViewerTimestampFilter>({});
  const [ filterOptions, setFilterOptions ] = useState<LogViewerTimestampFilter>({});
  const [ isLastReached, setIsLastReached ] = useState<boolean>(false);
  const [ isLoading, setIsLoading ] = useState<boolean>(false);
  const [ isOnBottom, setIsOnBottom ] = useState<boolean>(false);
  const [ logs, logsDispatch ] = useReducer<Reducer<ViewerLog[], LogStoreAction>>(
    logStoreReducer,
    [],
  );

  const addLogs = useCallback((addedLogs: TrialLog[], isPrepend = false): void => {
    const newLogs = addedLogs
      .map(log => {
        // Try to handle TQDM gracefully, even though that isn't really possible from the rendering side
        if (log.message.includes('\r')) {
          // TQDM doesn't write new lines so fluent interprets each TQDM update as more characters on the
          // same line. This info won't be flushed to the logs backend until it hits some max size limit
          // where it gets flushed to the logs backend even though the line is still in-progress.
          // Then it shows up in the log backend as one huge line with \r separating TQDM updates from
          // each other. The best we can do is to show the most recent update rather than trying to show
          // all of them (which is the cause of the rendering bug where the log line is so huge, the
          // webui gives it a ridiculous height and it shows up as a blank section in the logs).
          //
          // Fluent is convinced that all of the TQDM updates are part of the same log line which share a
          // single timestamp. As more of the TQDM data makes it into the log store due to the max log line
          // size setting, the new TQDM updates won't be added to the bottom of the logs viewer, they
          // will be added to the same area as the first TQDM log line because they share the same timestamp.
          //
          // The correct solution to TQDM is to build a custom integration that has a newline behavior that
          // works well with Determined/Fluent - https://github.com/tqdm/tqdm#custom-integration
          //
          // But with this PR, TQDM will no longer ruin the entire log viewer, only TQDM will be weird.
          //
          // Fingers crossed no one is actually trying to use \r in their logs.
          const tqdm_lines = log.message.split('\r');
          // The last line might not be a full progress bar - use the second last line
          log.message = tqdm_lines[tqdm_lines.length - 2];
        }
        const formattedTime = log.time ? formatDatetime(log.time, DATETIME_FORMAT) : '';
        return { ...log, formattedTime };
      })
      .sort((logA, logB) => {
        const logATime = logA.time || '';
        const logBTime = logB.time || '';
        return logATime.localeCompare(logBTime);
      });
    if (newLogs.length === 0) return;

    logsDispatch({
      type: isPrepend ? LogStoreActionType.Prepend : LogStoreActionType.Append,
      value: newLogs,
    });

    // Restore the previous scroll position when prepending log entries.
    if (isPrepend) {
      listRef.current?.resetAfterIndex(0);
      listRef.current?.scrollToItem(newLogs.length + 1, 'start');
    }
  }, [ listRef, logsDispatch ]);

  const clearLogs = useCallback(() => {
    logsDispatch({ type: LogStoreActionType.Clear });
    listRef.current?.resetAfterIndex(0);
    setIsLastReached(false);
    setIsLoading(true);
  }, [ logsDispatch ]);

  const fetchAndAppendLogs =
    useCallback((direction: DIRECTIONS, filters: LogViewerTimestampFilter): AbortController => {
      const canceler = new AbortController();
      let fetchArgs = null;
      let isPrepend = false;

      if (direction === DIRECTIONS.BOTTOM_TO_TOP) {
        fetchArgs = onFetchLogBefore(filters, canceler);
        isPrepend = true;
      }

      if (direction === DIRECTIONS.TOP_TO_BOTTOM) {
        fetchArgs = onFetchLogAfter({
          ...filters,
          timestampAfter: filters.timestampAfter?.subtract(1, 'millisecond'),
        }, canceler);
        isPrepend = false;
      }

      if (fetchArgs) {
        let buffer: TrialLog[] = [];
        consumeStream(
          fetchArgs,
          event => {
            const logEntry = fetchToLogConverter(event);
            direction === DIRECTIONS.TOP_TO_BOTTOM
              ? buffer.unshift(logEntry) : buffer.push(logEntry);
          },
        ).then(() => {
          if (!canceler.signal.aborted && buffer.length < TAIL_SIZE) {
            setIsLastReached(true);
          }

          addLogs(buffer, isPrepend);

          setIsLoading(false);

          buffer = [];
        });
      }

      return canceler;
    }, [ addLogs, fetchToLogConverter, onFetchLogAfter, onFetchLogBefore ]);

  const getItemHeight = useCallback((index: number): number => {
    const log = logs[index];
    if (!log) {
      return charMeasures.height;
    }

    const maxCharPerLine = Math.floor(
      (listMeasure.width - ICON_WIDTH - dateTimeWidth) / charMeasures.width,
    );

    const lineCount = log.message
      .split('\n')
      .map(line => line.length > maxCharPerLine ? Math.ceil(line.length / maxCharPerLine) : 1)
      .reduce((acc, count) => acc + count, 0);

    return lineCount * charMeasures.height;
  }, [ charMeasures, dateTimeWidth, listMeasure, logs ]);

  const handleCopyToClipboard = useCallback(async () => {
    const content = logs.map(log => `${formatClipboardHeader(log)}${log.message || ''}`).join('\n');

    try {
      await copyToClipboard(content);
      const linesLabel = logs.length === 1 ? 'entry' : 'entries';
      notification.open({
        description: `${logs.length} ${linesLabel} copied to the clipboard.`,
        message: 'Available logs Copied',
      });
    } catch (e) {
      notification.warn({
        description: e.message,
        message: 'Unable to Copy to Clipboard',
      });
    }
  }, [ logs ]);

  const handleDownload = useCallback(() => {
    if (onDownloadClick) onDownloadClick();
  }, [ onDownloadClick ]);

  const handleEnableTailing = useCallback(() => {
    setDirection(DIRECTIONS.BOTTOM_TO_TOP);
    listRef.current?.scrollToItem(logs.length);
  }, [ listRef, logs.length ]);

  const handleFullScreen = useCallback(() => {
    if (baseRef.current && screenfull.isEnabled) screenfull.toggle();
  }, []);

  const handleScrollToTop = useCallback(() => {
    setDirection(DIRECTIONS.TOP_TO_BOTTOM);
  }, []);

  const onItemsRendered =
    useCallback(({ visibleStartIndex, visibleStopIndex }: ListOnItemsRenderedProps) => {
      setIsOnBottom(visibleStopIndex === (logs.length - 1));

      if (isLoading) return;
      if (isLastReached) return;
      if (!listRef?.current) return;

      const logTimes = logs.map(log => log.time).sort();

      // Fetch older log when direction=bottom_to_top and scroll is on top.
      if (direction === DIRECTIONS.BOTTOM_TO_TOP && visibleStartIndex === 0) {
        const canceler = fetchAndAppendLogs(direction, {
          ...filter,
          timestampBefore: dayjs(logTimes.first()),
        });
        return () => canceler.abort();
      }

      // Fetch newer log when direction=top_to_bottom and scroll is on bottom.
      if (direction === DIRECTIONS.TOP_TO_BOTTOM && visibleStopIndex === (logs.length - 1)) {
        const canceler = fetchAndAppendLogs(direction, {
          ...filter,
          timestampAfter: dayjs(logTimes.last()),
        });
        return () => canceler.abort();
      }
    }, [
      direction,
      fetchAndAppendLogs,
      filter,
      isLastReached,
      isLoading,
      logs,
    ]);

  /*
   * This overwrites the copy to clipboard event handler for the purpose of modifying the user
   * selected content. By default when copying content from a collection of HTML elements, each
   * element content will have a newline appended in the clipboard content. This handler will
   * detect which lines within the copied content to be the timestamp content and strip out the
   * newline from that field.
   */
  useLayoutEffect(() => {
    if (!container.current) return;

    const target = container.current;
    const handleCopy = (e: ClipboardEvent): void => {
      const clipboardFormat = 'text/plain';
      const levelValues = Object.values(LogLevel).join('|');
      const levelRegex = new RegExp(`<\\[(${levelValues})\\]>\n`, 'gim');
      const selection = (window.getSelection()?.toString() || '').replace(levelRegex, '<$1> ');
      const lines = selection?.split('\n');

      if (lines?.length <= 1) {
        e.clipboardData?.setData(clipboardFormat, selection);
      } else {
        const oddOrEven = lines.map(line => /^\[/.test(line) || /\]$/.test(line))
          .reduce((acc, isTimestamp, index) => {
            if (isTimestamp) acc[index % 2 === 0 ? 'even' : 'odd']++;
            return acc;
          }, { even: 0, odd: 0 });
        const isEven = oddOrEven.even > oddOrEven.odd;
        const content = lines.reduce((acc, line, index) => {
          const skipNewline = (isEven && index % 2 === 0) || (!isEven && index % 2 === 1);
          return acc + line + (skipNewline ? ' ' : '\n');
        }, '');
        e.clipboardData?.setData(clipboardFormat, content);
      }
      e.preventDefault();
    };

    target.addEventListener('copy', handleCopy);

    return (): void => target?.removeEventListener('copy', handleCopy);
  }, []);

  /*
   * Fetch filters data.
   */
  useEffect(() => {
    const canceler = new AbortController();

    consumeStream(
      onFetchLogFilter(canceler),
      event => setFilterOptions(event as LogViewerTimestampFilter),
    );

    return () => canceler.abort();
  }, [ onFetchLogFilter ]);

  /*
   * Fetch first batch of log (on direction change, on filters change, on page load).
   */
  useEffect(() => {
    clearLogs();
    const canceler = fetchAndAppendLogs(direction, filter);

    return () => canceler.abort();
  }, [ clearLogs, direction, fetchAndAppendLogs, filter ]);

  /*
   * Fetch Log tail (api follow).
   */
  useEffect(() => {
    if (direction !== DIRECTIONS.BOTTOM_TO_TOP) return;

    const canceler = new AbortController();

    let buffer: TrialLog[] = [];
    const throttleFunc = throttle(THROTTLE_TIME, () => {
      addLogs(buffer);
      buffer = [];
    });

    consumeStream(
      onFetchLogTail(filter, canceler),
      event => {
        buffer.push(fetchToLogConverter(event));
        throttleFunc();
      },
    );

    return () => {
      canceler.abort();
      throttleFunc.cancel();
    };
  }, [ addLogs, direction, fetchToLogConverter, filter, onFetchLogTail ]);

  /*
   * Automatically scroll to log tail (if tailing).
   */
  useLayoutEffect(() => {
    if (!isOnBottom) return;
    if (!listRef?.current) return;
    if (direction !== DIRECTIONS.BOTTOM_TO_TOP) return;

    listRef.current.scrollToItem(logs.length);
  }, [ direction, isOnBottom, listRef, logs ]);

  /*
   * Force recomputing messages height when width changes
   */
  useLayoutEffect(() => {
    listRef.current?.resetAfterIndex(0);
  }, [ listMeasure.width, listRef ]);

  const logOptions = (
    <Space>
      <Tooltip placement="bottomRight" title="Copy to Clipboard">
        <Button
          aria-label="Copy to Clipboard"
          disabled={logs.length === 0}
          icon={<Icon name="clipboard" />}
          onClick={handleCopyToClipboard} />
      </Tooltip>
      <Tooltip placement="bottomRight" title="Toggle Fullscreen Mode">
        <Button
          aria-label="Toggle Fullscreen Mode"
          icon={<Icon name="fullscreen" />}
          onClick={handleFullScreen} />
      </Tooltip>
      {onDownloadClick && <Tooltip placement="bottomRight" title="Download Logs">
        <Button
          aria-label="Download Logs"
          icon={<Icon name="download" />}
          onClick={handleDownload} />
      </Tooltip>}
    </Space>
  );

  const enableTailingClasses = [ css.enableTailing ];
  if (isOnBottom && direction === DIRECTIONS.BOTTOM_TO_TOP) enableTailingClasses.push(css.enabled);

  const LogViewerRow: React.FC<ListChildComponentProps> = useCallback(({ data, index, style }) => {
    const log = data[index];

    const messageClasses = [ css.message ];
    if (log.level) messageClasses.push(css[log.level]);

    return (
      <div className={css.line} style={style}>
        <LogViewerLevel logLevel={log.level} />
        <div className={css.time} style={{ width: toRem(dateTimeWidth) }}>
          {log.formattedTime}
        </div>
        <div
          className={messageClasses.join(' ')}
          dangerouslySetInnerHTML={{ __html: ansiToHtml(log.message) }}
        />
      </div>
    );
  }, [ dateTimeWidth ]);

  return (
    <Section
      bodyBorder
      bodyNoPadding
      filters={FilterComponent && <FilterComponent
        filter={filter}
        filterOptions={filterOptions}
        onChange={setFilter}
      />}
      maxHeight
      options={logOptions}
    >
      <div className={css.base} ref={baseRef}>
        <div className={css.container} ref={container}>
          <VariableSizeList
            height={listMeasure.height}
            itemCount={logs.length}
            itemData={logs}
            itemSize={getItemHeight}
            ref={listRef}
            width='100%'
            onItemsRendered={onItemsRendered}
          >
            {LogViewerRow}
          </VariableSizeList>
        </div>
        <div className={css.scrollTo}>
          <Tooltip placement="left" title="Scroll to Top">
            <Button
              aria-label="Scroll to Top"
              className={[ css.scrollToTop, css.show ].join(' ')}
              icon={<Icon name="arrow-up" />}
              onClick={handleScrollToTop} />
          </Tooltip>
          <Tooltip
            placement="left"
            title={direction === DIRECTIONS.BOTTOM_TO_TOP ? 'Tailing Enabled' : 'Enable Tailing'}
          >
            <Button
              aria-label="Enable Tailing"
              className={enableTailingClasses.join(' ')}
              icon={<Icon name="arrow-down" />}
              onClick={handleEnableTailing} />
          </Tooltip>
        </div>
      </div>
    </Section>
  );
};

export default LogViewerTimestamp;
