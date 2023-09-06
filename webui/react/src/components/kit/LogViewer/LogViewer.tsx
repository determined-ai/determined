import { Space } from 'antd';
import React, { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import { flushSync } from 'react-dom';
import {
  ListChildComponentProps,
  ListOnItemsRenderedProps,
  ListOnScrollProps,
  VariableSizeList,
} from 'react-window';
import screenfull from 'screenfull';
import { sprintf } from 'sprintf-js';
import { throttle } from 'throttle-debounce';

import Button from 'components/kit/Button';
import ClipboardButton from 'components/kit/ClipboardButton';
import Icon from 'components/kit/Icon';
import {
  clone,
  dateTimeStringSorter,
  formatDatetime,
  numericSorter,
} from 'components/kit/internal/functions';
import Link from 'components/kit/internal/Link';
import Section from 'components/kit/internal/Section';
import { readLogStream } from 'components/kit/internal/services';
import {
  ErrorHandler,
  FetchArgs,
  Log,
  LogLevel,
  RecordKey,
  ValueOf,
} from 'components/kit/internal/types';
import useGetCharMeasureInContainer from 'components/kit/internal/useGetCharMeasureInContainer';
import useResize from 'components/kit/internal/useResize';
import Spinner from 'components/kit/Spinner';

import css from './LogViewer.module.scss';
import LogViewerEntry, { DATETIME_FORMAT, ICON_WIDTH, MAX_DATETIME_LENGTH } from './LogViewerEntry';

export interface Props {
  decoder: (data: unknown) => Log;
  handleCloseLogs?: () => void;
  initialLogs?: unknown[];
  onDownload?: () => void;
  onFetch?: (config: FetchConfig, type: FetchType) => FetchArgs;
  onError: ErrorHandler;
  serverAddress: (path: string) => string;
  sortKey?: keyof Log;
  title?: React.ReactNode;
}

export interface ViewerLog extends Log {
  formattedTime: string;
}

type Hash = Record<RecordKey, boolean>;

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

const PAGE_LIMIT = 100;
const PADDING = 8;
const THROTTLE_TIME = 50;

const defaultLocal = {
  fetchOffset: -PAGE_LIMIT,
  idMap: {} as Hash,
  isAtOffsetEnd: false,
  isFetching: false,
  isOnBottom: false,
  isOnTop: false,
  isScrollReady: false,
  previousHeight: 0,
  previousWidth: 0,
  scrollOffset: 0,
};

export const formatLogEntry = (log: Log): ViewerLog => {
  const formattedTime = log.time ? formatDatetime(log.time, { format: DATETIME_FORMAT }) : '';
  return { ...log, formattedTime };
};

const formatClipboardHeader = (log: Log): string => {
  const logEntry = formatLogEntry(log);
  const format = `%${MAX_DATETIME_LENGTH - 1}s `;
  const level = `<${logEntry.level || ''}>`;
  return sprintf(`%-9s ${format}`, level, logEntry.formattedTime);
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

const LogViewer: React.FC<Props> = ({
  decoder,
  initialLogs,
  onDownload,
  onFetch,
  onError,
  serverAddress,
  sortKey = 'time',
  handleCloseLogs,
  ...props
}: Props) => {
  const baseRef = useRef<HTMLDivElement>(null);
  const listRef = useRef<VariableSizeList>(null);
  const [isFetching, setIsFetching] = useState(false);
  const local = useRef(clone(defaultLocal));
  const [canceler] = useState(new AbortController());
  const [fetchDirection, setFetchDirection] = useState<FetchDirection>(FetchDirection.Older);
  const [isTailing, setIsTailing] = useState<boolean>(true);
  const [showButtons, setShowButtons] = useState<boolean>(false);
  const [logs, setLogs] = useState<ViewerLog[]>([]);
  const { elementRef: logsRef, ref, size: containerSize } = useResize();
  const { size: pageSize } = useResize();
  const charMeasures = useGetCharMeasureInContainer(logsRef);

  const { dateTimeWidth, maxCharPerLine } = useMemo(() => {
    const dateTimeWidth = charMeasures.width * MAX_DATETIME_LENGTH;
    const maxCharPerLine = Math.floor(
      (containerSize.width - ICON_WIDTH - dateTimeWidth - 2 * PADDING) / charMeasures.width,
    );
    return { dateTimeWidth, maxCharPerLine };
  }, [charMeasures.width, containerSize.width]);

  const getItemHeight = useCallback(
    (index: number): number => {
      const log = logs[index];
      if (!log) return charMeasures.height;

      const lineCount = log.message
        .split('\n')
        .map((line) => (line.length > maxCharPerLine ? Math.ceil(line.length / maxCharPerLine) : 1))
        .reduce((acc, count) => acc + count, 0);
      const itemHeight = lineCount * charMeasures.height;

      return index === 0 || index === logs.length - 1 ? itemHeight + PADDING : itemHeight;
    },
    [charMeasures, maxCharPerLine, logs],
  );

  const resizeLogs = useCallback(() => listRef.current?.resetAfterIndex(0), []);

  const processLogs = useCallback(
    (newLogs: Log[]) => {
      const map = local.current.idMap;
      return newLogs
        .filter((log) => {
          const isDuplicate = map[log.id];
          const isTqdm = log.message.includes('\r');
          map[log.id] = true;
          return !isDuplicate && !isTqdm;
        })
        .map((log) => formatLogEntry(log))
        .sort(logSorter(sortKey));
    },
    [sortKey],
  );

  const addLogs = useCallback(
    (newLogs: ViewerLog[], prepend = false): void => {
      if (newLogs.length === 0) return;
      flushSync(() => {
        setLogs((prevLogs) => (prepend ? [...newLogs, ...prevLogs] : [...prevLogs, ...newLogs]));
      });
      resizeLogs();
    },
    [resizeLogs],
  );

  const fetchLogs = useCallback(
    async (config: Partial<FetchConfig>, type: FetchType): Promise<ViewerLog[]> => {
      if (!onFetch) return [];

      const buffer: Log[] = [];

      setIsFetching(true);
      local.current.isFetching = true;

      await readLogStream(
        serverAddress,
        onFetch({ limit: PAGE_LIMIT, ...config } as FetchConfig, type),
        onError,
        (event) => {
          const logEntry = decoder(event);
          fetchDirection === FetchDirection.Older
            ? buffer.unshift(logEntry)
            : buffer.push(logEntry);
        },
      );

      setIsFetching(false);
      local.current.isFetching = false;

      return processLogs(buffer);
    },
    [decoder, fetchDirection, onFetch, onError, processLogs, serverAddress],
  );

  const handleItemsRendered = useCallback(
    async ({ visibleStartIndex, visibleStopIndex }: ListOnItemsRenderedProps) => {
      // Scroll may occur before the initial logs have rendered.
      if (!local.current.isScrollReady) return;

      local.current.isOnTop = visibleStartIndex === 0;
      local.current.isOnBottom = visibleStopIndex === logs.length - 1;

      // Still busy with a previous fetch, prevent another fetch.
      if (local.current.isFetching || local.current.isAtOffsetEnd) return;

      // Detect when user scrolls to the "edge" and requires more logs to load.
      const shouldFetchNewLogs =
        local.current.isOnBottom && fetchDirection === FetchDirection.Newer;
      const shouldFetchOldLogs = local.current.isOnTop && fetchDirection === FetchDirection.Older;

      if (shouldFetchNewLogs || shouldFetchOldLogs) {
        const newLogs = await fetchLogs(
          {
            canceler,
            fetchDirection,
            offsetLog: shouldFetchNewLogs ? logs.last() : logs.first(),
          },
          shouldFetchNewLogs ? FetchType.Newer : FetchType.Older,
        );

        addLogs(newLogs, shouldFetchOldLogs);

        // Restore previous scroll position upon adding older logs.
        if (shouldFetchOldLogs) {
          listRef.current?.scrollToItem(newLogs.length + 1, 'start');
        }

        // No more logs will load.
        if (newLogs.length === 0) {
          local.current.isAtOffsetEnd = true;

          /**
           * The user has scrolled all the way to the newest entry,
           * enable tailing behavior.
           */
          if (shouldFetchNewLogs) {
            setIsTailing(true);
            setFetchDirection(FetchDirection.Older);
          }
        }
      }
    },
    [addLogs, canceler, fetchDirection, fetchLogs, logs],
  );

  /**
   * scrollUpdateWasRequested:
   *   true:  if the scroll was caused by scrollTo() or scrollToItem()
   *   false: if the scroll was caused by user interaction in the browser
   */
  const handleScroll = useCallback(
    ({ scrollDirection, scrollOffset, scrollUpdateWasRequested }: ListOnScrollProps) => {
      /**
       * `react-window` automatically adjusts floating point offsets to integers.
       * Unfortunately, this triggers a second `onScroll` event with the `scrollUpdateWasRequested`
       * set as `false` indicating user triggered scrolling, which is not the case.
       * `isAutoWindowAdjustment` logic is used to filter out these auto adjustments made
       * my `react-window`.
       */
      const prevScrollOffset = local.current.scrollOffset;
      const isUserScrollBackwards = scrollDirection === 'backward' && !scrollUpdateWasRequested;
      const isAutoWindowAdjustment =
        prevScrollOffset !== scrollOffset && Math.floor(prevScrollOffset) === scrollOffset;
      if (isUserScrollBackwards && !isAutoWindowAdjustment) setIsTailing(false);

      // Re-engage tailing if the scroll position is at the bottom of the scrollable window.
      if (logsRef.current) {
        const listParent = logsRef.current.firstElementChild;
        const list = listParent?.firstElementChild;
        const scrollHeight = list?.scrollHeight ?? 0;
        const parentHeight = listParent?.clientHeight ?? 0;
        const scrollTop = scrollHeight - parentHeight;
        if (scrollTop && scrollTop === scrollOffset) setIsTailing(true);
      }

      // Store last scrollOffset.
      local.current.scrollOffset = scrollOffset;
    },
    [logsRef],
  );

  const handleScrollToOldest = useCallback(() => {
    setIsTailing(false);

    if (fetchDirection === FetchDirection.Newer) {
      listRef.current?.scrollToItem(0, 'start');
    } else {
      local.current.fetchOffset = 0;
      local.current.idMap = {};
      local.current.isScrollReady = false;
      local.current.isAtOffsetEnd = false;

      setLogs([]);
      setFetchDirection(FetchDirection.Newer);
    }
  }, [fetchDirection]);

  const handleEnableTailing = useCallback(() => {
    setIsTailing(true);

    if (fetchDirection === FetchDirection.Older) {
      listRef.current?.scrollToItem(logs.length, 'end');
    } else {
      local.current.fetchOffset = -PAGE_LIMIT;
      local.current.idMap = {};
      local.current.isScrollReady = false;
      local.current.isAtOffsetEnd = false;

      setLogs([]);
      setFetchDirection(FetchDirection.Older);
    }
  }, [fetchDirection, logs.length]);

  const clipboardCopiedMessage = useMemo(() => {
    const linesLabel = logs.length === 1 ? 'entry' : 'entries';
    return `Copied ${logs.length} ${linesLabel}!`;
  }, [logs]);

  const getClipboardContent = useCallback(() => {
    return logs.map((log) => `${formatClipboardHeader(log)}${log.message || ''}`).join('\n');
  }, [logs]);

  const handleFullScreen = useCallback(() => {
    if (baseRef.current && screenfull.isEnabled) screenfull.toggle();
  }, []);

  const handleDownload = useCallback(() => {
    onDownload?.();
  }, [onDownload]);

  // Fetch initial logs on a mount or when the mode changes.
  useEffect(() => {
    fetchLogs({ canceler, fetchDirection }, FetchType.Initial).then((logs) => {
      addLogs(logs, true);

      if (fetchDirection === FetchDirection.Older) {
        // Slight delay on scrolling to the end for the log viewer to render and resolve everything.
        setTimeout(() => {
          listRef.current?.scrollToItem(Number.MAX_SAFE_INTEGER, 'end');
          local.current.isScrollReady = true;
        }, 100);
      } else {
        listRef.current?.scrollToItem(0, 'start');
        local.current.isScrollReady = true;
      }
    });
  }, [addLogs, canceler, fetchDirection, fetchLogs]);

  // Enable streaming for loading latest entries.
  useEffect(() => {
    const canceler = new AbortController();
    let buffer: Log[] = [];

    const processBuffer = () => {
      const logs = processLogs(buffer);
      buffer = [];

      if (logs.length !== 0) {
        /*
         * We need to take a snapshot of `isOnBottom` BEFORE adding logs,
         * to determine if the log viewer is tailing.
         * The action of adding logs causes `isOnBottom` to be always false,
         * because the newly append logs are past the visible window.
         */
        const currentIsOnBottom = local.current.isOnBottom;

        addLogs(logs);

        if (currentIsOnBottom) {
          listRef.current?.scrollToItem(Number.MAX_SAFE_INTEGER, 'end');
        }
      }
    };
    const throttledProcessBuffer = throttle(THROTTLE_TIME, processBuffer);

    if (fetchDirection === FetchDirection.Older && onFetch) {
      readLogStream(
        serverAddress,
        onFetch({ canceler, fetchDirection, limit: PAGE_LIMIT }, FetchType.Stream),
        onError,
        (event) => {
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

  // Re-fetch logs when fetch callback changes.
  useEffect(() => {
    local.current = clone(defaultLocal);

    setLogs([]);
    setIsTailing(true);
    setFetchDirection(FetchDirection.Older);
  }, [onFetch]);

  // Initialize logs if applicable.
  useEffect(() => {
    if (!initialLogs) return;

    addLogs(initialLogs.map((log) => formatLogEntry(decoder(log))));
  }, [addLogs, decoder, initialLogs]);

  // Abort all outstanding API calls if log viewer unmounts.
  useEffect(() => {
    return () => {
      canceler.abort();
    };
  }, [canceler]);

  // Force recomputing messages height when container size changes.
  useLayoutEffect(() => {
    if (containerSize.width === 0 || containerSize.height === 0) return;

    const sizeChanged =
      containerSize.height !== local.current.previousHeight ||
      containerSize.width !== local.current.previousWidth;
    if (sizeChanged) resizeLogs();

    local.current.previousWidth = containerSize.width;
    local.current.previousHeight = containerSize.height;
  }, [containerSize, resizeLogs]);

  // Show scrolling buttons based on whether or not logs spill outside of the list view.
  useLayoutEffect(() => {
    setShowButtons(() => {
      if (!logsRef.current || logs.length === 0) return false;
      const listParent = logsRef.current.firstElementChild;
      const list = listParent?.firstElementChild;
      const scrollHeight = list?.scrollHeight ?? 0;
      const parentHeight = listParent?.clientHeight ?? 0;
      return scrollHeight > parentHeight;
    });
  }, [logs.length, logsRef]);

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

  const logViewerOptions = (
    <div className={css.options}>
      <Space>
        <ClipboardButton copiedMessage={clipboardCopiedMessage} getContent={getClipboardContent} />
        <Button
          aria-label="Toggle Fullscreen Mode"
          icon={<Icon name="fullscreen" showTooltip title="Toggle Fullscreen Mode" />}
          onClick={handleFullScreen}
        />
        {handleCloseLogs && (
          <Link onClick={handleCloseLogs}>
            <Icon name="close" title="Close Logs" />
          </Link>
        )}
        {onDownload && (
          <Button
            aria-label="Download Logs"
            icon={<Icon name="download" showTooltip title="Download Logs" />}
            onClick={handleDownload}
          />
        )}
      </Space>
    </div>
  );

  const LogViewerRow: React.FC<ListChildComponentProps> = useCallback(
    ({ data, index, style }) => (
      <LogViewerEntry
        style={{
          ...style,
          left: parseFloat(`${style.left}`) + PADDING,
          outline: 'none',
          paddingTop: index === 0 ? PADDING : 0,
          width: `calc(100% - ${2 * PADDING}px)`,
        }}
        timeStyle={{ width: dateTimeWidth }}
        {...data[index]}
      />
    ),
    [dateTimeWidth],
  );

  return (
    <Section
      bodyNoPadding
      bodyScroll
      divider
      maxHeight
      options={logViewerOptions}
      title={props.title}>
      <Spinner center spinning={isFetching} tip={logs.length === 0 ? 'No logs to show.' : ''}>
        <div className={css.base} ref={baseRef}>
          <div className={css.container}>
            <div className={css.logs} ref={ref}>
              <VariableSizeList
                height={pageSize.height - 250}
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
};

export default LogViewer;
