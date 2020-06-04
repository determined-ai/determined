import { Button, notification, Space, Tooltip } from 'antd';
import React, {
  forwardRef, useCallback, useEffect, useImperativeHandle, useMemo, useRef, useState,
} from 'react';
import screenfull from 'screenfull';

import Icon from 'components/Icon';
import Section from 'components/Section';
import useScroll from 'hooks/useScroll';
import { Log } from 'types';
import { ansiToHtml, toRem } from 'utils/dom';

import css from './LogViewer.module.scss';

interface Props {
  noWrap?: boolean;
  ref?: React.Ref<LogViewerHandles>;
  title: string;
  onLoadOlderLogs?: (oldestLogId: number) => void;
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
}

// What factor to multiply against the displayable lines in the visible view.
const BUFFER_FACTOR = 1;

// Max datetime size: [YYYY-MM-DDTHH:mm:ss.ssssss-HH:mm]
const MAX_DATETIME_LENGTH = 35;

/*
 * The LogViewer is wrapped with `forwardRef` to provide the parent component
 * a reference to be able to call functions inside the LogViewer.
 */
const LogViewer: React.FC<Props> = forwardRef((
  props: Props,
  ref?: React.Ref<LogViewerHandles>,
) => {
  const baseRef = useRef<HTMLDivElement>(null);
  const container = useRef<HTMLDivElement>(null);
  const spacer = useRef<HTMLDivElement>(null);
  const measure = useRef<HTMLDivElement>(null);
  const { scroll, scrollTo } = useScroll(container);
  const [ logIds, setLogIds ] = useState<Record<number, boolean>>({});
  const [ logs, setLogs ] = useState<Log[]>([]);
  const [ hasAutoScrolled, setHasAutoScrolled ] = useState(false);
  const [ isLoadingOlderLogs, setIsLoadingOlderLogs ] = useState(false);
  const [ prevScrollHeight, setPrevScrollHeight ] = useState(0);
  const [ config, setConfig ] = useState<LogConfig>({
    charHeight: 0,
    charWidth: 0,
    dateTimeWidth: 0,
    lineNumberWidth: 0,
    messageSizes: {},
    messageWidth: 0,
    totalContentHeight: 0,
  });
  const classes = [ css.base ];

  const spacerStyle = { height: toRem(config.totalContentHeight) };
  const dateTimeStyle = { width: toRem(config.dateTimeWidth) };
  const lineNumberStyle = { width: toRem(config.lineNumberWidth) };

  if (props.noWrap) classes.push(css.noWrap);

  const addLogs = useCallback((addedLogs: Log[], prepend = false): void => {
    // Mark that the loading of older logs is complete.
    if (prepend) setIsLoadingOlderLogs(false);

    // Only process new logs that don't exist in the log viewer
    const newLogIds: Record<number, boolean> = {};
    const newLogs = addedLogs.filter(log => {
      newLogIds[log.id] = true;
      return config.messageSizes[log.id] == null && !logIds[log.id];
    });

    // Add new logs to existing logs either in the beginning or the end.
    setLogIds(prevLogIds => ({ ...prevLogIds, ...newLogIds }));
    setLogs(prevLogs => prepend ? [ ...newLogs, ...prevLogs ] : [ ...prevLogs, ...newLogs ]);

    /*
     * Preserve the current scroll height to restore scroll position when loading
     * prepending logs.
     */
    if (container.current) setPrevScrollHeight(container.current.scrollHeight);

    // Automatically scroll to the bottom of the log if adding the first set of logs.
    if (logs.length === 0) {
      setTimeout(() => {
        if (!container.current) return;
        const top = container.current?.scrollHeight - container.current?.clientHeight;
        scrollTo({ behavior: 'auto', top }).then(() => setHasAutoScrolled(true));
      }, 0);
    }
  }, [ config, container, logIds, logs, scrollTo ]);

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
  useImperativeHandle(ref, () => ({ addLogs }));

  /*
   * Pass scrollToTop event to parent when user has manually scrolled to the top.
   * Detecting just the scrollTop position of 0 is not enough since 0 is
   * the default position until the first set of logs are added.
   */
  useEffect(() => {
    if (!props.onLoadOlderLogs) return;
    if (isLoadingOlderLogs) return;
    if (logs.length === 0) return;
    if (scroll.scrollTop > 0 || !hasAutoScrolled) return;
    setIsLoadingOlderLogs(true);
    props.onLoadOlderLogs(logs[0].id - 1);
  }, [ hasAutoScrolled, isLoadingOlderLogs, logs, props, props.onLoadOlderLogs, scroll ]);

  /*
   * This side effect calculates all the sizes of the log parts such as
   * the line numbers, datetime and message whenever new logs are added.
   */
  useEffect(() => {
    // Check to make sure all the necessary elements are available.
    if (!container.current || !measure.current || !spacer.current) return;

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
    const maxLineNumber = logs.length === 0 ? 1000 : logs[logs.length - 1].id + 1;
    const lineDigits = Math.ceil(Math.log(maxLineNumber) / Math.log(10)) + 1;
    const lineNumberWidth = charRect.width * lineDigits;

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
    const messageWidth = spacerRect.width - lineNumberWidth - dateTimeWidth;

    /*
      * Measure the dimensions of every message in the available data.
      * Add up all the height to figure out what the scroll height is.
      */
    let totalContentHeight = 0;
    const messageSizes: Record<string, MessageSize> = {};
    measure.current.style.width = toRem(messageWidth);
    logs.forEach(line => {
      /* eslint-disable @typescript-eslint/no-non-null-assertion */
      measure.current!.textContent = line.message;
      const rect = measure.current!.getBoundingClientRect();
      messageSizes[line.id] = { height: rect.height, top: totalContentHeight };
      totalContentHeight += rect.height;
    });

    // Save all the calculated sizes for log view configuartion.
    setConfig(prevConfig => ({
      ...prevConfig,
      charHeight: charRect.height,
      charWidth: charRect.width,
      dateTimeWidth,
      lineNumberWidth,
      messageSizes,
      messageWidth,
      totalContentHeight,
    }));

    // Hide the measure element
    measure.current.style.display = 'none';

    // Scroll to previous position
    if (logs.length !== 0) {
      scrollTo({ behavior: 'auto', top: totalContentHeight - prevScrollHeight });
    }
  }, [ logs, prevScrollHeight, scrollTo ]);

  const handleCopyToClipboard = useCallback(() => {
    const content = logs.map(log => [ log.time, log.message ].join(' ')).join('\n');

    navigator.clipboard.writeText(content);

    notification.open({
      description: `Available ${props.title} copied to clipboard.`,
      message: `${props.title} Copied`,
    });
  }, [ logs, props.title ]);

  const handleFullScreen = useCallback(() => {
    if (baseRef.current && screenfull.isEnabled) screenfull.toggle();
  }, []);

  const logOptions = (
    <Space>
      <Tooltip placement="bottomRight" title="Copy to Clipboard">
        <Button
          aria-label="Copy to Clipboard"
          icon={<Icon name="clipboard" />}
          onClick={handleCopyToClipboard} />
      </Tooltip>
      <Tooltip placement="bottomRight" title="Toggle Fullscreen Mode">
        <Button
          aria-label="Toggle Fullscreen Mode"
          icon={<Icon name="fullscreen" />}
          onClick={handleFullScreen} />
      </Tooltip>
    </Space>
  );

  return (
    <div className={css.base} ref={baseRef}>
      <Section maxHeight options={logOptions} title={props.title}>
        <div className={css.container} ref={container}>
          <div className={css.scrollSpacer} ref={spacer} style={spacerStyle}>
            {visibleLogs.map(log => (
              <div className={css.line} key={log.id} style={{
                height: toRem(config.messageSizes[log.id]?.height),
                top: toRem(config.messageSizes[log.id]?.top),
              }}>
                <div className={css.number} style={lineNumberStyle}>{log.id + 1}</div>
                <div className={css.time} style={dateTimeStyle}>{log.time}</div>
                <div
                  className={css.message}
                  dangerouslySetInnerHTML={{ __html: ansiToHtml(log.message) }} />
              </div>
            ))}
          </div>
          <div className={css.measure} ref={measure} />
        </div>
      </Section>
    </div>
  );
});

export default LogViewer;
