import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Button } from 'antd';
import React, { useEffect } from 'react';

import StoreProvider, { StoreAction, useStoreDispatch } from 'contexts/Store';

import useJupyterLabModal from './useJupyterLabModal';

const MODAL_TITLE = 'Launch JupyterLab';
const SIMPLE_CONFIG_TEMPLATE_TEXT = 'Template';

const MonacoEditorMock: React.FC = () => <></>;

jest.mock('services/api', () => ({
  getResourcePools: () => Promise.resolve([]),
  getTaskTemplates: () => Promise.resolve([]),
  launchJupyterLab: () => Promise.resolve({ config: '' }),
}
));

jest.mock('wait', () => ({
  openCommand: () => {
    return null;
  },
  waitPageUrl: () => '',
}
));

jest.mock('components/MonacoEditor', () => ({
  __esModule: true,
  default: () => {
    return MonacoEditorMock;
  },
}));

const ModalTrigger: React.FC = () => {

  const storeDispatch = useStoreDispatch();
  const { modalOpen } = useJupyterLabModal();

  useEffect(() => {
    storeDispatch({ type: StoreAction.SetAuth, value: { isAuthenticated: true } });
  }, [ storeDispatch ]);

  return (
    <Button onClick={() => modalOpen()}>Show Jupyter Lab</Button>
  );
};

const ModalTriggerContainer: React.FC = () => {
  return (
    <StoreProvider>
      <ModalTrigger />
    </StoreProvider>
  );
};

const setup = async () => {

  render(
    <ModalTriggerContainer />,
  );

  userEvent.click(await screen.findByRole('button'));
};

describe('useJupyterLabModal', () => {
  it('open modal', async () => {
    await setup();

    expect(await screen.findByText(MODAL_TITLE)).toBeInTheDocument();
  });

  it('modal defaults to simple config', async () => {
    await setup();

    expect(await screen.findByText(SIMPLE_CONFIG_TEMPLATE_TEXT)).toBeInTheDocument();
  });

  it('switch to full config', async () => {
    await setup();

    await screen.findByText(MODAL_TITLE);

    userEvent.click(screen.getByRole('button', { name: /Show Full Config/i }));

    await waitFor(() => {
      expect(screen.queryByText(SIMPLE_CONFIG_TEMPLATE_TEXT)).not.toBeInTheDocument();
    });
  });

  it('close modal', async () => {
    await setup();

    await screen.findByText(MODAL_TITLE);

    userEvent.click(screen.getByRole('button', { name: /Launch/i }));

    await waitFor(() => {
      expect(screen.queryByText(MODAL_TITLE)).not.toBeInTheDocument();
    });
  });

});
