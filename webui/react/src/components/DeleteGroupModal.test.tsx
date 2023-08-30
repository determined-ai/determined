import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';

import DeleteGroupModalComponent, {
  API_SUCCESS_MESSAGE,
  MODAL_HEADER,
} from 'components/DeleteGroupModal';
import Button from 'components/kit/Button';
import { useModal } from 'components/kit/Modal';
import { deleteGroup as mockDeleteGroup } from 'services/api';
import { V1GroupSearchResult } from 'services/api-ts-sdk';
import { StoreProvider as UIProvider } from 'stores/contexts/UI';

vi.mock('services/api', () => ({
  deleteGroup: vi.fn(),
}));

const OPEN_MODAL_TEXT = 'Open Modal';
const GROUPNAME = 'test_groupname1';

const user = userEvent.setup();

interface Props {
  group: V1GroupSearchResult;
}

const Container: React.FC<Props> = ({ group }) => {
  const DeleteGroupModal = useModal(DeleteGroupModalComponent);

  return (
    <div>
      <Button onClick={DeleteGroupModal.open}>{OPEN_MODAL_TEXT}</Button>
      <DeleteGroupModal.Component group={group} />
    </div>
  );
};

const setup = async () => {
  const group = {
    group: {
      groupId: 1,
      name: GROUPNAME,
    },
    numMembers: 0,
  };
  const view = render(
    <UIProvider>
      <Container group={group} />
    </UIProvider>,
  );

  await user.click(await view.findByText(OPEN_MODAL_TEXT));
  await view.findByText(MODAL_HEADER);

  return view;
};

describe('Delete Group Modal', () => {
  it('should submit a valid delete group request', async () => {
    await setup();

    await user.click(screen.getByRole('button', { name: 'Delete' }));

    // Check for successful toast message.
    await waitFor(() => {
      expect(
        screen.getByText(API_SUCCESS_MESSAGE, { collapseWhitespace: false }),
      ).toBeInTheDocument();
    });

    // Check for the modal to be dismissed.
    await waitFor(() => {
      expect(screen.queryByRole('heading', { name: MODAL_HEADER })).not.toBeInTheDocument();
    });

    // Check that the API method was called with the correct parameters.
    expect(mockDeleteGroup).toHaveBeenCalledWith({ groupId: 1 });
  });
});
