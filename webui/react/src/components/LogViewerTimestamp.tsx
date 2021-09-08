import { Button, notification, Space, Tooltip } from 'antd';
import dayjs, { Dayjs } from 'dayjs';
import queryString from 'query-string';
import React, {
  Reducer, useCallback, useEffect, useLayoutEffect, useReducer, useRef, useState,
} from 'react';
import { useLocation } from 'react-router-dom';
import {
  ListChildComponentProps, ListOnItemsRenderedProps, ListOnScrollProps, VariableSizeList,
} from 'react-window';
import screenfull from 'screenfull';
import { sprintf } from 'sprintf-js';
import { throttle } from 'throttle-debounce';

import Icon from 'components/Icon';
import useGetCharMeasureInContainer from 'hooks/useGetCharMeasureInContainer';
import useResize from 'hooks/useResize';
import { LogViewerTimestampFilterComponentProp } from 'pages/TrialDetails/Logs/TrialLogFilters';
import { FetchArgs } from 'services/api-ts-sdk';
import { consumeStream } from 'services/utils';
import { LogLevel, TrialLog } from 'types';
import { formatDatetime } from 'utils/date';
import { copyToClipboard } from 'utils/dom';

import css from './LogViewer.module.scss';
import { LogStoreAction, LogStoreActionType, logStoreReducer, ViewerLog } from './LogViewer.store';
import LogViewerEntry, { DATETIME_FORMAT, ICON_WIDTH, MAX_DATETIME_LENGTH } from './LogViewerEntry';
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
  onFetchLogTail?: (filters: LogViewerTimestampFilter, canceler: AbortController) => FetchArgs;
}

export const TAIL_SIZE = 100;

const THROTTLE_TIME = 500;
const PADDING = 8;

enum Direction {
  TopToBottom = 'top-to-bottom', // show oldest logs and infinite-scroll newest ones at the bottom
  BottomToTop = 'bottom-to-top', // show newest logs and infinite-scroll oldest ones at the top
}

const formatClipboardHeader = (log: TrialLog): string => {
  const format = `%${MAX_DATETIME_LENGTH - 1}s `;
  const level = `<${log.level || ''}>`;
  const datetime = log.time ? formatDatetime(log.time, DATETIME_FORMAT) : '';
  return sprintf(`%-9s ${format}`, level, datetime);
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
  const containerRef = useRef<HTMLDivElement>(null);
  const listRef = useRef<VariableSizeList>(null);
  const listOffset = useRef<number>(0);

  const location = useLocation();
  const charMeasures = useGetCharMeasureInContainer(containerRef);
  const containerSize = useResize(containerRef);

  const dateTimeWidth = charMeasures.width * MAX_DATETIME_LENGTH;
  const maxCharPerLine = Math.floor(
    (containerSize.width - ICON_WIDTH - dateTimeWidth - 2 * PADDING) / charMeasures.width,
  );

  const [ direction, setDirection ] = useState(Direction.BottomToTop);
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
        // If the line is probably a TQDM line, hide it
        const hide = log.message.includes('\r');
        const formattedTime = log.time ? formatDatetime(log.time, DATETIME_FORMAT) : '';
        return { ...log, formattedTime, hide };
      })
      .filter(logEntry => !logEntry.hide)
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
  }, [ logsDispatch ]);

  const fetchAndAppendLogs = useCallback((
    direction: Direction,
    filters: LogViewerTimestampFilter,
  ): AbortController => {
    const canceler = new AbortController();
    let fetchArgs = null;
    let isPrepend = false;

    if (direction === Direction.BottomToTop) {
      fetchArgs = onFetchLogBefore(filters, canceler);
      isPrepend = true;
    }

    if (direction === Direction.TopToBottom) {
      fetchArgs = onFetchLogAfter({
        ...filters,
        timestampAfter: filters.timestampAfter?.subtract(1, 'millisecond'),
      }, canceler);
      isPrepend = false;
    }

    if (fetchArgs) {
      setIsLoading(true);

      let buffer: TrialLog[] = [];
      consumeStream(
        fetchArgs,
        event => {
          const logEntry = fetchToLogConverter(event);
          direction === Direction.TopToBottom
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
    if (!log) return charMeasures.height;

    const lineCount = log.message
      .split('\n')
      .map(line => line.length > maxCharPerLine ? Math.ceil(line.length / maxCharPerLine) : 1)
      .reduce((acc, count) => acc + count, 0);
    const itemHeight = lineCount * charMeasures.height;

    return (index === 0 || index === logs.length - 1) ? itemHeight + PADDING : itemHeight;
  }, [ charMeasures, maxCharPerLine, logs ]);

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
    setDirection(Direction.BottomToTop);
    listRef.current?.scrollToItem(logs.length, 'end');
  }, [ listRef, logs.length ]);

  const handleFullScreen = useCallback(() => {
    if (baseRef.current && screenfull.isEnabled) screenfull.toggle();
  }, []);

  const handleScrollToTop = useCallback(() => {
    setDirection(Direction.TopToBottom);
  }, []);

  const handleItemsRendered = useCallback((
    { visibleStartIndex, visibleStopIndex }: ListOnItemsRenderedProps,
  ) => {
    setIsOnBottom(visibleStopIndex === (logs.length - 1));

    if (isLoading) return;
    if (isLastReached) return;
    if (!listRef?.current) return;

    const logTimes = logs.map(log => log.time).sort();

    // Fetch older log when direction=BottomToTop and scroll is on top.
    if (direction === Direction.BottomToTop && visibleStartIndex === 0) {
      const canceler = fetchAndAppendLogs(direction, {
        ...filter,
        timestampBefore: dayjs(logTimes.first()),
      });
      return () => canceler.abort();
    }

    // Fetch newer log when direction=TopToBottom and scroll is on bottom.
    if (direction === Direction.TopToBottom && visibleStopIndex === (logs.length - 1)) {
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

  const handleScroll = useCallback((event: ListOnScrollProps) => {
    if (event.scrollOffset) listOffset.current = event.scrollOffset;
  }, []);

  /*
   * This overwrites the copy to clipboard event handler for the purpose of modifying the user
   * selected content. By default when copying content from a collection of HTML elements, each
   * element content will have a newline appended in the clipboard content. This handler will
   * detect which lines within the copied content to be the timestamp content and strip out the
   * newline from that field.
   */
  useLayoutEffect(() => {
    if (!containerRef.current) return;

    const target = containerRef.current;
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
    if (direction !== Direction.BottomToTop || !onFetchLogTail) return;

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
   * If query param `tail` is set, enable tailing behavior.
   */
  useEffect(() => {
    const { tail } = queryString.parse(location.search);
    if (tail !== undefined) {
      setDirection(Direction.BottomToTop);
      setTimeout(() => {
        listRef.current?.scrollToItem(logs.length, 'end');
      }, 0);
    }
  }, [ location.search, logs.length ]);

  /*
   * Automatically scroll to log tail (if tailing).
   */
  useLayoutEffect(() => {
    if (!isOnBottom) return;
    if (!listRef.current) return;
    if (direction !== Direction.BottomToTop) return;

    listRef.current.scrollToItem(logs.length, 'end');
  }, [ direction, isOnBottom, listRef, logs ]);

  /*
   * Force recomputing messages height when container width changes.
   */
  useLayoutEffect(() => {
    const ref = listRef.current;
    ref?.resetAfterIndex(0);

    // Restore the list offset if applicable.
    if (listOffset.current) {
      setTimeout(() => ref?.scrollTo(listOffset.current), 0);
    }

    return () => ref?.resetAfterIndex(0);
  }, [ containerSize.width, containerSize.height ]);

  const logOptions = (
    <div className={css.options}>
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
    </div>
  );

  const enableTailingClasses = [ css.enableTailing ];
  if (isOnBottom && direction === Direction.BottomToTop) enableTailingClasses.push(css.enabled);

  const LogViewerRow: React.FC<ListChildComponentProps> = useCallback(({ data, index, style }) => (
    <LogViewerEntry
      style={{
        ...style,
        left: parseFloat(`${style.left}`) + PADDING,
        paddingTop: index === 0 ? PADDING : 0,
        width: `calc(100% - ${2 * PADDING}px)`,
      }}
      timeStyle={{ width: dateTimeWidth }}
      {...data[index]}
    />
  ), [ dateTimeWidth ]);

  return (
    <Section
      bodyNoPadding
      bodyScroll
      filters={FilterComponent && <FilterComponent
        filter={filter}
        filterOptions={filterOptions}
        onChange={setFilter}
      />}
      maxHeight
      options={logOptions}>
      <div className={css.base} ref={baseRef}>
        <div className={css.container} ref={containerRef}>
          <VariableSizeList
            height={containerSize.height}
            itemCount={logs.length}
            itemData={logs}
            itemSize={getItemHeight}
            ref={listRef}
            width="100%"
            onItemsRendered={handleItemsRendered}
            onScroll={handleScroll}>
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
            title={direction === Direction.BottomToTop ? 'Tailing Enabled' : 'Enable Tailing'}>
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
