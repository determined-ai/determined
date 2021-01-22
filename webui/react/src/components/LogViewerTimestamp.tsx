import { Button, notification, Space, Tooltip } from 'antd';
import dayjs, { Dayjs } from 'dayjs';
import React, {
  useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState,
} from 'react';
import screenfull from 'screenfull';
import { sprintf } from 'sprintf-js';
import { throttle } from 'throttle-debounce';

import Icon from 'components/Icon';
import usePrevious from 'hooks/usePrevious';
import useResize, { DEFAULT_RESIZE_THROTTLE_TIME } from 'hooks/useResize';
import useScroll, { defaultScrollInfo } from 'hooks/useScroll';
import {
  LogViewerTimestampFilterComponentProp,
  TrialLogFiltersInterface,
} from 'pages/TrialLogs/TrialLogFilters';
import { FetchArgs } from 'services/api-ts-sdk';
import { consumeStream } from 'services/utils';
import { LogLevel, TrialLog } from 'types';
import { formatDatetime } from 'utils/date';
import { ansiToHtml, copyToClipboard, toRem } from 'utils/dom';
import { capitalize } from 'utils/string';

import css from './LogViewer.module.scss';
import Page, { Props as PageProps } from './Page';

export interface LogViewerTimestampFilter {
  limit?: number,
  timestampAfter?: Dayjs,   // exclusive of the specified date time
  timestampBefore?: Dayjs,  // inclusive of the specified date time
}

interface Props {
  FilterComponent?: React.ComponentType<LogViewerTimestampFilterComponentProp>,
  debugMode?: boolean;
  disableLevel?: boolean;
  fetchToLogConverter: (data: unknown) => TrialLog,
  noWrap?: boolean;
  onDownloadClick?: () => void;
  onFetchLogAfter: (filters: LogViewerTimestampFilter, canceler: AbortController) => FetchArgs;
  onFetchLogBefore: (filters: LogViewerTimestampFilter, canceler: AbortController) => FetchArgs;
  onFetchLogFilter: (canceler: AbortController) => FetchArgs;
  onFetchLogTail: (filters: LogViewerTimestampFilter, canceler: AbortController) => FetchArgs;
  pageProps: Partial<PageProps>;
}

interface ViewerLog extends TrialLog {
  formattedTime: string;
}

interface LogConfig {
  charHeight: number;
  charWidth: number;
  dateTimeWidth: number;
  messageSizes: Record<string, MessageSize>;
  messageWidth: number;
  totalContentHeight: number;
}

interface MessageSize {
  height: number;
  top: number;
}

export const TAIL_SIZE = 500;

// What factor to multiply against the displayable lines in the visible view.
const BUFFER_FACTOR = 1;

// Format the datetime to...
const DATETIME_PREFIX = '[';
const DATETIME_SUFFIX = ']';
const DATETIME_FORMAT = `[${DATETIME_PREFIX}]YYYY-MM-DD HH:mm:ss${DATETIME_SUFFIX}`;

// Max datetime size: DATETIME_FORMAT (plus 1 for a space suffix)
const MAX_DATETIME_LENGTH = 23;

// Number of pixels from the top of the scroll to trigger the `onScrollToTop` callback.
const SCROLL_TOP_THRESHOLD = 36;

const SCROLL_BOTTOM_THRESHOLD = 36;

const ICON_WIDTH = 26;

const THROTTLE_TIME = 500;

const defaultLogConfig = {
  charHeight: 0,
  charWidth: 0,
  dateTimeWidth: 0,
  messageSizes: {},
  messageWidth: 0,
  totalContentHeight: 0,
};

const DIRECTIONS = {
  OLDEST: 'OLDEST', // show oldest logs and infinite-scroll newest ones at the bottom
  TAILING: 'TAILING', // show newest logs and infinite-scroll oldest ones at the top
};

const LogViewerTimestamp: React.FC<Props> = ({
  fetchToLogConverter,
  FilterComponent,
  onDownloadClick,
  onFetchLogAfter,
  onFetchLogBefore,
  onFetchLogFilter,
  onFetchLogTail,
  ...props
}: Props) => {
  const baseRef = useRef<HTMLDivElement>(null);
  const container = useRef<HTMLDivElement>(null);
  const spacer = useRef<HTMLDivElement>(null);
  const measure = useRef<HTMLDivElement>(null);

  const resize = useResize(container);
  const scroll = useScroll(container);

  const previousScroll = usePrevious(scroll, defaultScrollInfo);

  const [ config, setConfig ] = useState<LogConfig>(defaultLogConfig);
  const [ direction, setDirection ] = useState(DIRECTIONS.TAILING);
  const [ filter, setFilter ] = useState<LogViewerTimestampFilter>({ limit: TAIL_SIZE });
  const [ filterOptions, setFilterOptions ] = useState<LogViewerTimestampFilter>({});
  const [ historyCanceler ] = useState(new AbortController());
  const [ isLastReached, setIsLastReached ] = useState(false);
  const [ isPrevScrollOnBottom, setIsPrevScrollOnBottom ] = useState(false);
  const [ isPrevScrollOnTop, setIsPrevScrollOnTop ] = useState(false);
  const [ isScrollOnBottom, setIsScrollOnBottom ] = useState(true);
  const [ isScrollOnTop, setIsScrollOnTop ] = useState(false);
  const [ logTimestampRange, setLogTimestampRange ] = useState<{max:null|string, min:null|string}>({
    max: null,
    min: null,
  });
  const [ logs, setLogs ] = useState<ViewerLog[]>([]);

  const classes = [ css.base ];
  if (props.noWrap) classes.push(css.noWrap);
  const scrollToTopClasses = [ css.scrollToTop, css.show ];
  if (direction === DIRECTIONS.OLDEST) scrollToTopClasses.push(css.enabled);
  const enableTailingClasses = [ css.enableTailing ];
  if (direction === DIRECTIONS.TAILING) enableTailingClasses.push(css.enabled);

  const spacerStyle = { height: toRem(config.totalContentHeight) };
  const dateTimeStyle = { width: toRem(config.dateTimeWidth) };
  const messageStyle = { width: toRem(config.messageWidth) };
  const levelStyle = { width: toRem(ICON_WIDTH) };

  /*
   * Calculate all the sizes of the log pieces such as the individual character size,
   * line numbers, datetime and message whenever new logs are added.
   */
  const measureLogs = useCallback((logs): LogConfig => {
    // Check to make sure all the necessary elements are available.
    if (!measure.current || !spacer.current) throw new Error('Missing log measuring elements.');

    // Fetch container sizes for upcoming calculations.
    const spacerRect = spacer.current.getBoundingClientRect();

    // Show the measure element to support measuring text.
    measure.current.style.display = 'inline';

    // Get the width for a single character of the monospace font.
    measure.current.textContent = 'W';
    measure.current.style.width = 'auto';
    const charRect = measure.current.getBoundingClientRect();

    /*
     * Set the datetime column width based on the character width.
     * Largest possible datetime string is 34 characters:
     * eg. [YYYY-MM-DDTHH:mm:ss.ssssss-HH:MM]
     * Add one to account for the trailing space character.
     */
    const dateTimeWidth = charRect.width * MAX_DATETIME_LENGTH;

    /*
     * Calculate the width of message based on how much space is left
     * after rendering line and timestamp.
     */
    const iconWidth = props.disableLevel ? 0 : ICON_WIDTH;
    const messageWidth = Math.floor(spacerRect.width - iconWidth - dateTimeWidth);
    const messageCharCount = Math.floor(messageWidth / charRect.width);

    /*
      * Calculate the dimensions of every message in the available data.
      * Add up all the height to figure out what the scroll height is.
      */
    let totalContentHeight = 0;
    const messageSizes: Record<string, MessageSize> = {};
    measure.current.style.width = toRem(messageWidth);
    logs.forEach((log: ViewerLog) => {
      const lineCount = log.message
        .split('\n')
        .map(line => line.length > messageCharCount ? Math.ceil(line.length / messageCharCount) : 1)
        .reduce((acc, count) => acc + count, 0);
      const height = lineCount * charRect.height;
      messageSizes[log.id] = { height, top: totalContentHeight };
      totalContentHeight += height;
    });

    // Hide the measure element
    measure.current.style.display = 'none';

    // Return all the calculated sizes for log view configuartion.
    return {
      charHeight: charRect.height,
      charWidth: charRect.width,
      dateTimeWidth,
      messageSizes,
      messageWidth,
      totalContentHeight,
    };
  }, [ props.disableLevel ]);

  /*
   * Figure out which logs lines to actually render based on whether it
   * is visible in the scroll view window or not.
   */
  const visibleLogs = useMemo(() => {
    if (config.totalContentHeight === 0) return logs;

    const viewTop = scroll.scrollTop - scroll.viewHeight * BUFFER_FACTOR;
    const viewBottom = scroll.scrollTop + scroll.viewHeight * (1 + BUFFER_FACTOR);

    return logs.filter(log => {
      const size = config.messageSizes[log.id];
      if (!size) return false;
      const top = size.top;
      const bottom = size.top + size.height;
      return (top > viewTop && top < viewBottom) || (bottom > viewTop && bottom < viewBottom);
    });
  }, [ config, logs, scroll ]);

  /*
   * Detect log viewer resize events to trigger
   * recalculation of measured log entries.
   */
  useLayoutEffect(() => {
    const throttleFunc = throttle(DEFAULT_RESIZE_THROTTLE_TIME, () => {
      if (!container.current) return;
      setConfig(measureLogs(logs));
    });

    throttleFunc();
  }, [ logs, measureLogs, resize ]);

  /*
   * Check if user scroll is on top/bottom.
   */
  useLayoutEffect(() => {
    setIsScrollOnBottom(
      scroll.scrollHeight
      - scroll.viewHeight
      - scroll.scrollTop
      < SCROLL_BOTTOM_THRESHOLD,
    );
    setIsScrollOnTop(
      scroll.scrollTop < SCROLL_TOP_THRESHOLD,
    );
  }, [ scroll ]);

  /*
   * Check if user previous scroll was on top/bottom.
   */
  useLayoutEffect(() => {
    setIsPrevScrollOnBottom(
      previousScroll.scrollHeight
      - previousScroll.viewHeight
      - previousScroll.scrollTop
      < SCROLL_BOTTOM_THRESHOLD,
    );
    setIsPrevScrollOnTop(
      previousScroll.scrollTop < SCROLL_TOP_THRESHOLD,
    );
  }, [ previousScroll ]);

  /*
   * Automatically scroll to log tail (if tailing).
   */
  useLayoutEffect(() => {
    const element = container.current;

    if (
      !element
      || !isScrollOnBottom
      || direction !== DIRECTIONS.TAILING
    ) return;

    setTimeout(() => {
      element.scrollTo({ top: element.scrollHeight });
    });
  }, [ container, direction, isScrollOnBottom, logs ]);

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

  const formatClipboardHeader = useCallback((log: TrialLog): string => {
    const format = `%${MAX_DATETIME_LENGTH - 1}s `;
    const level = `<${log.level || ''}>`;
    const datetime = log.time ? formatDatetime(log.time, DATETIME_FORMAT) : '';
    return props.disableLevel ?
      sprintf(format, datetime) :
      sprintf(`%-9s ${format}`, level, datetime);
  }, [ props.disableLevel ]);

  const handleCopyToClipboard = useCallback(async () => {
    const content = logs.map(log => `${formatClipboardHeader(log)}${log.message || ''}`).join('\n');

    try {
      await copyToClipboard(content);
      const linesLabel = logs.length === 1 ? 'entry' : 'entries';
      notification.open({
        description: `${logs.length} ${linesLabel} copied to the clipboard.`,
        message: `Available ${props.pageProps.title} Copied`,
      });
    } catch (e) {
      notification.warn({
        description: e.message,
        message: 'Unable to Copy to Clipboard',
      });
    }
  }, [ formatClipboardHeader, logs, props.pageProps.title ]);

  const handleFullScreen = useCallback(() => {
    if (baseRef.current && screenfull.isEnabled) screenfull.toggle();
  }, []);

  const handleScrollToTop = useCallback(() => {
    if (!container.current) return;
    setDirection(DIRECTIONS.OLDEST);
    container.current.scrollTo({ behavior: 'auto', top: 0 });
  }, []);

  const handleEnableTailing = useCallback(() => {
    if (!container.current) return;
    setDirection(DIRECTIONS.TAILING);
    container.current.scrollTo({ behavior: 'auto', top: container.current.scrollHeight });
  }, []);

  const handleDownload = useCallback(() => {
    if (onDownloadClick) onDownloadClick();
  }, [ onDownloadClick ]);

  const addLogs = useCallback((addedLogs: TrialLog[], isPrepend = false): void => {
    const newLogs = addedLogs
      .map(log => {
        const formattedTime = log.time ? formatDatetime(log.time, DATETIME_FORMAT) : '';
        return { ...log, formattedTime };
      });
    if (newLogs.length === 0) return;

    const prevScrollHeight = container?.current?.scrollHeight;

    setLogs(prevLogs => {
      const logs = isPrepend ? [ ...newLogs, ...prevLogs ] : [ ...prevLogs, ...newLogs ];
      return logs.filter((log, index, self) => {
        return self.map(mapObj => mapObj.id).indexOf(log.id) === index;
      });
    });

    /*
     * Restore the previous scroll position when prepending log entries.
     */
    if (isPrepend && container?.current && prevScrollHeight) {
      container.current.scrollTo({
        top: (
          container.current.scrollHeight
          + container.current.scrollTop
          - prevScrollHeight
        ),
      });
    }

    // update log timestamp range
    let max: string|null = null;
    let min: string|null = null;
    newLogs.forEach(log => {
      if (!log.time) return;
      if (!max || log.time > max) max = log.time;
      if (!min || log.time < min) min = log.time;
    });
    setLogTimestampRange(prev => ({
      max: (prev.max && max && dayjs(prev.max).isAfter(dayjs(max))) ? prev.max : max,
      min: (prev.min && min && dayjs(prev.min).isBefore(dayjs(min))) ? prev.min : min,
    }));
  }, [ container ]);

  const clearLogs = useCallback(() => {
    setIsLastReached(false);
    setLogs([]);
    setLogTimestampRange({
      max: null,
      min: null,
    });
  }, []);

  /*
   * Fetch filters data.
   */
  useEffect(() => {
    consumeStream(
      onFetchLogFilter(historyCanceler),
      event => setFilterOptions(event as LogViewerTimestampFilter),
    );

    return () => historyCanceler.abort();
  }, [ historyCanceler, onFetchLogFilter ]);

  /*
   * Load first batch of logs (tail or oldest).
   */
  useEffect(() => {
    const canceler = new AbortController();
    clearLogs();

    let buffer: TrialLog[] = [];
    const throttleFunc = throttle(THROTTLE_TIME, () => {
      addLogs(buffer);
      buffer = [];
    });

    const fetchArgs = (
      direction === DIRECTIONS.TAILING
        ? onFetchLogTail(filter, canceler)
        : onFetchLogAfter({ ...filter, timestampAfter: dayjs(0) }, canceler)
    );

    consumeStream(
      fetchArgs,
      event => {
        buffer.push(fetchToLogConverter(event));
        throttleFunc();
      },
    );

    return () => {
      canceler.abort();
      throttleFunc.cancel();
    };
  }, [
    addLogs,
    clearLogs,
    direction,
    fetchToLogConverter,
    filter,
    onFetchLogAfter,
    onFetchLogTail,
  ]);

  /*
   * Load older/newer log entries when the user scroll to the top/bottom.
   */
  useEffect(() => {
    if (isLastReached) return;

    const canceler = new AbortController();
    let fetchArgs = null;
    let isPrepend = false;

    if (direction === DIRECTIONS.TAILING) {
      if (!logTimestampRange.min) return;
      if (!isScrollOnTop) return;
      if (isPrevScrollOnTop) return;

      fetchArgs = onFetchLogBefore({
        ...filter,
        timestampBefore: dayjs(logTimestampRange.min),
      }, canceler);
      isPrepend = true;
    }

    if (direction === DIRECTIONS.OLDEST) {
      if (!logTimestampRange.max) return;
      if (!isScrollOnBottom) return;
      if (isPrevScrollOnBottom) return;

      fetchArgs = onFetchLogAfter({
        ...filter,
        timestampAfter: dayjs(logTimestampRange.max),
      }, canceler);
      isPrepend = false;
    }

    if (fetchArgs) {
      let buffer: TrialLog[] = [];
      consumeStream(
        fetchArgs,
        event => buffer.push(fetchToLogConverter(event)),
      ).then(() => {
        if (buffer.length < TAIL_SIZE) setIsLastReached(true);

        addLogs(buffer, isPrepend);
        buffer = [];
      });
    }
  }, [
    addLogs,
    direction,
    fetchToLogConverter,
    filter,
    isLastReached,
    isPrevScrollOnBottom,
    isPrevScrollOnTop,
    isScrollOnBottom,
    isScrollOnTop,
    logs,
    logTimestampRange,
    onFetchLogAfter,
    onFetchLogBefore,
  ]);

  const logOptions = (
    <Space>
      {FilterComponent && (
        <FilterComponent
          filter={filter}
          filterOptions={filterOptions}
          onChange={(newFilters: TrialLogFiltersInterface) => setFilter(newFilters)}
        />
      )}
      {props.debugMode && <div className={css.debugger}>
        <span data-label="ScrollLeft:">{scroll.scrollLeft}</span>
        <span data-label="ScrollTop:">{scroll.scrollTop}</span>
        <span data-label="ScrollWidth:">{scroll.scrollWidth}</span>
        <span data-label="ScrollHeight:">{scroll.scrollHeight}</span>
      </div>}
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

  const levelCss = (defaultCss: string, level?: string): string => {
    const classes = [ defaultCss ];
    if (level) classes.push(css[level]);
    return classes.join(' ');
  };

  return (
    <Page {...props.pageProps} options={logOptions}>
      <div className={css.base} ref={baseRef}>
        <div className={css.container} ref={container}>
          <div className={css.scrollSpacer} ref={spacer} style={spacerStyle}>
            {visibleLogs.map(log => (
              <div
                className={css.line}
                id={`log-${log.id}`}
                key={log.id}
                style={{
                  height: toRem(config.messageSizes[log.id]?.height),
                  top: toRem(config.messageSizes[log.id]?.top),
                }}>
                {!props.disableLevel ? (
                  <Tooltip placement="top" title={`Level: ${capitalize(log.level || '')}`}>
                    <div className={levelCss(css.level, log.level)} style={levelStyle}>
                      <div className={css.levelLabel}>&lt;[{log.level || ''}]&gt;</div>
                      <Icon name={log.level} size="small" />
                    </div>
                  </Tooltip>
                ) : null}
                <div className={css.time} style={dateTimeStyle}>{log.formattedTime}</div>
                <div
                  className={levelCss(css.message, log.level)}
                  dangerouslySetInnerHTML={{ __html: ansiToHtml(log.message) }}
                  style={messageStyle} />
              </div>
            ))}
          </div>
          <div className={css.measure} ref={measure} />
        </div>
        <div className={css.scrollTo}>
          <Tooltip placement="topRight" title="Scroll to Top">
            <Button
              aria-label="Scroll to Top"
              className={scrollToTopClasses.join(' ')}
              icon={<Icon name="arrow-up" />}
              onClick={handleScrollToTop} />
          </Tooltip>
          <Tooltip
            placement="topRight"
            title={direction === DIRECTIONS.TAILING ? 'Tailing Enabled' : 'Enable Tailing'}
          >
            <Button
              aria-label="Enable Tailing"
              className={enableTailingClasses.join(' ')}
              icon={<Icon name="arrow-down" />}
              onClick={handleEnableTailing} />
          </Tooltip>
        </div>
      </div>
    </Page>
  );
};

export default LogViewerTimestamp;
