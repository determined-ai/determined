import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';

import Button from 'components/kit/Button';
import { V1GroupSearchResult } from 'services/api-ts-sdk';
import { CreateGroupsParams, GetGroupParams } from 'services/types';
import { StoreProvider as UIProvider } from 'shared/contexts/stores/UI';
import { DetailedUser } from 'types';

import useModalCreateGroup, {
  API_SUCCESS_MESSAGE_CREATE,
  GROUP_NAME_LABEL,
  MODAL_HEADER_LABEL_CREATE,
  MODAL_HEADER_LABEL_EDIT,
  USER_ADD_LABEL,
  USER_LABEL,
} from './useModalCreateGroup';

const OPEN_MODAL_TEXT = 'Open Modal';
const GROUPNAME = 'test_groupname1';

const user = userEvent.setup();

const mockCreateGroup = jest.fn();

const users: Array<DetailedUser> = [
  {
    id: 1,
    isActive: true,
    isAdmin: false,
    username: 'test_username0',
  },
  {
    id: 2,
    isActive: true,
    isAdmin: false,
    username: 'test_username1',
  },
];

jest.mock('services/api', () => ({
  createGroup: (params: CreateGroupsParams) => {
    return mockCreateGroup(params);
  },
  getGroup: (params: GetGroupParams) => {
    return Promise.resolve({
      group: {
        groupId: params.groupId,
        name: GROUPNAME,
        users: users,
      },
    });
  },
}));

interface Props {
  group?: V1GroupSearchResult;
}

const Container: React.FC<Props> = ({ group }) => {
  const { contextHolder, modalOpen } = useModalCreateGroup({ group: group, users: users });

  return (
    <div>
      <Button onClick={() => modalOpen()}>{OPEN_MODAL_TEXT}</Button>
      {contextHolder}
    </div>
  );
};

const setup = async (group?: V1GroupSearchResult) => {
  const view = render(
    <UIProvider>
      <Container group={group} />
    </UIProvider>,
  );

  await user.click(await view.findByText(OPEN_MODAL_TEXT));
  await view.findByRole('heading', {
    name: group ? MODAL_HEADER_LABEL_EDIT : MODAL_HEADER_LABEL_CREATE,
  });

  return view;
};

describe('useModalCreateGroup', () => {
  it('should open modal with correct values', async () => {
    await setup();

    expect(screen.getByLabelText(GROUP_NAME_LABEL)).toBeInTheDocument();
    expect(screen.getByLabelText(USER_LABEL)).toBeInTheDocument();
  });

  it('should close the modal via upper right close button', async () => {
    await setup();

    await user.click(await screen.findByLabelText('Close'));

    // Check for the modal to be dismissed.
    await waitFor(() => {
      expect(
        screen.queryByRole('heading', { name: MODAL_HEADER_LABEL_CREATE }),
      ).not.toBeInTheDocument();
    });
  });

  it('should close the modal via cancel button', async () => {
    await setup();

    await user.click(await screen.findByText('Cancel'));

    // Check for the modal to be dismissed.
    await waitFor(() => {
      expect(
        screen.queryByRole('heading', { name: MODAL_HEADER_LABEL_CREATE }),
      ).not.toBeInTheDocument();
    });
  });

  it('should submit a valid create group request', async () => {
    await setup();

    await user.type(screen.getByLabelText(GROUP_NAME_LABEL), GROUPNAME);
    await user.click(screen.getByRole('button', { name: MODAL_HEADER_LABEL_CREATE }));

    // Check for successful toast message.
    await waitFor(() => {
      expect(
        screen.getByText(API_SUCCESS_MESSAGE_CREATE, { collapseWhitespace: false }),
      ).toBeInTheDocument();
    });

    // Check for the modal to be dismissed.
    await waitFor(() => {
      expect(
        screen.queryByRole('heading', { name: MODAL_HEADER_LABEL_CREATE }),
      ).not.toBeInTheDocument();
    });

    // Check that the API method was called with the correct parameters.
    expect(mockCreateGroup).toHaveBeenCalledWith({ name: GROUPNAME });
  });

  it('should open edit modal with correct values', async () => {
    const group = {
      group: {
        groupId: 1,
        name: GROUPNAME,
      },
      numMembers: 0,
    };
    await setup(group);

    expect(screen.getByLabelText(GROUP_NAME_LABEL)).toBeInTheDocument();
    expect(screen.getByLabelText(USER_ADD_LABEL)).toBeInTheDocument();
  });
});
