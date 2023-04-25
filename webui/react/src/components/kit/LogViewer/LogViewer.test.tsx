import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import { FetchArgs } from 'services/api-ts-sdk';
import { mapV1LogsResponse } from 'services/decoder';
import { StoreProvider as UIProvider } from 'shared/contexts/stores/UI';
import { generateAlphaNumeric } from 'shared/utils/string';
import { LogLevelFromApi } from 'types';

import * as src from './LogViewer';

interface TestLog {
  id: number | string;
  level?: string;
  message: string;
  time: string;
}

const DEFAULT_MIN_WORD_COUNT = 5;
const DEFAULT_MAX_WORD_COUNT = 8;
const DEFAULT_MIN_WORD_LENGTH = 3;
const DEFAULT_MAX_WORD_LENGTH = 12;
const LEVELS = Object.values(LogLevelFromApi) as string[];
const NOW = Date.now();

/**
 * This is based on the window height, which is determined by `DEFAUL_SIZE`,
 * returned by the `useResize` mocked hook.
 * The generated messages are intentionally kept short to ensure that
 * the log entries don't wrap with the given window width.
 */
const VISIBLE_LINES = 57;

const user = userEvent.setup();

const generateMessage = (
  options: {
    maxWordCount?: number;
    maxWordLength?: number;
    minWordCount?: number;
    minWordLength?: number;
  } = {},
): string => {
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

const generateLogs = (
  count = 1,
  startIndex = 0,
  nowIndex?: number, // when undefined, assumed the last generated log is now
): TestLog[] => {
  const dateIndex = nowIndex != null ? nowIndex : count - 1;
  return new Array(count).fill(null).map((_, i) => {
    const index = startIndex + i;
    const timeOffset = (dateIndex - index) * 1000;
    const timestamp = NOW - timeOffset;
    return {
      id: generateAlphaNumeric(),
      level: LEVELS[Math.floor(Math.random() * LEVELS.length)],
      message: `index: ${index} - timestamp: ${timestamp} - ${generateMessage()}`,
      time: new Date(timestamp).toString(),
    };
  });
};

const setup = (props: src.Props) => {
  return render(
    <UIProvider>
      <src.default {...props} />
      {/* increase variation in DOM */}
      <span>{Math.random()}</span>
    </UIProvider>,
  );
};

/**
 * canceler -        AbortController to manually stop ongoing API calls.
 * logsReference -   Allows tests to pass in an array to reflect the current state of loaded logs.
 * skipStreaming -   Disables the streaming portion of the mocked `readStream` function.
 * streamingRounds - How many rounds of stream chunks to simulate.
 */
const mockOnFetch =
  (
    mockOptions: {
      canceler?: AbortController;
      existingLogs?: TestLog[];
      logsReference?: TestLog[];
      skipStreaming?: boolean;
      streamingRounds?: number;
    } = {},
  ) =>
  (config: src.FetchConfig, type: src.FetchType): FetchArgs => {
    const options = {
      existingLogs: mockOptions.existingLogs,
      follow: false,
      limit: config.limit,
      logsReference: mockOptions.logsReference,
      orderBy: 'ORDER_BY_UNSPECIFIED',
      signal: mockOptions.canceler?.signal,
      skipStreaming: mockOptions.skipStreaming,
      streamingRounds: mockOptions.streamingRounds,
      timestampAfter: '',
      timestampBefore: '',
    };

    if (type === src.FetchType.Initial) {
      options.orderBy =
        config.fetchDirection === src.FetchDirection.Older ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC';
    } else if (type === src.FetchType.Newer) {
      options.orderBy = 'ORDER_BY_ASC';
      if (config.offsetLog?.time) options.timestampAfter = config.offsetLog.time;
    } else if (type === src.FetchType.Older) {
      options.orderBy = 'ORDER_BY_DESC';
      if (config.offsetLog?.time) options.timestampBefore = config.offsetLog.time;
    } else if (type === src.FetchType.Stream) {
      options.follow = true;
      options.limit = 0;
      options.orderBy = 'ORDER_BY_ASC';
      options.timestampAfter = new Date(NOW).toISOString();
    }

    return { options, url: 'byTime' };
  };

const findTimeLogIndex = (logs: TestLog[], timeString: string): number => {
  const timestamp = new Date(timeString).getTime().toString();
  return logs.findIndex((log) => log.message.includes(timestamp));
};

vi.mock('hooks/useResize', () => ({
  __esModule: true,
  default: () => ({ height: 1824, width: 1280, x: 0, y: 0 }),
}));

vi.mock('hooks/useGetCharMeasureInContainer', () => ({
  __esModule: true,
  default: () => ({ height: 18, width: 7 }),
}));

vi.mock('services/utils', () => ({
  __esModule: true,
  readStream: ({ options }: FetchArgs, onEvent: (event: unknown) => void): void => {
    // Default mocking options.
    const existingLogs = options.existingLogs ?? [];
    const skipStreaming = options.skipStreaming ?? true;
    const streamingRounds = options.streamingRounds ?? 100;
    const desc = options.orderBy === 'ORDER_BY_DESC';

    if (!options.follow) {
      const range = [0, existingLogs.length - 1];
      if (desc) {
        if (options.timestampBefore) {
          const before = findTimeLogIndex(existingLogs, options.timestampBefore);
          range[0] = before - options.limit;
          range[1] = before;
        } else {
          range[0] = existingLogs.length - options.limit;
          range[1] = existingLogs.length;
        }
      } else {
        if (options.timestampAfter) {
          const after = findTimeLogIndex(existingLogs, options.timestampAfter);
          range[0] = after + 1;
          range[1] = after + options.limit + 1;
        } else {
          range[0] = 0;
          range[1] = options.limit;
        }
      }
      const filteredLogs: TestLog[] = existingLogs.slice(range[0], range[1]);
      if (desc) filteredLogs.reverse();
      if (options.logsReference) options.logsReference.push(...filteredLogs);
      filteredLogs.forEach((log) => onEvent(log));
    } else if (options.follow && !skipStreaming) {
      let startIndex = existingLogs.length;
      let rounds = 0;
      while (rounds < streamingRounds) {
        const count = Math.floor(Math.random() * 4) + 1;
        const logs = generateLogs(count, startIndex, existingLogs.length - 1);
        if (options.logsReference) options.logsReference.push(...logs);
        logs.forEach((log) => onEvent(log));
        startIndex += count;
        rounds++;
      }
    }
  },
}));

describe('LogViewer', () => {
  const decoder = mapV1LogsResponse;

  describe('static logs', () => {
    it('should render logs with initial logs and show partial logs', async () => {
      const initialLogs = generateLogs(VISIBLE_LINES + 100);
      const firstLog = initialLogs[0];
      const lastLog = initialLogs[initialLogs.length - 1];
      setup({ decoder, initialLogs });

      /*
       * The react-window should only display the 1st `VISIBILE_LINES` log entrys
       * but not the logs outside of that range.
       */
      expect(screen.queryByText(firstLog.message)).toBeInTheDocument();
      await waitFor(() => {
        expect(screen.queryByText(lastLog.message)).not.toBeInTheDocument();
      });

      const enableTailingButton = screen.getByLabelText(src.ARIA_LABEL_ENABLE_TAILING);
      await user.click(enableTailingButton);

      expect(screen.queryByText(lastLog.message)).toBeInTheDocument();
      await waitFor(() => {
        expect(screen.queryByText(firstLog.message)).not.toBeInTheDocument();
      });
    });

    it('should hide scrolling buttons when log content is empty', async () => {
      setup({ decoder, initialLogs: [] });

      await waitFor(() => {
        expect(screen.queryByLabelText(src.ARIA_LABEL_SCROLL_TO_OLDEST)).not.toBeVisible();
        expect(screen.queryByLabelText(src.ARIA_LABEL_ENABLE_TAILING)).not.toBeVisible();
      });
    });

    it('should not show log close button by default', () => {
      const { container } = setup({ decoder });

      const icon = container.querySelector('.icon-close');
      expect(icon).toBeNull();
      expect(icon).not.toBeInTheDocument();
    });

    it('should show log close button when prop is supplied', () => {
      const handleCloseLogs = () => {
        return;
      };
      const { container } = setup({ decoder, handleCloseLogs });

      const icon = container.querySelector('.icon-close');
      expect(icon).not.toBeNull();
      expect(icon).toBeInTheDocument();
    });
  });

  describe('streaming logs', () => {
    const streamingRounds = 5;
    const existingLogCount = 5000;
    let canceler: AbortController;
    let existingLogs: TestLog[];
    let logsReference: TestLog[];
    let onFetch: (config: src.FetchConfig, type: src.FetchType) => FetchArgs;

    beforeEach(() => {
      canceler = new AbortController();
      existingLogs = generateLogs(existingLogCount, 0, existingLogCount - 1);
      logsReference = [];
      onFetch = mockOnFetch({
        canceler,
        existingLogs,
        logsReference,
        skipStreaming: false,
        streamingRounds,
      });
    });

    it('should render logs with streaming', async () => {
      setup({ decoder, onFetch });

      await waitFor(
        () => {
          const lastLog = logsReference[logsReference.length - 1];
          expect(lastLog.message).not.toBeNull();
          expect(screen.queryByText(lastLog.message)).toBeInTheDocument();
        },
        { timeout: 6000 },
      );
    }, 6500);

    it('should show oldest logs', async () => {
      setup({ decoder, onFetch });

      await waitFor(() => {
        const lastLog = logsReference[logsReference.length - 1];
        expect(screen.queryByText(lastLog.message)).toBeInTheDocument();
      });

      await waitFor(() => {
        const lastExistingLog = existingLogs[existingLogs.length - 1];
        expect(screen.queryByText(lastExistingLog.message)).toBeInTheDocument();
      });

      const scrollToOldestButton = screen.getByLabelText(src.ARIA_LABEL_SCROLL_TO_OLDEST);
      await user.click(scrollToOldestButton);

      await waitFor(() => {
        const firstLog = existingLogs[0];
        expect(screen.queryByText(firstLog.message)).toBeInTheDocument();
      });
    });

    it('should show newest logs when enabling tailing', async () => {
      setup({ decoder, onFetch });

      const scrollToOldestButton = screen.getByLabelText(src.ARIA_LABEL_SCROLL_TO_OLDEST);
      await user.click(scrollToOldestButton);

      await waitFor(() => {
        const firstLog = existingLogs[0];
        expect(screen.queryByText(firstLog.message)).toBeInTheDocument();
      });

      const enableTailingButton = screen.getByLabelText(src.ARIA_LABEL_ENABLE_TAILING);
      await user.click(enableTailingButton);

      await waitFor(() => {
        const lastLog = logsReference[logsReference.length - 1];
        expect(screen.queryByText(lastLog.message)).toBeInTheDocument();
      });
    });
  });
});
