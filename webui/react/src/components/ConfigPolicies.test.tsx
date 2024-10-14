import { render, screen } from '@testing-library/react';
import UIProvider, { DefaultTheme } from 'hew/Theme';
import { ConfirmationProvider } from 'hew/useConfirm';

import ConfigPolicies from './ConfigPolicies';
import { ThemeProvider } from './ThemeProvider';

const mocks = vi.hoisted(() => {
  return {
    canModifyWorkspaceConfigPolicies: false,
  };
});

vi.mock('hooks/usePermissions', () => {
  const usePermissions = vi.fn(() => {
    return {
      canModifyWorkspaceConfigPolicies: mocks.canModifyWorkspaceConfigPolicies,
    };
  });
  return {
    default: usePermissions,
  };
});

vi.mock('@uiw/react-codemirror', () => ({
  __esModule: true,
  default: () => <></>,
}));

vi.mock('services/api', () => ({
  getWorkspaceConfigPolicies: () => Promise.resolve({ configPolicies: {} }),
}));

const setup = () => {
  render(
    <UIProvider theme={DefaultTheme.Light}>
      <ThemeProvider>
        <ConfirmationProvider>
          <ConfigPolicies workspaceId={1} />
        </ConfirmationProvider>
      </ThemeProvider>
    </UIProvider>,
  );
};

describe('Config Policies', () => {
  it('allows changes with permissions', async () => {
    mocks.canModifyWorkspaceConfigPolicies = true;
    setup();
    expect(await screen.findByRole('button', { name: 'Apply' })).toBeInTheDocument();
  });

  it('prevents changes without permissions', () => {
    mocks.canModifyWorkspaceConfigPolicies = false;
    setup();
    expect(screen.queryByRole('button', { name: 'Apply' })).not.toBeInTheDocument();
  });
});
