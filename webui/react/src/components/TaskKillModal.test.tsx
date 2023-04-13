import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';

import Button from 'components/kit/Button';
import { useModal } from 'components/kit/Modal';
import TaskKillModalComponent, { BUTTON_TEXT } from 'components/TaskKillModal';
import { killTask as mockKillTask } from 'services/api';

import { generateTestTaskData } from '../utils/tests/generateTestData';

const user = userEvent.setup();

vi.mock('services/api', () => ({
  killTask: vi.fn(),
}));

const task = generateTestTaskData();

const ModalTrigger: React.FC = () => {
  const TaskKillModal = useModal(TaskKillModalComponent);

  return (
    <>
      <Button onClick={TaskKillModal.open} />
      <TaskKillModal.Component task={task} />
    </>
  );
};

const setup = async () => {
  render(<ModalTrigger />);

  await user.click(screen.getByRole('button'));
};

describe('Kill Task Modal', () => {
  it('submits a valid kill task request', async () => {
    await setup();

    await user.click(screen.getByRole('button', { name: BUTTON_TEXT }));

    expect(mockKillTask).toHaveBeenCalledWith(task);
  });
});
