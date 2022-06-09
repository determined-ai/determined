import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Button, Modal } from 'antd';
import React, { useEffect } from 'react';

import StoreProvider, { StoreAction, useStoreDispatch } from 'contexts/Store';
import { generateTestExperimentData } from 'storybook/shared/generateTestExperiments';

import useModalExperimentCreate, { CreateExperimentType } from './useModalExperimentCreate';

const MODAL_TITLE = 'Fork';
const SHOW_FULL_CONFIG_TEXT = 'Show Full Config';

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
  const [ createExperimentModal, createExperimentModalContextHolder ] = Modal.useModal();
  const { modalOpen } = useModalExperimentCreate({ modal: createExperimentModal });
  const { experiment, trial } = generateTestExperimentData();
  useEffect(() => {
    storeDispatch({ type: StoreAction.SetAuth, value: { isAuthenticated: true } });
  }, [ storeDispatch ]);

  return (
    <>
      {createExperimentModalContextHolder}
      <Button onClick={() =>
        modalOpen({ experiment: experiment, trial: trial, type: CreateExperimentType.Fork })}>
        Show Jupyter Lab
      </Button>
    </>
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

describe('useModalExperimentCreate', () => {
  it('modal can be opened', async () => {
    await setup();

    expect(await screen.findByText(MODAL_TITLE)).toBeInTheDocument();
  });

  it('modal defaults to simple config', async () => {
    await setup();

    expect(await screen.findByText(SHOW_FULL_CONFIG_TEXT)).toBeInTheDocument();
  });

});
