import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { DefaultTheme, UIProvider } from 'hew/Theme';
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import { HelmetProvider } from 'react-helmet-async';
import { BrowserRouter } from 'react-router-dom';

import { ThemeProvider } from 'components/ThemeProvider';
import { SettingsProvider } from 'hooks/useSettingsProvider';

import GroupManagement from './GroupManagement';

const GROUP_NAME = 'test_group_name';
const GROUP_MEMBER_COUNT = 5;

vi.mock('hooks/usePermissions', () => {
  const usePermissions = vi.fn(() => {
    return {
      canModifyGroups: true,
      canViewGroups: true,
    };
  });
  return {
    default: usePermissions,
  };
});

vi.mock('services/api', () => ({
  getGroupRoles: () => Promise.resolve([]),
  getGroups: () =>
    Promise.resolve({
      groups: [
        {
          group: {
            groupId: 5,
            name: GROUP_NAME,
          },
          numMembers: GROUP_MEMBER_COUNT,
        },
      ],
      pagination: {
        endIndex: 10,
        limit: 10,
        offset: 0,
        startIndex: 0,
        total: 10,
      },
    }),
}));

const setup = () =>
  render(
    <UIProvider theme={DefaultTheme.Light}>
      <ThemeProvider>
        <DndProvider backend={HTML5Backend}>
          <SettingsProvider>
            <HelmetProvider>
              <BrowserRouter>
                <GroupManagement onGroupsUpdate={() => {}} />;
              </BrowserRouter>
            </HelmetProvider>
          </SettingsProvider>
        </DndProvider>
      </ThemeProvider>
    </UIProvider>,
  );

const user = userEvent.setup();

describe('GroupManagement', () => {
  it('should render with correct group data', async () => {
    setup();

    expect(await screen.findByTestId('Group')).toBeInTheDocument();
    expect(await screen.findByTestId('Members')).toBeInTheDocument();

    expect(await screen.findByText(GROUP_NAME)).toBeInTheDocument();
    expect(await screen.findByText(GROUP_MEMBER_COUNT)).toBeInTheDocument();
  });

  it('should open action menu for a group', async () => {
    setup();
    expect(await screen.findByTestId('Group')).toBeInTheDocument();
    expect(await screen.findByTestId('Members')).toBeInTheDocument();

    expect(await screen.findByText(GROUP_NAME)).toBeInTheDocument();
    expect(await screen.findByText(GROUP_MEMBER_COUNT)).toBeInTheDocument();

    const buttons = await screen.queryAllByRole('button');
    const menuButton = buttons[2];
    await user.click(menuButton);

    expect(screen.getByText('Add Members to Group')).toBeInTheDocument();
    expect(screen.getByText('Edit Group')).toBeInTheDocument();
    expect(screen.getByText('Delete Group')).toBeInTheDocument();
  });
});
