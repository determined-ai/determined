import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React, { useEffect } from 'react';
import { BrowserRouter } from 'react-router-dom';

import JupyterLabModalComponent from 'components/JupyterLabModal';
import Button from 'components/kit/Button';
import { StoreProvider as UIProvider } from 'components/kit/contexts/UI';
import { useModal } from 'components/kit/Modal';
import { SettingsProvider } from 'hooks/useSettingsProvider';
import authStore from 'stores/auth';
import { WorkspaceState } from 'types';

const SIMPLE_CONFIG_TEMPLATE_TEXT = 'Template';
const SHOW_SIMPLE_CONFIG_TEXT = 'Show Simple Config';

vi.mock('services/api', () => ({
  getCurrentUser: () => Promise.resolve({ id: 1 }),
  getResourcePools: () => Promise.resolve([]),
  getTaskTemplates: () => Promise.resolve([]),
  getUsers: () => Promise.resolve({ users: [] }),
  getUserSetting: () => Promise.resolve({ settings: [] }),
  getWorkspaces: () => Promise.resolve({ workspaces: [] }),
  launchJupyterLab: () => Promise.resolve({ config: '' }),
  previewJupyterLab: () =>
    Promise.resolve({
      description: 'JupyterLab (freely-distinct-mustang)',
    }),
}));

vi.mock('stores/cluster', async (importOriginal) => {
  const loadable = await import('utils/loadable');
  const observable = await import('utils/observable');

  const store = { resourcePools: observable.observable(loadable.Loaded([])) };
  return {
    __esModule: true,
    ...(await importOriginal<typeof import('stores/cluster')>()),
    clusterStore: store,
  };
});

vi.mock('utils/wait', () => ({
  openCommand: () => null,
  waitPageUrl: () => '',
}));

vi.mock('components/kit/CodeEditor', () => ({
  __esModule: true,
  default: () => <></>,
}));

const ModalTrigger: React.FC = () => {
  const JupyterLabModal = useModal(JupyterLabModalComponent);

  useEffect(() => {
    authStore.setAuth({ isAuthenticated: true });
    authStore.setAuthChecked();
  }, []);

  return (
    <SettingsProvider>
      <>
        <Button onClick={JupyterLabModal.open}>Show Jupyter Lab</Button>
        <JupyterLabModal.Component
          workspace={{
            archived: false,
            id: 1,
            immutable: false,
            name: 'Uncategorized',
            numExperiments: 0,
            numProjects: 0,
            pinned: false,
            state: WorkspaceState.Unspecified,
            userId: 1,
          }}
        />
      </>
    </SettingsProvider>
  );
};

const setup = async () => {
  const user = userEvent.setup();

  render(
    <BrowserRouter>
      <UIProvider>
        <ModalTrigger />
      </UIProvider>
    </BrowserRouter>,
  );

  await user.click(await screen.findByRole('button'));

  return user;
};

describe('JupyterLab Modal', () => {
  it('should show modal in simple form mode', async () => {
    await setup();

    expect(await screen.findByText(SIMPLE_CONFIG_TEMPLATE_TEXT)).toBeInTheDocument();
  });

  it('should switch modal to full config', async () => {
    const user = await setup();

    const button = await screen.findByRole('button', { name: /Show Full Config/i });
    await user.click(button);

    await waitFor(() => {
      expect(screen.queryByText(SHOW_SIMPLE_CONFIG_TEXT)).toBeInTheDocument();
    });
  });
});
