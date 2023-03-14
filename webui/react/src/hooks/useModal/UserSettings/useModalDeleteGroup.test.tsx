import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';

import Button from 'components/kit/Button';
import { V1GroupSearchResult } from 'services/api-ts-sdk';
import { DeleteGroupParams } from 'services/types';
import { StoreProvider as UIProvider } from 'shared/contexts/stores/UI';

import useModalDeleteGroup, { API_SUCCESS_MESSAGE, MODAL_HEADER } from './useModalDeleteGroup';

const mockDeleteGroup = jest.fn();

jest.mock('services/api', () => ({
  deleteGroup: (params: DeleteGroupParams) => {
    return mockDeleteGroup(params);
  },
}));

const OPEN_MODAL_TEXT = 'Open Modal';
const GROUPNAME = 'test_groupname1';

const user = userEvent.setup();

interface Props {
  group: V1GroupSearchResult;
}

const Container: React.FC<Props> = ({ group }) => {
  const { contextHolder, modalOpen } = useModalDeleteGroup({ group: group });

  return (
    <div>
      <Button onClick={() => modalOpen()}>{OPEN_MODAL_TEXT}</Button>
      {contextHolder}
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
  await view.findByRole('heading', { name: MODAL_HEADER });

  return view;
};

describe('useModalCreateGroup', () => {
  it('should open modal with correct values', async () => {
    await setup();

    expect(
      screen.getByText(`Are you sure you want to delete group ${GROUPNAME} (ID: 1).`),
    ).toBeInTheDocument();
  });

  it('should close the modal via upper right close button', async () => {
    await setup();

    await user.click(await screen.findByLabelText('Close'));

    // Check for the modal to be dismissed.
    await waitFor(() => {
      expect(screen.queryByRole('heading', { name: MODAL_HEADER })).not.toBeInTheDocument();
    });
  });

  it('should close the modal via cancel button', async () => {
    await setup();

    await user.click(await screen.findByText('Cancel'));

    // Check for the modal to be dismissed.
    await waitFor(() => {
      expect(screen.queryByRole('heading', { name: MODAL_HEADER })).not.toBeInTheDocument();
    });
  });

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
