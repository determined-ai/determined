import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent, { PointerEventsCheckLevel } from '@testing-library/user-event';
import React from 'react';

import Button from 'components/kit/Button';
import { createExperiment as mockCreateExperiment } from 'services/api';
import { generateTestExperimentData } from 'utils/tests/generateTestData';

import useModalHyperparameterSearch from './useModalHyperparameterSearch';

const MODAL_TITLE = 'Hyperparameter Search';

vi.mock('stores/cluster', async (importOriginal) => {
  const types = await import('types');
  const sdkTypes = await import('services/api-ts-sdk');
  const loadable = await import('utils/loadable');
  const observable = await import('utils/observable');

  const store = {
    fetchResourcePools: vi.fn(),
    resourcePools: observable.observable(
      loadable.Loaded([
        {
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
          schedulerFittingPolicy: sdkTypes.V1FittingPolicy.UNSPECIFIED,
          schedulerType: sdkTypes.V1SchedulerType.UNSPECIFIED,
          slotsAvailable: 1,
          slotsUsed: 0,
          slotType: types.ResourceType.CUDA,
          startupScript: '',
          type: sdkTypes.V1ResourcePoolType.UNSPECIFIED,
        },
      ]),
    ),
  };

  return {
    ...(await importOriginal<typeof import('stores/cluster')>()),
    default: store,
  };
});

vi.mock('services/api', () => ({
  createExperiment: vi.fn().mockReturnValue(
    Promise.resolve({
      experiment: { id: 1 },
      maxSlotsExceeded: false,
    }),
  ),
  getResourcePools: vi.fn().mockReturnValue(Promise.resolve([])),
}));

const { experiment } = generateTestExperimentData();

const ModalTrigger: React.FC = () => {
  const { contextHolder, modalOpen } = useModalHyperparameterSearch({ experiment: experiment });

  return (
    <>
      <Button onClick={() => modalOpen()}>Open Modal</Button>
      {contextHolder}
    </>
  );
};

const Container: React.FC = () => {
  return <ModalTrigger />;
};

const user = userEvent.setup({ pointerEventsCheck: PointerEventsCheckLevel.Never });

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
    await user.click(view.getByRole('button', { name: 'Run Experiment' }));

    expect(mockCreateExperiment).toHaveBeenCalled();
  });

  it('should only allow current on constant hyperparameter', async () => {
    const { view } = await setup();

    await user.click(view.getByRole('button', { name: 'Select Hyperparameters' }));

    await user.click(view.getAllByRole('combobox')[0]);
    await user.click(within(view.getAllByLabelText('Type')[0]).getByText('Constant'));

    expect(view.getAllByLabelText('Current')[0]).not.toBeDisabled();
    expect(view.getAllByLabelText('Min value')[0]).toBeDisabled();
    expect(view.getAllByLabelText('Max value')[0]).toBeDisabled();
  });

  it('should only allow min and max on int hyperparameter', async () => {
    const { view } = await setup();

    await user.click(view.getByRole('button', { name: 'Select Hyperparameters' }));

    await user.click(view.getAllByRole('combobox')[0]);
    await user.click(within(view.getAllByLabelText('Type')[0]).getByText('Int'));

    expect(view.getAllByLabelText('Current')[0]).toBeDisabled();
    expect(view.getAllByLabelText('Min value')[0]).not.toBeDisabled();
    expect(view.getAllByLabelText('Max value')[0]).not.toBeDisabled();
  });

  it('should show count fields when using grid searcher', async () => {
    const { view } = await setup();

    await user.click(view.getByRole('button', { name: /grid/i }));
    await user.click(view.getByRole('button', { name: 'Select Hyperparameters' }));

    expect(view.getByText('Grid Count')).toBeInTheDocument();
  });

  it('should remove adaptive fields when not using adaptive searcher', async () => {
    const { view } = await setup();

    await user.click(view.getByRole('button', { name: /adaptive/i }));
    expect(view.getByText(/Early stopping mode/i)).toBeInTheDocument();

    await user.click(view.getByRole('button', { name: /grid/i }));
    await waitFor(() => {
      expect(view.queryByText(/Early stopping mode/i)).not.toBeInTheDocument();
    });
  });
});
