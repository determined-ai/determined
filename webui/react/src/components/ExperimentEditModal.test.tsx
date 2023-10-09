import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';

import ExperimentEditModalComponent, {
  BUTTON_TEXT,
  DESCRIPTION_LABEL,
  NAME_LABEL,
} from 'components/ExperimentEditModal';
import Button from 'components/kit/Button';
import { useModal } from 'components/kit/Modal';
import { patchExperiment as mockPatchExperiment } from 'services/api';
import { generateTestExperimentData } from 'utils/tests/generateTestData';

const user = userEvent.setup();

vi.mock('services/api', () => ({
  patchExperiment: vi.fn(),
}));

const { experiment } = generateTestExperimentData();
const callback = vi.fn();

const ModalTrigger: React.FC = () => {
  const ExperimentEditModal = useModal(ExperimentEditModalComponent);

  return (
    <>
      <Button onClick={ExperimentEditModal.open} />
      <ExperimentEditModal.Component
        description={experiment.description ?? ''}
        experimentId={experiment.id}
        experimentName={experiment.name}
        onEditComplete={callback}
      />
    </>
  );
};

const setup = async () => {
  render(<ModalTrigger />);

  await user.click(screen.getByRole('button'));
};

describe('Edit Experiment Modal', () => {
  it('submits a valid edit experiment request', async () => {
    await setup();

    const addition = 'ASDF';

    await user.type(screen.getByLabelText(NAME_LABEL), addition);
    await user.type(screen.getByLabelText(DESCRIPTION_LABEL), addition);

    await user.click(screen.getByRole('button', { name: BUTTON_TEXT }));

    expect(mockPatchExperiment).toHaveBeenCalledWith({
      body: {
        description: (experiment.description ?? '') + addition,
        name: (experiment.name ?? '') + addition,
      },
      experimentId: experiment.id,
    });

    expect(callback).toHaveBeenCalled();
  });
});
