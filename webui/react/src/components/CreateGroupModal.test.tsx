import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Button from 'hew/Button';
import { useModal } from 'hew/Modal';
import { DefaultTheme, UIProvider } from 'hew/Theme';
import React from 'react';

import { ThemeProvider } from 'components/ThemeProvider';
import { createGroup as mockCreateGroup } from 'services/api';
import { V1GroupSearchResult } from 'services/api-ts-sdk';
import { GetGroupParams } from 'services/types';
import { DetailedUser } from 'types';

import CreateGroupModalComponent, {
  API_SUCCESS_MESSAGE_CREATE,
  GROUP_NAME_LABEL,
  MODAL_HEADER_LABEL_CREATE,
  MODAL_HEADER_LABEL_EDIT,
} from './CreateGroupModal';

const OPEN_MODAL_TEXT = 'Open Modal';
const GROUPNAME = 'test_groupname1';

const user = userEvent.setup();

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

vi.mock('services/api', () => ({
  createGroup: vi.fn(),
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
  const CreateGroupModal = useModal(CreateGroupModalComponent);

  return (
    <div>
      <Button onClick={CreateGroupModal.open}>{OPEN_MODAL_TEXT}</Button>
      <CreateGroupModal.Component group={group} />
    </div>
  );
};

const setup = async (group?: V1GroupSearchResult) => {
  const view = render(
    <UIProvider theme={DefaultTheme.Light}>
      <ThemeProvider>
        <Container group={group} />
      </ThemeProvider>
    </UIProvider>,
  );

  await user.click(await view.findByText(OPEN_MODAL_TEXT));
  await view.getAllByText(group ? MODAL_HEADER_LABEL_EDIT : MODAL_HEADER_LABEL_CREATE);

  return view;
};

describe('Create Group Modal', () => {
  it('should open modal with correct values', async () => {
    await setup();

    expect(screen.getByLabelText(GROUP_NAME_LABEL)).toBeInTheDocument();
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
  });
});
