import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';

import Button from 'components/kit/Button';
import useConfirm, { ConfirmationProvider, voidFn } from 'components/kit/useConfirm';

const CONFIRM_TITLE = 'Really?!';
const CONFIRM_CONTENT = 'Do you really want to do this?!';

const handleConfirm = vi.fn();
const handleClose = vi.fn();

const Container: React.FC = () => {
  const confirm = useConfirm();

  return (
    <Button
      onClick={() =>
        confirm({
          content: CONFIRM_CONTENT,
          onClose: handleClose,
          onConfirm: handleConfirm,
          onError: voidFn,
          title: CONFIRM_TITLE,
        })
      }>
      Open Confirmation
    </Button>
  );
};

const user = userEvent.setup();

const setup = async () => {
  render(
    <ConfirmationProvider>
      <Container />
    </ConfirmationProvider>,
  );
  await user.click(await screen.findByRole('button'));
};

describe('Modal', () => {
  it('should open confirmation', async () => {
    await setup();

    expect(await screen.findByText(CONFIRM_TITLE)).toBeInTheDocument();
    expect(await screen.findByText(CONFIRM_CONTENT)).toBeInTheDocument();
  });

  it('should confirm confirmation', async () => {
    await setup();

    const confirmButton = await screen.findByRole('button', { name: 'Confirm' });
    await user.click(confirmButton);

    await waitFor(() => {
      expect(screen.queryByText(CONFIRM_TITLE)).not.toBeInTheDocument();
      expect(screen.queryByText(CONFIRM_CONTENT)).not.toBeInTheDocument();
    });

    expect(handleConfirm).toHaveBeenCalled();
  });

  it('should close confirmation', async () => {
    await setup();

    const cancelButton = await screen.findByRole('button', { name: 'Cancel' });
    await user.click(cancelButton);

    await waitFor(() => {
      expect(screen.queryByText(CONFIRM_TITLE)).not.toBeInTheDocument();
      expect(screen.queryByText(CONFIRM_CONTENT)).not.toBeInTheDocument();
    });

    expect(handleClose).toHaveBeenCalled();
  });
});
