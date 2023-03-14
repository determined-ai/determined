import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React, { useEffect } from 'react';

import Button from 'components/kit/Button';
import { setAuth } from 'stores/auth';
import { generateTestExperimentData } from 'storybook/shared/generateTestData';

import useModalExperimentCreate, { CreateExperimentType } from './useModalExperimentCreate';

const MODAL_TITLE = 'Fork';
const SHOW_FULL_CONFIG_TEXT = 'Show Full Config';

const MonacoEditorMock: React.FC = () => <></>;

jest.mock('services/api', () => ({
  getResourcePools: () => Promise.resolve([]),
  getTaskTemplates: () => Promise.resolve([]),
  launchJupyterLab: () => Promise.resolve({ config: '' }),
}));

jest.mock('components/MonacoEditor', () => ({
  __esModule: true,
  default: () => MonacoEditorMock,
}));

const ModalTrigger: React.FC = () => {
  const { contextHolder, modalOpen } = useModalExperimentCreate();
  const { experiment, trial } = generateTestExperimentData();
  useEffect(() => {
    setAuth({ isAuthenticated: true });
  }, []);

  return (
    <>
      <Button
        onClick={() =>
          modalOpen({ experiment: experiment, trial: trial, type: CreateExperimentType.Fork })
        }>
        Show Jupyter Lab
      </Button>
      {contextHolder}
    </>
  );
};

const Container: React.FC = () => <ModalTrigger />;

const setup = async () => {
  const user = userEvent.setup();

  render(<Container />);

  await user.click(screen.getByRole('button'));
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
