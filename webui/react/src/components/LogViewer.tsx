import useScroll from 'hooks/useScroll';
import React, {
  forwardRef, useCallback, useEffect, useImperativeHandle, useLayoutEffect, useRef, useState,
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

export interface LogViewerHandles {
  addLogs: (newLogs: Log[]) => void;
}

// What factor to multiply against the displayable lines in the visible view.
const BUFFER_FACTOR = 2;

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
  const scroll = useScroll(container);
  const [ logs, setLogs ] = useState<Log[]>([]);
  const [ charSize, setCharSize ] = useState({ height: 0, width: 0 });
  const [ lineNumberStyle, setLineNumberStyle ] = useState({ width: 'auto' });
  const [ dateTimeStyle, setDateTimeStyle ] = useState({ width: 'auto' });
  const [ contentHeight, setContentHeight ] = useState(0);
  const [ visibleLines, setVisibleLines ] = useState(0);
  const [ isLoading, setIsLoading ] = useState(true);
  const classes = [ css.base ];

  if (props.disableSearch) classes.push(css.disableSearch);
  if (props.fullPage) classes.push(css.fullPage);
  if (props.noWrap) classes.push(css.noWrap);

  const addLogs = useCallback((newLogs: Log[]): void => {
    setLogs([ ...logs, ...newLogs ]);
  }, [ logs, setLogs ]);

  /*
   * The useImperitiveHandle hook provides the parent component
   * access to functions defined here to modify LogViewer state.
   */
  useImperativeHandle(ref, () => ({ addLogs }));

  useEffect(() => {
  }, [ scroll ]);

  useLayoutEffect(() => {
    if (!Array.isArray(logs) || logs.length === 0 || !container) return;

    // Check to make sure the container exists.
    const containerRect = container.current?.getBoundingClientRect();
    if (!containerRect?.height) return;

    // Check to make sure at least one line has been rendered properly.
    const messageContainer = container.current?.firstElementChild?.lastElementChild;
    const messageRect = messageContainer?.getBoundingClientRect();
    if (messageRect?.width === 0) return;

    /*
     * Add a single element to use a temporary container to guage the
     * size any text.
     */
    const measureElement = document.createElement('div');
    measureElement.style.position = 'absolute';
    container.current?.append(measureElement);

    // Get the width for a single character of the monospace font.
    measureElement.textContent = 'W';
    const charRect = measureElement.getBoundingClientRect();
    setCharSize({ height: charRect.height, width: charRect.width });

    /*
     * Figure out how many lines can fit in the visible window,
     * assuming each log is a one-liner.
     */
    setVisibleLines(Math.round(containerRect?.height / charRect.height));

    /*
     * Set the line number column width based on the character width.
     * Add one to account for the trailing space character.
     */
    const lineDigits = Math.ceil(Math.log(logs.length) / Math.log(10)) + 1;
    setLineNumberStyle({ width: toRem(charRect.width * lineDigits) });

    /*
     * Set the datetime column width based on the character width.
     * Largest possible datetime string is 34 characters:
     * eg. [YYYY-MM-DDTHH:mm:ss.ssssss-HH:MM]
     * Add one to account for the trailing space character.
     */
    setDateTimeStyle({ width: toRem(charRect.width * MAX_DATETIME_LENGTH) });

    /*
     * Measure the dimensions of every message in the available data.
     * Add up all the height to figure out what the scroll height is.
     */
    let contentHeight = 0;
    measureElement.style.width = toRem(messageRect?.width);
    logs.forEach(line => {
      measureElement.textContent = line.message;
      const rect = measureElement.getBoundingClientRect();
      contentHeight += rect.height;
    });
    setContentHeight(contentHeight);

    // Remove temporary element.
    container.current?.removeChild(measureElement);

    // Scroll to the bottom of the log
    container.current?.scrollTo(0, container.current?.scrollHeight || 0);

    setIsLoading(false);
  }, [ logs, setLineNumberStyle, setDateTimeStyle ]);

  return (
    <div className={css.base}>
      <div className={css.controller}>
        Control
      </div>
      <div className={css.container} ref={container}>
        <div className={css.scrollSpacer}>
          {logs.map((log, index) => (
            <div className={css.line} key={log.id}>
              <div className={css.number} style={lineNumberStyle}>{index + 1}</div>
              <div className={css.time} style={dateTimeStyle}>{log.time}</div>
              <div
                className={css.message}
                dangerouslySetInnerHTML={{ __html: ansiToHtml(log.message) }} />
            </div>
          ))}
        </div>
      </div>
      {isLoading && <Spinner fillContainer shade />}
    </div>
  );
});

export default LogViewer;
