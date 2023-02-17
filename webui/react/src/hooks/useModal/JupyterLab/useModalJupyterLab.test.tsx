import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React, { useEffect, useState } from 'react';
import { BrowserRouter } from 'react-router-dom';

import Button from 'components/kit/Button';
import { SettingsProvider } from 'hooks/useSettingsProvider';
import { StoreProvider as UIProvider } from 'shared/contexts/stores/UI';
import { setAuth, setAuthChecked } from 'stores/auth';
import { WorkspacesProvider } from 'stores/workspaces';
import { WorkspaceState } from 'types';

import useModalJupyterLab from './useModalJupyterLab';

const MODAL_TITLE = 'Launch JupyterLab';
const SIMPLE_CONFIG_TEMPLATE_TEXT = 'Template';
const SHOW_SIMPLE_CONFIG_TEXT = 'Show Simple Config';

vi.mock('services/api', () => ({
  getCurrentUser: () => Promise.resolve({ id: 1 }),
  getResourcePools: () => Promise.resolve([]),
  getTaskTemplates: () => Promise.resolve([]),
  getUsers: () => Promise.resolve({ users: [] }),
  getUserSetting: () => Promise.resolve({ settings: [] }),
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
    useClusterStore: () => store,
  };
});

vi.mock('utils/wait', () => ({
  openCommand: () => null,
  waitPageUrl: () => '',
}));

vi.mock('components/MonacoEditor', () => ({
  __esModule: true,
  default: () => <></>,
}));

const ModalTrigger: React.FC = () => {
  const [canceler] = useState(new AbortController());
  const ensureCurrentUserFetched = useEnsureCurrentUserFetched(canceler);
  const { contextHolder, modalOpen } = useModalJupyterLab({
    workspace: {
      archived: false,
      id: 1,
      immutable: false,
      name: 'Uncategorized',
      numExperiments: 0,
      numProjects: 0,
      pinned: false,
      state: WorkspaceState.Unspecified,
      userId: 1,
    },
  });

  useEffect(() => {
    ensureCurrentUserFetched();
    setAuth({ isAuthenticated: true });
    setAuthChecked();
  }, [ensureCurrentUserFetched]);

  return (
    <SettingsProvider>
      <>
        <Button onClick={() => modalOpen()}>Show Jupyter Lab</Button>
        {contextHolder}
      </>
    </SettingsProvider>
  );
};

const setup = async () => {
  const user = userEvent.setup();

  render(
    <BrowserRouter>
      <UIProvider>
        <WorkspacesProvider>
          <ModalTrigger />
        </WorkspacesProvider>
      </UIProvider>
    </BrowserRouter>,
  );

  const button = await screen.findByRole('button');
  await waitFor(() => user.click(button));

  return user;
};

describe('useModalJupyterLab', () => {
  it('should open modal', async () => {
    await setup();

    expect(await screen.findByText(MODAL_TITLE)).toBeInTheDocument();
  });

  it('should close modal', async () => {
    const user = await setup();

    await screen.findByText(MODAL_TITLE);
    const button = await screen.findByRole('button', { name: /Launch/i });
    user.click(button);

    await waitFor(() => {
      expect(screen.queryByText(MODAL_TITLE)).not.toBeInTheDocument();
    });
  });

  it('should show modal in simple form mode', async () => {
    await setup();

    expect(await screen.findByText(SIMPLE_CONFIG_TEMPLATE_TEXT)).toBeInTheDocument();
  });

  it('should switch modal to full config', async () => {
    const user = await setup();

    await screen.findByText(MODAL_TITLE);
    const button = await screen.findByRole('button', { name: /Show Full Config/i });
    user.click(button);

    await waitFor(() => {
      expect(screen.queryByText(SHOW_SIMPLE_CONFIG_TEXT)).toBeInTheDocument();
    });
  });
});
