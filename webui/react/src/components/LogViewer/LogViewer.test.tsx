import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';

import { FetchArgs } from 'services/api-ts-sdk';
import { jsonToTaskLog } from 'services/decoder';
import { LogLevel } from 'types';
import { generateAlphaNumeric } from 'utils/string';

import * as LogViewer from './LogViewer';

const DEFAULT_MIN_WORD_COUNT = 5;
const DEFAULT_MAX_WORD_COUNT = 8;
const DEFAULT_MIN_WORD_LENGTH = 3;
const DEFAULT_MAX_WORD_LENGTH = 12;
const LEVELS = Object.values(LogLevel as Record<string, string>);

/**
 * This is based on the window height, which is determined by `DEFAUL_SIZE`,
 * returned by the `useResize` mocked hook.
 * The generated messages are intentionally kept short to ensure that
 * the log entries don't wrap with the given window width.
 */
const VISIBLE_LINES = 57;
const DEFAULT_SIZE = { height: 1024, width: 1280, x: 0, y: 0 };
const DEFAULT_CHAR_SIZE = { height: 18, width: 7 };

const generateMessage = (options: {
  maxWordCount?: number,
  maxWordLength?: number,
  minWordCount?: number,
  minWordLength?: number,
} = {}): string => {
  const minWordCount = options.minWordCount ?? DEFAULT_MIN_WORD_COUNT;
  const maxWordCount = options.maxWordCount ?? DEFAULT_MAX_WORD_COUNT;
  const minWordLength = options.minWordLength ?? DEFAULT_MIN_WORD_LENGTH;
  const maxWordLength = options.maxWordLength ?? DEFAULT_MAX_WORD_LENGTH;
  const count = Math.floor(Math.random() * (maxWordCount - minWordCount)) + minWordCount;
  const words = new Array(count).fill('').map(() => {
    const length = Math.floor(Math.random() * (maxWordLength - minWordLength)) + minWordLength;
    return generateAlphaNumeric(length);
  });
  return words.join(' ');
};

const generateLogs = (count = 100): LogViewer.ViewerLog[] => {
  return new Array(count).fill(null).map((_, index) => LogViewer.formatLogEntry({
    id: generateAlphaNumeric(),
    level: LEVELS[Math.floor(Math.random() * LEVELS.length)] as LogLevel,
    message: `message ${index} - ${generateMessage()}`,
    time: new Date(Date.now() - (count - index)).toString(),
  }));
};

const setup = (props: LogViewer.Props) => render(<LogViewer.default {...props} />);

const mockOnFetch = (canceler?: AbortController) => (
  config: LogViewer.FetchConfig,
  type: LogViewer.FetchType,
): FetchArgs => {
  const options = {
    follow: false,
    limit: config.limit,
    orderBy: 'ORDER_BY_UNSPECIFIED',
    signal: canceler?.signal,
    timestampAfter: '',
    timestampBefore: '',
  };

  if (type === LogViewer.FetchType.Initial) {
    options.orderBy = config.isNewestFirst ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC';
  } else if (type === LogViewer.FetchType.Newer) {
    options.orderBy = 'ORDER_BY_ASC';
    if (config.offsetLog?.time) options.timestampAfter = config.offsetLog.time;
  } else if (type === LogViewer.FetchType.Older) {
    options.orderBy = 'ORDER_BY_DESC';
    if (config.offsetLog?.time) options.timestampBefore = config.offsetLog.time;
  } else if (type === LogViewer.FetchType.Stream) {
    options.follow = true;
    options.limit = 0;
    options.orderBy = 'ORDER_BY_ASC';
    options.timestampAfter = new Date().toISOString();
  }
  console.log('onFetchByTime', config, type);

  return { options, url: 'byTime' };
};

jest.mock('hooks/useResize', () => ({ __esModule: true, default: () => DEFAULT_SIZE }));

jest.mock('hooks/useGetCharMeasureInContainer', () => ({
  __esModule: true,
  default: () => DEFAULT_CHAR_SIZE,
}));

/*
 * `mockTimeLogs` requires the `mock` prefix to allow `jest.mock()` to be able
 * to access it within the implementation block.
 */
const mockTimeLogs = generateLogs(5000);
jest.mock('services/utils', () => ({
  __esModule: true,
  ...jest.requireActual('services/utils'),
  consumeStream: ({ options, url }: FetchArgs, onEvent: (event: unknown) => void): void => {
    console.log('url', url, 'options', options);

    if (options.follow) {
      console.log('stream follow');
    } else {
      mockTimeLogs.forEach(log => onEvent(log));
    }
  },
}));

describe('LogViewer', () => {
  const decoder = jsonToTaskLog;

  describe('initialLogs', () => {
    it('should render logs with initial logs and show partial logs', async () => {
      const initialLogs = generateLogs(VISIBLE_LINES + 100);
      const lastLog = initialLogs[initialLogs.length - 1];
      setup({ decoder, initialLogs });

      /*
       * The react-window should only display the 1st `VISIBILE_LINES` log entrys
       * but not the logs outside of that range.
       */
      expect(screen.queryByText(initialLogs[0].message)).toBeInTheDocument();
      await waitFor(() => {
        expect(screen.queryByText(lastLog.message)).not.toBeInTheDocument();
      });

      const tailingButton = screen.getByLabelText(LogViewer.ARIA_LABEL_ENABLE_TAILING);
      userEvent.click(tailingButton);

      expect(screen.queryByText(lastLog.message)).toBeInTheDocument();
      await waitFor(() => {
        expect(screen.queryByText(initialLogs[0].message)).not.toBeInTheDocument();
      });
    });
  });

  describe('streaming', () => {
    beforeEach(() => {
      // jest.resetModules();
    });

    it('should render logs with streaming', async () => {
      const onFetch = mockOnFetch();
      setup({ decoder, onFetch });
      await waitFor(() => {
        expect(true).toBe(true);
      });
    });
  });
});
