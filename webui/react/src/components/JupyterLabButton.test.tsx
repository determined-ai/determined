import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import UIProvider, { DefaultTheme } from 'hew/Theme';

import { ThemeProvider } from 'components/ThemeProvider';

import JupyterLabButton from './JupyterLabButton';

const SIMPLE_CONFIG_TEMPLATE_TEXT = 'Template';

vi.mock('services/api', () => ({
  getTaskTemplates: () => Promise.resolve([]),
  getWorkspaces: () => Promise.resolve({ workspaces: [] }),
}));

vi.mock('hooks/useSettings', async (importOriginal) => {
  const useSettings = vi.fn(() => {
    const settings = {
      jupyterLab: {
        alt: false,
        ctrl: false,
        key: 'L',
        meta: true,
        shift: true,
      },
    };
    return { isLoading: false, settings };
  });

  return {
    __esModule: true,
    ...(await importOriginal<typeof import('hooks/useSettings')>()),
    useSettings,
  };
});

const user = userEvent.setup();

const setup = (enabled: boolean) => {
  render(
    <UIProvider theme={DefaultTheme.Light}>
      <ThemeProvider>
        <JupyterLabButton enabled={enabled} />
      </ThemeProvider>
    </UIProvider>,
  );
};

describe('Dashboard', () => {
  it('shows disabled state', () => {
    setup(false);
    expect(screen.getByRole('button')).toBeDisabled();
  });

  it('opens JupyterLabModal', async () => {
    setup(true);
    await user.click(screen.getByRole('button'));
    expect(screen.getByText(SIMPLE_CONFIG_TEMPLATE_TEXT)).toBeInTheDocument();
  });
});
