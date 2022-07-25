import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Button } from 'antd';
import React, { useEffect } from 'react';

import StoreProvider, { StoreAction, useStoreDispatch } from 'contexts/Store';
import { generateTestExperimentData } from 'storybook/shared/generateTestExperiments';

import useModalHyperparameterSearch from './useModalHyperparameterSearch';

const MODAL_TITLE = 'Hyperparameter Search';

jest.mock('services/api', () => ({ getResourcePools: () => Promise.resolve([]) }
));

const { experiment } = generateTestExperimentData();

const ModalTrigger: React.FC = () => {
  const storeDispatch = useStoreDispatch();

  const {
    contextHolder,
    modalOpen,
  } = useModalHyperparameterSearch({ experiment: experiment });

  useEffect(() => {
    storeDispatch({ type: StoreAction.SetAuth, value: { isAuthenticated: true } });
  }, [ storeDispatch ]);

  return (
    <>
      <Button onClick={() => modalOpen()}>
        Open Modal
      </Button>
      {contextHolder}
    </>
  );
};

const Container: React.FC = () => {
  return (
    <StoreProvider>
      <ModalTrigger />
    </StoreProvider>
  );
};

const setup = async () => {
  const user = userEvent.setup();

  const view = render(<Container />);
  await user.click(screen.getByRole('button', { name: 'Open Modal' }));

  return { user, view };
};

describe('useModalExperimentCreate', () => {
  it('modal can be opened', async () => {
    const { view } = await setup();

    expect(await view.findByText(MODAL_TITLE)).toBeInTheDocument();
  });
});
