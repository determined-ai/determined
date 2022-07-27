import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Button } from 'antd';
import React, { useEffect } from 'react';

import StoreProvider, { StoreAction, useStoreDispatch } from 'contexts/Store';
import { V1FittingPolicy, V1ResourcePoolType, V1SchedulerType } from 'services/api-ts-sdk';
import { CreateExperimentParams } from 'services/types';
import { generateTestExperimentData } from 'storybook/shared/generateTestExperiments';
import { ResourceType } from 'types';

import useModalHyperparameterSearch from './useModalHyperparameterSearch';

const MODAL_TITLE = 'Hyperparameter Search';

const mockCreateExperiment = jest.fn();

jest.mock('services/api', () => ({
  createExperiment: (params: CreateExperimentParams) => {
    return mockCreateExperiment(params);
  },
  getResourcePools: () => Promise.resolve([]),
}
));

const { experiment } = generateTestExperimentData();

const ModalTrigger: React.FC = () => {
  const storeDispatch = useStoreDispatch();

  const {
    contextHolder,
    modalOpen,
  } = useModalHyperparameterSearch({ experiment: experiment });

  useEffect(() => {
    storeDispatch({
      type: StoreAction.SetResourcePools,
      value: [ {
        agentDockerImage: '',
        agentDockerNetwork: '',
        agentDockerRuntime: '',
        agentFluentImage: '',
        auxContainerCapacity: 0,
        auxContainerCapacityPerAgent: 0,
        auxContainersRunning: 0,
        containerStartupScript: '',
        defaultAuxPool: false,
        defaultComputePool: true,
        description: '',
        details: {},
        imageId: '',
        instanceType: '',
        location: '',
        masterCertName: '',
        masterUrl: '',
        maxAgents: 1,
        maxAgentStartingPeriod: 1000,
        maxIdleAgentPeriod: 1000,
        minAgents: 0,
        name: 'default',
        numAgents: 1,
        preemptible: false,
        schedulerFittingPolicy: V1FittingPolicy.UNSPECIFIED,
        schedulerType: V1SchedulerType.UNSPECIFIED,
        slotsAvailable: 1,
        slotsUsed: 0,
        slotType: ResourceType.CUDA,
        startupScript: '',
        type: V1ResourcePoolType.UNSPECIFIED,
      } ],
    });
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

const user = userEvent.setup();

const setup = async () => {
  const view = render(<Container />);
  await user.click(screen.getByRole('button', { name: 'Open Modal' }));

  return { view };
};

describe('useModalHyperparameterSearch', () => {
  it('should open modal', async () => {
    const { view } = await setup();

    expect(await view.findByText(MODAL_TITLE)).toBeInTheDocument();
  });
  it('should cancel modal', async () => {
    const { view } = await setup();

    await user.click(view.getAllByRole('button', { name: 'Cancel' })[0]);

    // Check for the modal to be dismissed.
    await waitFor(() => {
      expect(view.queryByText(MODAL_TITLE)).not.toBeInTheDocument();
    });
  });
  it('should submit experiment', async () => {
    const { view } = await setup();

    await user.click(view.getByRole('button', { name: 'Select Hyperparameters' }));
    mockCreateExperiment.mockReturnValue({ id: 1 });
    await user.click(view.getByRole('button', { name: 'Run Experiment' }));

    expect(mockCreateExperiment).toHaveBeenCalled();
  });
});
