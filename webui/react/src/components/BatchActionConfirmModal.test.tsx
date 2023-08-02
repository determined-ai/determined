import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';

import Button from 'components/kit/Button';
import { DEFAULT_CANCEL_LABEL, useModal } from 'components/kit/Modal';

import { ExperimentAction as Action, ExperimentAction } from '../types';

import BatchActionConfirmModalComponent from './BatchActionConfirmModal';

interface Props {
  action: ExperimentAction;
}

const handleConfirm = vi.fn();
const handleCancel = vi.fn();

const ModalTrigger: React.FC<Props> = ({ action }) => {
  const BatchActionConfirmModal = useModal(BatchActionConfirmModalComponent);

  return (
    <>
      <Button onClick={BatchActionConfirmModal.open} />
      <BatchActionConfirmModal.Component
        batchAction={action}
        onClose={handleCancel}
        onConfirm={handleConfirm}
      />
    </>
  );
};

const user = userEvent.setup();

const setup = async (action: ExperimentAction) => {
  render(<ModalTrigger action={action} />);

  await user.click(screen.getByRole('button'));
};

const actionList = [
  Action.OpenTensorBoard,
  Action.Activate,
  Action.Move,
  Action.Pause,
  Action.Archive,
  Action.Unarchive,
  Action.Cancel,
  Action.Kill,
  Action.Delete,
];

describe('Batch Action Confirmation Modal', () => {
  it.each(actionList)('renders %s batch action confirm modal', async (action) => {
    await setup(action);
    expect(await screen.findByText(action)).toBeInTheDocument();
  });

  it('calls confirm handler', async () => {
    const action = ExperimentAction.Pause;

    await setup(action);
    await user.click(screen.getByRole('button', { name: action }));

    expect(handleConfirm).toHaveBeenCalled();
  });

  it('calls cancel handler', async () => {
    const action = ExperimentAction.Pause;

    await setup(action);
    await user.click(screen.getByRole('button', { name: DEFAULT_CANCEL_LABEL }));

    expect(handleCancel).toHaveBeenCalled();
  });
});
