import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React, { useState } from 'react';

import Button from 'components/kit/Button';
import { Modal, useModal } from 'components/kit/Modal';

const MODAL_TITLE = 'Modal Title';
const MODAL_CONTENT = 'Modal string value';

const ModalComponent: React.FC<{ value: string }> = ({ value }) => {
  return (
    <Modal title={MODAL_TITLE}>
      <div>{value}</div>
    </Modal>
  );
};

const ModalTrigger: React.FC = () => {
  const Modal = useModal(ModalComponent);
  const [modalValue, setModalValue] = useState<string>('');
  return (
    <>
      <Button
        onClick={() => {
          setModalValue(MODAL_CONTENT);
          Modal.open();
        }}>
        Open Modal
      </Button>
      <Modal.Component value={modalValue} />
    </>
  );
};

const setup = async () => {
  const user = userEvent.setup();

  render(<ModalTrigger />);

  await user.click(await screen.findByRole('button'));

  return user;
};

describe('Modal', () => {
  it('should open', async () => {
    await setup();

    expect(await screen.findByText(MODAL_TITLE)).toBeInTheDocument();
    expect(await screen.findByText(MODAL_CONTENT)).toBeInTheDocument();
  });

  it('should close', async () => {
    const user = await setup();

    const closeButton = await screen.findByLabelText('Close');
    await user.click(closeButton);

    await waitFor(() => {
      expect(screen.queryByText(MODAL_TITLE)).not.toBeInTheDocument();
      expect(screen.queryByText(MODAL_CONTENT)).not.toBeInTheDocument();
    });
  });
});
