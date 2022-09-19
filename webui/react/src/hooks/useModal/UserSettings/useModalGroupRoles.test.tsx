import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Button } from 'antd';
import React from 'react';

import StoreProvider from 'contexts/Store';

import useModalGroupRoles from './useModalGroupRoles';

const OPEN_MODAL_TEXT = 'Open';
const GROUP_NAME = 'Test Group';

jest.mock('services/api', () => ({
  getGroupRoles: () => {
    return Promise.resolve([]);
  },
}));

const user = userEvent.setup();

const Container: React.FC = () => {
  const group = {
    group: {
      groupId: -1,
      name: GROUP_NAME,
    },
  };
  const roles = [ {
    id: -2,
    name: 'Test Role',
  } ];

  const { contextHolder, modalOpen } = useModalGroupRoles({ group, roles });

  return (
    <div>
      <Button onClick={() => modalOpen()}>Open</Button>
      {contextHolder}
    </div>
  );
};

const setup = async () => {
  const view = render(
    <StoreProvider>
      <Container />
    </StoreProvider>,
  );

  await user.click(await view.findByText(OPEN_MODAL_TEXT));
  await view.findByRole('heading', { name: 'Update Roles' });

  return view;
};

describe('useModalGropRoles', () => {
  it('should open modal with group name and a section for roles', async () => {
    await setup();

    expect(screen.getByText(`Add Roles to: ${GROUP_NAME}`)).toBeInTheDocument();
    expect(screen.getByLabelText('Roles')).toBeInTheDocument();
  });

  it('should close the modal via upper right close button', async () => {
    await setup();

    await user.click(await screen.findByLabelText('Close'));

    // Check for the modal to be dismissed.
    await waitFor(() => {
      expect(
        screen.queryByRole('heading', { name: 'Update Roles' }),
      ).not.toBeInTheDocument();
    });
  });

  it('should close the modal via cancel button', async () => {
    await setup();

    await user.click(await screen.findByText('Cancel'));

    // Check for the modal to be dismissed.
    await waitFor(() => {
      expect(
        screen.queryByRole('heading', { name: 'Update Roles' }),
      ).not.toBeInTheDocument();
    });
  });
});
