import { Button, notification, Space, Tooltip } from 'antd';
import React, {
  forwardRef, useCallback, useEffect, useImperativeHandle,
  useLayoutEffect, useMemo, useRef, useState,
} from 'react';
import screenfull from 'screenfull';
import { sprintf } from 'sprintf-js';
import { throttle } from 'throttle-debounce';

import Icon from 'components/Icon';
import usePrevious from 'hooks/usePrevious';
import useResize, { DEFAULT_RESIZE_THROTTLE_TIME } from 'hooks/useResize';
import useScroll, { defaultScrollInfo } from 'hooks/useScroll';
import { Log, LogLevel } from 'types';
import { formatDatetime } from 'utils/date';
import { ansiToHtml, copyToClipboard, toRem } from 'utils/dom';
import { capitalize } from 'utils/string';

import css from './LogViewer.module.scss';
import Page, { Props as PageProps } from './Page';

interface Props {
  debugMode?: boolean;
  disableLevel?: boolean;
  disableLineNumber?: boolean;
  filterOptions?: React.ReactNode;
  isDownloading?: boolean;
  isLoading?: boolean;
  noWrap?: boolean;
  onDownload?: () => void;
  onScrollToTop?: (oldestLogId: number) => void;
  pageProps: Partial<PageProps>;
  ref?: React.Ref<LogViewerHandles>;
}

interface ViewerLog extends Log {
  formattedTime: string;
}

interface MessageSize {
  height: number;
  top: number;
}

interface LogConfig {
  charHeight: number;
  charWidth: number;
  dateTimeWidth: number;
  lineNumberWidth: number;
  messageSizes: Record<string, MessageSize>;
  messageWidth: number;
  totalContentHeight: number;
}

interface MessageSize {
  height: number;
  top: number;
}

interface LogConfig {
  charHeight: number;
  charWidth: number;
  dateTimeWidth: number;
  lineNumberWidth: number;
  messageSizes: Record<string, MessageSize>;
  messageWidth: number;
  totalContentHeight: number;
}

export interface LogViewerHandles {
  addLogs: (newLogs: Log[], prepend?: boolean) => void;
  clearLogs: () => void;
}

export const TAIL_SIZE = 1000;

// What factor to multiply against the displayable lines in the visible view.
const BUFFER_FACTOR = 1;

// Format the datetime to...
const DATETIME_PREFIX = '[';
const DATETIME_SUFFIX = ']';
const DATETIME_FORMAT = `[${DATETIME_PREFIX}]YYYY-MM-DD, HH:mm:ss${DATETIME_SUFFIX}`;

// Max datetime size: DATETIME_FORMAT (plus 1 for a space suffix)
const MAX_DATETIME_LENGTH = 23;

// Number of pixels from the top of the scroll to trigger the `onScrollToTop` callback.
const SCROLL_TOP_THRESHOLD = 36;

const SCROLL_BOTTOM_THRESHOLD = 36;

const ICON_WIDTH = 26;

const defaultLogConfig = {
  charHeight: 0,
  charWidth: 0,
  dateTimeWidth: 0,
  lineNumberWidth: 0,
  messageSizes: {},
  messageWidth: 0,
  totalContentHeight: 0,
};

/*
 * The LogViewer is wrapped with `forwardRef` to provide the parent component
 * a reference to be able to call functions inside the LogViewer.
 */
const LogViewer: React.FC<Props> = forwardRef((
  { onDownload, onScrollToTop, ...props }: Props,
  ref?: React.Ref<LogViewerHandles>,
) => {
  const baseRef = useRef<HTMLDivElement>(null);
  const container = useRef<HTMLDivElement>(null);
  const spacer = useRef<HTMLDivElement>(null);
  const measure = useRef<HTMLDivElement>(null);
  const resize = useResize(container);
  const scroll = useScroll(container);
  const [ logs, setLogs ] = useState<ViewerLog[]>([]);
  const [ logIdRange, setLogIdRange ] =
    useState({ max: Number.MIN_SAFE_INTEGER, min: Number.MAX_SAFE_INTEGER });
  const [ scrollToInfo, setScrollToInfo ] =
    useState({ isPrepend: false, logId: 0 });
  const [ config, setConfig ] = useState<LogConfig>(defaultLogConfig);
  const [ isTailing, setIsTailing ] = useState(true);
  const previousScroll = usePrevious(scroll, defaultScrollInfo);
  const previousLogs = usePrevious<Log[]>(logs, []);
  const classes = [ css.base ];
  const scrollToTopClasses = [ css.scrollToTop ];
  const enableTailingClasses = [ css.enableTailing ];

  const spacerStyle = { height: toRem(config.totalContentHeight) };
  const dateTimeStyle = { width: toRem(config.dateTimeWidth) };
  const lineNumberStyle = { width: toRem(config.lineNumberWidth) };
  const messageStyle = { width: toRem(config.messageWidth) };
  const levelStyle = { width: toRem(ICON_WIDTH) };

  if (props.noWrap) classes.push(css.noWrap);
  if (scroll.scrollTop > SCROLL_TOP_THRESHOLD) scrollToTopClasses.push(css.show);
  // if (scroll.scrollTop > scroll.scrollHeight - scroll.viewHeight - SCROLL_BOTTOM_THRESHOLD) {
  if (isTailing) enableTailingClasses.push(css.enabled);

  /*
   * Calculate all the sizes of the log pieces such as the individual character size,
   * line numbers, datetime and message whenever new logs are added.
   */
  const measureLogs = useCallback((logs: ViewerLog[]): LogConfig => {
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
     * Set the line number column width based on the character width.
     * Add one to account for the trailing space character.
     */
    let lineNumberWidth = 0;
    if (!props.disableLineNumber) {
      const maxLineNumber = logs.length === 0 ? 1000 : logs.last().id + 1;
      const lineDigits = Math.ceil(Math.log(maxLineNumber) / Math.log(10)) + 1;
      lineNumberWidth = charRect.width * lineDigits;
    }

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
    const messageWidth = Math.floor(spacerRect.width - iconWidth - lineNumberWidth - dateTimeWidth);
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
      lineNumberWidth,
      messageSizes,
      messageWidth,
      totalContentHeight,
    };
  }, [ props.disableLevel, props.disableLineNumber ]);

  const addLogs = useCallback((addedLogs: Log[], prepend = false): void => {
    // Only process new logs that don't exist in the log viewer
    const newLogs = addedLogs
      .filter(log => log.id < logIdRange.min || log.id > logIdRange.max)
      .map(log => {
        const formattedTime = log.time ? formatDatetime(log.time, DATETIME_FORMAT) : '';
        return { ...log, formattedTime };
      });
    if (newLogs.length === 0) return;

    // Add new logs to existing logs either at the beginning or the end.
    const updatedLogs = prepend ? [ ...newLogs, ...logs ] : [ ...logs, ...newLogs ];
    const logConfig = measureLogs(updatedLogs);

    setConfig(logConfig);
    setScrollToInfo({ isPrepend: prepend, logId: logs[0]?.id });
    setLogs(updatedLogs);
    setLogIdRange(prevLogIdRange => ({
      max: Math.max(prevLogIdRange.max, newLogs.last().id),
      min: Math.min(prevLogIdRange.min, newLogs[0].id),
    }));
  }, [ logs, logIdRange, measureLogs ]);

  const clearLogs = useCallback((): void => {
    setConfig(defaultLogConfig);
    setScrollToInfo({ isPrepend: false, logId: 0 });
    setLogs([]);
    setLogIdRange({
      max: Number.MIN_SAFE_INTEGER,
      min: Number.MAX_SAFE_INTEGER,
    });
    setIsTailing(true);
  }, []);

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
   * The useImperitiveHandle hook provides the parent component
   * access to functions defined here to modify LogViewer state.
   */
  useImperativeHandle(ref, () => ({ addLogs, clearLogs }));

  /*
   * Pass event of user manually scrolling to the top to parent
   * in order to notify the parent to attempt to load older log entries.
   */
  useEffect(() => {
    // If there no logs to begin with, no need to load older logs.
    if (logs.length === 0) return;

    // Check to make sure the scroll position is at the top.
    if (scroll.scrollTop > SCROLL_TOP_THRESHOLD) return;

    // Skip if the previous state was already at the top.
    if (previousScroll.scrollTop <= SCROLL_TOP_THRESHOLD) return;

    // Skip if there isn't a callback.
    if (!onScrollToTop) return;

    // Send the callback the id of the earliest log entry.
    onScrollToTop(logs[0].id - 1);
  }, [ logs, previousScroll, onScrollToTop, scroll ]);

  /*
   * Detect the user navigating away from the bottom to disengage
   * the tailing behavior.
   */
  useEffect(() => {
    if (scroll.scrollTop < scroll.scrollHeight - scroll.viewHeight - SCROLL_BOTTOM_THRESHOLD) {
      setIsTailing(false);
    }
  }, [ scroll ]);

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
   * Scroll to the latest log entry when showing the very first
   * set of log entries. `setTimeout` is needed to ensure that
   * `scrollTo` occurs after the layout has settled.
   */
  useLayoutEffect(() => {
    if (previousLogs.length === 0 && logs.length > 0) {
      setTimeout(() => {
        if (!container.current) return;
        container.current.scrollTo({ behavior: 'auto', top: container.current.scrollHeight });
      });
    }
  }, [ logs, previousLogs ]);

  /*
   * This effect handles two special scrolling cases.
   * One for loading new log entries and another for when older
   * log entries have loaded. This is to allow the most seamless
   * user experience when scrolling through log entries.
   */
  useLayoutEffect(() => {
    if (isTailing) {
      /*
       * Automatically scroll to the latest log entry if previously
       * viewing the lastest log entry.
       */
      setTimeout(() => {
        if (!container.current) return;
        container.current.scrollTo({ behavior: 'auto', top: container.current.scrollHeight });
      });
    } else if (scrollToInfo.isPrepend) {
      /*
       * Restore the previous scroll position when loading older
       * log entries.
       */
      setTimeout(() => {
        if (!container.current || !scrollToInfo.logId) return;
        const top = config.messageSizes[scrollToInfo.logId].top;
        setScrollToInfo({ isPrepend: false, logId: 0 });
        container.current.scrollTo({ top });
      });
    }
  }, [ config, isTailing, scrollToInfo ]);

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

  const formatClipboardHeader = useCallback((log: Log): string => {
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
    container.current.scrollTo({ behavior: 'auto', top: 0 });
  }, []);

  const handleEnableTailing = useCallback(() => {
    if (!container.current) return;
    setIsTailing(true);
    container.current.scrollTo({ behavior: 'auto', top: container.current.scrollHeight });
  }, []);

  const handleDownload = useCallback(() => {
    if (onDownload) onDownload();
  }, [ onDownload ]);

  const logOptions = (
    <Space>
      {props.filterOptions}
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
      {onDownload && <Tooltip placement="bottomRight" title="Download Logs">
        <Button
          aria-label="Download Logs"
          icon={<Icon name="download" />}
          loading={props.isDownloading}
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
    <Page {...props.pageProps} loading={!!props.isLoading} options={logOptions}>
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
                {!props.disableLineNumber &&
                  <div className={css.number} data-label={log.id + 1} style={lineNumberStyle} />}
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
          <Tooltip placement="topRight" title={isTailing ? 'Tailing Enabled' : 'Enable Tailing'}>
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
});

export default LogViewer;
