import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Button } from 'antd';
import React, { useEffect } from 'react';
import { BrowserRouter } from 'react-router-dom';

import StoreProvider, { StoreAction, useStoreDispatch } from 'contexts/Store';

import useModalJupyterLab from './useModalJupyterLab';

const MODAL_TITLE = 'Launch JupyterLab';
const SIMPLE_CONFIG_TEMPLATE_TEXT = 'Template';
const SHOW_SIMPLE_CONFIG_TEXT = 'Show Simple Config';

const MonacoEditorMock: React.FC = () => <></>;

jest.mock('services/api', () => ({
  getResourcePools: () => Promise.resolve([]),
  getTaskTemplates: () => Promise.resolve([]),
  launchJupyterLab: () => Promise.resolve({ config: '' }),
}));

jest.mock('utils/wait', () => ({
  openCommand: () => null,
  waitPageUrl: () => '',
}));

jest.mock('components/MonacoEditor', () => ({
  __esModule: true,
  default: () => MonacoEditorMock,
}));

const ModalTrigger: React.FC = () => {
  const storeDispatch = useStoreDispatch();
  const { contextHolder, modalOpen } = useModalJupyterLab();

  useEffect(() => {
    storeDispatch({ type: StoreAction.SetAuth, value: { isAuthenticated: true } });
  }, [storeDispatch]);

  return (
    <>
      <Button onClick={() => modalOpen()}>Show Jupyter Lab</Button>
      {contextHolder}
    </>
  );
};

const setup = async () => {
  const user = userEvent.setup();

  render(
    <BrowserRouter>
      <StoreProvider>
        <ModalTrigger />
      </StoreProvider>
    </BrowserRouter>,
  );

  await user.click(screen.getByRole('button'));

  return user;
};

describe('useModalJupyterLab', () => {
  it('should open modal', async () => {
    await setup();

    expect(await screen.findByText(MODAL_TITLE)).toBeInTheDocument();
  });

  it('should show modal in simple form mode', async () => {
    await setup();

    expect(await screen.findByText(SIMPLE_CONFIG_TEMPLATE_TEXT)).toBeInTheDocument();
  });

  it('should switch modal to full config', async () => {
    const user = await setup();

    await screen.findByText(MODAL_TITLE);

    await user.click(screen.getByRole('button', { name: /Show Full Config/i }));

    await waitFor(() => {
      expect(screen.queryByText(SHOW_SIMPLE_CONFIG_TEXT)).toBeInTheDocument();
    });
  });

  it('should close modal', async () => {
    const user = await setup();

    await screen.findByText(MODAL_TITLE);

    await user.click(screen.getByRole('button', { name: /Launch/i }));

    await waitFor(() => {
      expect(screen.queryByText(MODAL_TITLE)).not.toBeInTheDocument();
    });
  });
});
