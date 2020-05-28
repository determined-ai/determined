import useScroll from 'hooks/useScroll';
import React, {
  forwardRef, useCallback, useImperativeHandle,
  useLayoutEffect, useMemo, useRef, useState,
} from 'react';

import Spinner from 'components/Spinner';
import useScroll from 'hooks/useScroll';
import { Log } from 'types';
import { ansiToHtml, toRem } from 'utils/dom';

import css from './LogViewer.module.scss';

interface Props {
  fullPage?: boolean;
  noWrap?: boolean;
  disableSearch?: boolean;
  ref?: React.Ref<LogViewerHandles>;
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
  addLogs: (newLogs: Log[]) => void;
}

// Number of digits to support before logs are added.
const DEFAULT_LINE_NUMBER_DIGITS = 2;

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
  const container = useRef<HTMLDivElement>(null);
  const spacer = useRef<HTMLDivElement>(null);
  const measure = useRef<HTMLDivElement>(null);
  const [ scroll, resizeScrollElement ] = useScroll(container);
  const [ logs, setLogs ] = useState<Log[]>([]);
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

  if (props.disableSearch) classes.push(css.disableSearch);
  if (props.fullPage) classes.push(css.fullPage);
  if (props.noWrap) classes.push(css.noWrap);

  const addLogs = useCallback((newLogs: Log[]): void => {
    if (newLogs.length === 0) return;

    // Add new logs to existing logs.
    setLogs(prevLogs => ([ ...prevLogs, ...newLogs ]));

    // Check to make sure all the necessary elements are available.
    if (!container.current || !measure.current || !spacer.current) return;

    // Fetch container sizes for upcoming calculations.
    const spacerRect = spacer.current.getBoundingClientRect();

    // Show the measure element to support measuring text.
    measure.current.style.display = 'inline';

    // Get the width for a single character of the monospace font.
    measure.current.textContent = 'W';
    const charRect = measure.current.getBoundingClientRect();

    /*
     * Set the line number column width based on the character width.
     * Add one to account for the trailing space character.
     */
    const lineDigits = Math.ceil(Math.log(newLogs.length) / Math.log(10)) + 1;
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
    newLogs.forEach(line => {
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

    // Recalculate the scroll element sizing.
    resizeScrollElement();

    // Hide the measure element
    measure.current.style.display = 'none';

    // Scroll to the bottom of the log if adding the first set of logs.
    if (logs.length === 0) {
      setTimeout(() => container.current?.scrollTo(0, container.current.scrollHeight || 0), 0);
    }
  }, [ logs, resizeScrollElement, setLogs ]);

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
   * Calculate log viewer sizes after the component has rendered.
   * This does not include calculations of individual log line heights,
   * which is done when logs are being added via `addLogs`.
   */
  useLayoutEffect(() => {
    // Check to make sure all the necessary elements are available.
    if (!container.current) return;

    // Scroll to the bottom of the log
    container.current.scrollTo(0, container.current.scrollHeight || 0);
  }, []);

  return (
    <div className={css.base}>
      <div className={css.controller}>
        Control
      </div>
      <div className={css.container} ref={container}>
        <div className={css.scrollSpacer} ref={spacer} style={spacerStyle}>
          {visibleLogs.map((log, index) => (
            <div className={css.line} key={log.id} style={{
              height: toRem(config.messageSizes[log.id]?.height),
              top: toRem(config.messageSizes[log.id]?.top),
            }}>
              <div className={css.number} style={lineNumberStyle}>{index + 1}</div>
              <div className={css.time} style={dateTimeStyle}>{log.time}</div>
              <div
                className={css.message}
                dangerouslySetInnerHTML={{ __html: ansiToHtml(log.message) }} />
            </div>
          ))}
        </div>
        <div className={css.measure} ref={measure} />
      </div>
    </div>
  );
});

export default LogViewer;
