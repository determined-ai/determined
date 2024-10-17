import { render } from '@testing-library/react';
import UIProvider, { DefaultTheme } from 'hew/Theme';

import VersionChecker from 'components/VersionChecker';

import { ThemeProvider } from './ThemeProvider';

const THEME_CLASS = 'ui-provider-test';
const OLDER_VERSION = '1';
const NEWER_VERSION = '2';

const mockWarning = vi.hoisted(() => vi.fn());
vi.mock('hew/Toast', () => ({
  notification: {
    warning: mockWarning,
  },
}));

vi.mock('hew/Theme', async (importOriginal) => {
  const useTheme = () => {
    return {
      themeSettings: {
        className: THEME_CLASS,
      },
    };
  };

  return {
    __esModule: true,
    ...(await importOriginal<typeof import('hew/Theme')>()),
    useTheme,
  };
});

const setup = () => {
  render(
    <UIProvider theme={DefaultTheme.Light}>
      <ThemeProvider>
        <VersionChecker version={NEWER_VERSION} />
      </ThemeProvider>
    </UIProvider>,
  );
};

describe('VersionChecker', () => {
  afterEach(() => {
    vi.unstubAllEnvs();
    mockWarning.mockReset();
  });

  it('shows warning if version mismatch in production mode', () => {
    vi.stubEnv('IS_DEV', 'false');
    vi.stubEnv('VERSION', OLDER_VERSION);
    setup();
    expect(mockWarning).toHaveBeenCalledWith(
      expect.objectContaining({
        className: THEME_CLASS,
        duration: 0,
        key: 'version-mismatch',
        message: 'New WebUI Version',
        placement: 'bottomRight',
      }),
    );
  });

  it('does not show warning in development mode', () => {
    vi.stubEnv('IS_DEV', 'true');
    vi.stubEnv('VERSION', OLDER_VERSION);
    setup();
    expect(mockWarning).not.toBeCalled();
  });

  it('does not show warning if version matches', () => {
    vi.stubEnv('IS_DEV', 'false');
    vi.stubEnv('VERSION', NEWER_VERSION);
    setup();
    expect(mockWarning).not.toBeCalled();
  });
});
