import Button from 'hew/Button';
import Icon from 'hew/Icon';
import { clone, dateTimeStringSorter, formatDatetime, numericSorter } from 'hew/internal/functions';
import { readLogStream } from 'hew/internal/services';
import { FetchArgs, Log, LogLevel, RecordKey } from 'hew/internal/types';
import LogViewerEntry, { DATETIME_FORMAT } from 'hew/LogViewer/LogViewerEntry';
import Message from 'hew/Message';
import Section from 'hew/Section';
import Spinner from 'hew/Spinner';
import { ErrorHandler } from 'hew/utils/error';
import { ValueOf } from 'hew/utils/types';
import React, { useCallback, useEffect, useId, useLayoutEffect, useRef, useState } from 'react';
import { ListItem, Virtuoso, VirtuosoHandle } from 'react-virtuoso';
import { throttle } from 'throttle-debounce';

import css from './LogViewer.module.scss';

export interface Props<T> {
  decoder: (data: T) => Log;

  height?: number | 'stretch';
  initialLogs?: T[];

  onFetch?: (config: FetchConfig, type: FetchType) => FetchArgs;
  onError: ErrorHandler;
  serverAddress: (path: string) => string;
  sortKey?: keyof Log;
  selectedLog?: Log;
  logs: ViewerLog[];
  setLogs: React.Dispatch<React.SetStateAction<ViewerLog[]>>;
  logsRef: React.RefObject<HTMLDivElement>;
  local: React.MutableRefObject<{
    idSet: Set<RecordKey>;
    isScrollReady: boolean;
  }>;
}

export interface ViewerLog extends Log {
  formattedTime: string;
}

export interface FetchConfig {
  canceler: AbortController;
  fetchDirection: FetchDirection;
  limit: number;
  offset?: number;
  offsetLog?: Log;
}

export const FetchType = {
  Initial: 'Initial',
  Newer: 'Newer',
  Older: 'Older',
  Stream: 'Stream',
} as const;

export type FetchType = ValueOf<typeof FetchType>;

export const FetchDirection = {
  Newer: 'Newer',
  Older: 'Older',
} as const;

export type FetchDirection = ValueOf<typeof FetchDirection>;

export const ARIA_LABEL_ENABLE_TAILING = 'Enable Tailing';
export const ARIA_LABEL_SCROLL_TO_OLDEST = 'Scroll to Oldest';

export const PAGE_LIMIT = 100;
const THROTTLE_TIME = 50;

export const formatLogEntry = (log: Log): ViewerLog => {
  const formattedTime = log.time ? formatDatetime(log.time, { format: DATETIME_FORMAT }) : '';
  return { ...log, formattedTime };
};

const logSorter =
  (key: keyof Log) =>
  (a: Log, b: Log): number => {
    const aValue = a[key];
    const bValue = b[key];
    if (key === 'id') return numericSorter(aValue as number, bValue as number);
    if (key === 'time') return dateTimeStringSorter(aValue as string, bValue as string);
    return 0;
  };

// This is an arbitrarily large number used as an index. See https://virtuoso.dev/prepend-items/
const START_INDEX = 2_000_000_000;

function LogViewer<T>({
  decoder,
  height = 'stretch',
  initialLogs,
  onFetch,
  onError,
  serverAddress,
  sortKey = 'time',
  selectedLog,
  logs,
  setLogs,
  logsRef,
  local,
}: Props<T>): JSX.Element {
  const componentId = useId();

  const virtuosoRef = useRef<VirtuosoHandle>(null);
  const [isFetching, setIsFetching] = useState(false);

  const [canceler] = useState(new AbortController());
  const [fetchDirection, setFetchDirection] = useState<FetchDirection>(FetchDirection.Older);
  const [isTailing, setIsTailing] = useState<boolean>(true);
  const [showButtons, setShowButtons] = useState(false);

  const [firstItemIndex, setFirstItemIndex] = useState(START_INDEX);
  const [scrolledForSearch, setScrolledForSearch] = useState(true);

  const processLogs = useCallback(
    (newLogs: Log[]) => {
      return newLogs
        .filter((log) => {
          const isDuplicate = local.current.idSet.has(log.id);
          const isTqdm = log.message.includes('\r');
          local.current.idSet.add(log.id);
          return !isDuplicate && !isTqdm;
        })
        .map((log) => formatLogEntry(log))
        .sort(logSorter(sortKey));
    },
    [sortKey, local],
  );

  const addLogs = useCallback(
    (newLogs: ViewerLog[] = [], prepend = false): void => {
      if (newLogs.length === 0) return;
      setLogs((prevLogs) => (prepend ? newLogs.concat(prevLogs) : prevLogs.concat(newLogs)));
      if (prepend) setFirstItemIndex((prev) => prev - newLogs.length);
    },
    [setLogs],
  );

  useEffect(() => {
    setScrolledForSearch(false);
  }, [selectedLog]);

  useEffect(() => {
    if (scrolledForSearch || !selectedLog || !local.current.isScrollReady) return;
    setTimeout(() => {
      const index = logs.findIndex((l) => l.id === selectedLog.id);
      if (index > -1 && index + 1 < logs.length) {
        virtuosoRef.current?.scrollToIndex({
          behavior: 'smooth',
          index: index,
        });
        setScrolledForSearch(true);
      }
    }, 250);
  }, [scrolledForSearch, selectedLog, logs, local]);

  const fetchLogs = useCallback(
    async (config: Partial<FetchConfig>, type: FetchType): Promise<ViewerLog[]> => {
      if (!onFetch) return [];

      const buffer: Log[] = [];

      setIsFetching(true);

      await readLogStream(
        serverAddress,
        onFetch({ limit: PAGE_LIMIT, ...config } as FetchConfig, type),
        onError,

        (event: T) => {
          const logEntry = decoder(event);
          fetchDirection === FetchDirection.Older
            ? buffer.unshift(logEntry)
            : buffer.push(logEntry);
        },
      );

      setIsFetching(false);

      return processLogs(buffer);
    },
    [decoder, fetchDirection, onFetch, onError, processLogs, serverAddress],
  );

  const handleFetchMoreLogs = useCallback(
    async (positionReached: 'start' | 'end') => {
      // Scroll may occur before the initial logs have rendered.
      if (!local.current.isScrollReady) return;

      // Still busy with a previous fetch, prevent another fetch.
      if (isFetching) return;

      // Detect when user scrolls to the "edge" and requires more logs to load.
      const shouldFetchNewLogs = positionReached === 'end';
      const shouldFetchOldLogs = positionReached === 'start';
      if (shouldFetchNewLogs || shouldFetchOldLogs) {
        const newLogs = await fetchLogs(
          {
            canceler,
            fetchDirection: shouldFetchNewLogs ? FetchType.Newer : FetchType.Older,
            offsetLog: shouldFetchNewLogs ? logs[logs.length - 1] : logs[0],
          },
          shouldFetchNewLogs ? FetchType.Newer : FetchType.Older,
        );

        const prevLogs = clone(logs);

        addLogs(newLogs, shouldFetchOldLogs);

        if (newLogs.length > 0 && shouldFetchNewLogs) {
          setTimeout(() => {
            virtuosoRef.current?.scrollToIndex({
              align: 'end',
              index: prevLogs.length - 1,
            });
          }, 100);
        }
        return newLogs;
      }
      return;
    },
    [addLogs, canceler, fetchLogs, isFetching, logs, local],
  );

  const handleScrollToOldest = useCallback(() => {
    setIsTailing(false);

    if (fetchDirection === FetchDirection.Newer) {
      virtuosoRef.current?.scrollToIndex({ index: firstItemIndex });
    } else {
      local.current = {
        idSet: new Set<RecordKey>([]),
        isScrollReady: false as boolean,
      };

      setLogs([]);
      setFetchDirection(FetchDirection.Newer);
      setFirstItemIndex(0);
    }
  }, [fetchDirection, firstItemIndex, setLogs, local]);

  const handleEnableTailing = useCallback(() => {
    setIsTailing(true);

    if (fetchDirection === FetchDirection.Older) {
      virtuosoRef.current?.scrollToIndex({ index: 'LAST' });
    } else {
      local.current = {
        idSet: new Set<RecordKey>([]),
        isScrollReady: false as boolean,
      };

      setLogs([]);
      setFetchDirection(FetchDirection.Older);
      setFirstItemIndex(START_INDEX);
    }
  }, [fetchDirection, setLogs, local]);

  // Re-fetch logs when fetch callback changes.
  useEffect(() => {
    local.current = {
      idSet: new Set<RecordKey>([]),
      isScrollReady: false as boolean,
    };

    setLogs([]);
    setIsTailing(true);
    setFetchDirection(FetchDirection.Older);
    setFirstItemIndex(START_INDEX);
  }, [onFetch, setLogs, local]);

  // Add initial logs if applicable.
  useEffect(() => {
    if (!initialLogs) return;
    addLogs(processLogs(initialLogs.map((log) => decoder(log))));
  }, [addLogs, decoder, initialLogs, processLogs]);

  // Initial fetch on mount or when fetch direction changes.
  useEffect(() => {
    fetchLogs({ canceler, fetchDirection }, FetchType.Initial).then((logs) => {
      addLogs(logs, false);

      // Slight delay on scrolling to the end for the log viewer to render and resolve everything.
      setTimeout(() => {
        local.current.isScrollReady = true;
      }, 200);
    });
  }, [addLogs, canceler, fetchDirection, fetchLogs, local]);

  // Enable streaming for loading latest entries.
  useEffect(() => {
    const canceler = new AbortController();
    let buffer: Log[] = [];

    const processBuffer = () => {
      const logs = processLogs(buffer);
      buffer = [];

      addLogs(logs);
    };
    const throttledProcessBuffer = throttle(THROTTLE_TIME, processBuffer);

    if (fetchDirection === FetchDirection.Older && onFetch) {
      readLogStream(
        serverAddress,
        onFetch({ canceler, fetchDirection, limit: PAGE_LIMIT }, FetchType.Stream),
        onError,
        (event: T) => {
          buffer.push(decoder(event));
          throttledProcessBuffer();
        },
      );
    }

    return () => {
      canceler.abort();
      throttledProcessBuffer.cancel();
    };
  }, [addLogs, decoder, fetchDirection, onError, serverAddress, onFetch, processLogs]);

  // Abort all outstanding API calls if log viewer unmounts.
  useEffect(() => {
    return () => {
      canceler.abort();
    };
  }, [canceler]);

  const handleItemsRendered = useCallback(
    (renderedLogs: ListItem<ViewerLog>[]) => {
      setShowButtons(renderedLogs.length < logs.length);
    },
    [logs.length],
  );

  const handleReachedBottom = useCallback(
    async (atBottom: boolean) => {
      // May trigger before the initial logs have rendered.
      if (isFetching || !local.current.isScrollReady) return;

      if (atBottom && !isTailing) {
        const newLogs = await handleFetchMoreLogs('end');
        if (newLogs?.length === 0) setIsTailing(true);
      } else if (!atBottom) setIsTailing(false);
    },
    [handleFetchMoreLogs, isFetching, isTailing, local],
  );

  const handleReachedTop = useCallback(
    (atTop: boolean) => {
      if (atTop) handleFetchMoreLogs('start');
    },
    [handleFetchMoreLogs],
  );

  /*
   * This overwrites the copy to clipboard event handler for the purpose of modifying the user
   * selected content. By default when copying content from a collection of HTML elements, each
   * element content will have a newline appended in the clipboard content. This handler will
   * detect which lines within the copied content to be the timestamp content and strip out the
   * newline from that field.
   */
  useLayoutEffect(() => {
    if (!logsRef.current) return;

    const target = logsRef.current;
    const handleCopy = (e: ClipboardEvent): void => {
      const clipboardFormat = 'text/plain';
      const levelValues = Object.values(LogLevel).join('|');
      const levelRegex = new RegExp(`<\\[(${levelValues})\\]>\n`, 'gim');
      const selection = (window.getSelection()?.toString() || '').replace(levelRegex, '<$1> ');
      const lines = selection?.split('\n');

      if (lines?.length <= 1) {
        e.clipboardData?.setData(clipboardFormat, selection);
      } else {
        const oddOrEven = lines
          .map((line) => /^\[/.test(line) || /\]$/.test(line))
          .reduce(
            (acc, isTimestamp, index) => {
              if (isTimestamp) acc[index % 2 === 0 ? 'even' : 'odd']++;
              return acc;
            },
            { even: 0, odd: 0 },
          );
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
  }, [logsRef]);

  return (
    <Section fullHeight>
      <Spinner center spinning={isFetching}>
        <div className={css.base} style={{ height: height === 'stretch' ? '100%' : `${height}px` }}>
          <div className={css.logs} ref={logsRef}>
            {logs.length > 0 ? (
              <Virtuoso
                atBottomStateChange={handleReachedBottom}
                atBottomThreshold={12}
                atTopStateChange={handleReachedTop}
                customScrollParent={logsRef.current || undefined}
                data={logs}
                firstItemIndex={firstItemIndex}
                followOutput="smooth"
                initialTopMostItemIndex={
                  fetchDirection === FetchDirection.Older
                    ? (initialLogs?.length ?? PAGE_LIMIT) - 1
                    : 0
                }
                itemContent={(_index, logEntry) => (
                  <LogViewerEntry
                    formattedTime={logEntry.formattedTime}
                    key={logEntry.id}
                    level={logEntry.level}
                    message={logEntry.message}
                    style={{
                      backgroundColor:
                        logEntry.id === selectedLog?.id ? 'var(--theme-ix-active)' : 'transparent',
                    }}
                  />
                )}
                itemsRendered={handleItemsRendered}
                key={(logs.length === 0 ? 'Loading' : fetchDirection) + componentId}
                ref={virtuosoRef}
              />
            ) : (
              <Message icon="warning" title="No logs to show. " />
            )}
          </div>
          <div className={css.buttons} style={{ display: showButtons ? 'flex' : 'none' }}>
            <Button
              aria-label={ARIA_LABEL_SCROLL_TO_OLDEST}
              icon={<Icon name="arrow-up" showTooltip title={ARIA_LABEL_SCROLL_TO_OLDEST} />}
              onClick={handleScrollToOldest}
            />
            <Button
              aria-label={ARIA_LABEL_ENABLE_TAILING}
              icon={
                <Icon
                  name="arrow-down"
                  showTooltip
                  title={isTailing ? 'Tailing Enabled' : ARIA_LABEL_ENABLE_TAILING}
                />
              }
              onClick={handleEnableTailing}
            />
          </div>
        </div>
      </Spinner>
    </Section>
  );
}

export default LogViewer;
