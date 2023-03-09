import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React, { useEffect } from 'react';
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

const MonacoEditorMock: React.FC = () => <></>;

jest.mock('services/api', () => ({
  getResourcePools: () => Promise.resolve([]),
  getTaskTemplates: () => Promise.resolve([]),
  getUsers: () => Promise.resolve({ users: [] }),
  getUserSetting: () => Promise.resolve({ settings: [] }),
  launchJupyterLab: () => Promise.resolve({ config: '' }),
}));

jest.mock('stores/cluster', () => {
  // eslint-disable-next-line @typescript-eslint/no-var-requires
  const loadable = require('utils/loadable');
  // eslint-disable-next-line @typescript-eslint/no-var-requires
  const observable = require('utils/observable');

  const store = { resourcePools: observable.observable(loadable.Loaded([])) };
  return {
    __esModule: true,
    ...jest.requireActual('stores/cluster'),
    useClusterStore: () => store,
  };
});

jest.mock('utils/wait', () => ({
  openCommand: () => null,
  waitPageUrl: () => '',
}));

jest.mock('components/MonacoEditor', () => ({
  __esModule: true,
  default: () => MonacoEditorMock,
}));

const ModalTrigger: React.FC = () => {
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
    setAuth({ isAuthenticated: true });
    setAuthChecked();
  }, []);

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

  const button = await waitFor(() => screen.findByRole('button'));
  user.click(button);

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
