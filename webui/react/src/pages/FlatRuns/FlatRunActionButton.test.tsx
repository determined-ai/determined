import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import dayjs from 'dayjs';
import UIProvider, { DefaultTheme } from 'hew/Theme';

import FlatRunActionButton from 'pages/FlatRuns/FlatRunActionButton';
import { FlatRun, RunState } from 'types';

vi.mock('services/api', async (importOriginal) => ({
  __esModule: true,
  ...(await importOriginal<typeof import('services/api')>()),
  getWorkspaceProjects: vi.fn(() =>
    Promise.resolve({ projects: [{ id: 1, name: 'project_1', workspaceId: 1 }] }),
  ),
  killRuns: vi.fn((params: { projectId: number; runIds: number[] }) => {
    return Promise.resolve({ failed: [], successful: params.runIds });
  }),
}));

const setup = (selectedFlatRuns: ReadonlyArray<Readonly<FlatRun>>) => {
  const user = userEvent.setup();
  const onActionSuccess = vi.fn();
  const onActionComplete = vi.fn();

  render(
    <UIProvider theme={DefaultTheme.Light}>
      <FlatRunActionButton
        isMobile={false}
        projectId={1}
        selectedRuns={selectedFlatRuns}
        selection={{ selections: selectedFlatRuns.map((run) => run.id), type: 'ONLY_IN' }}
        tableFilterString=""
        workspaceId={1}
        onActionComplete={onActionComplete}
        onActionSuccess={onActionSuccess}
      />
    </UIProvider>,
  );

  return {
    handler: { onActionSuccess },
    user,
  };
};

describe('canActionFlatRun function', () => {
  describe('Flat Run Action Button Visibility', () => {
    const flatRuns: ReadonlyArray<Readonly<FlatRun>> = [
      {
        archived: false,
        checkpointCount: 0,
        checkpointSize: 0,
        id: 1,
        parentArchived: false,
        projectId: 1,
        projectName: 'test',
        startTime: dayjs('2024-05-24T23:03:45.415603Z').toDate(),
        state: RunState.Active,
        workspaceId: 10,
        workspaceName: 'test',
      },
    ];

    it('should not appear without selected flat runs', () => {
      setup([]);
      expect(screen.queryByText('Actions')).not.toBeInTheDocument();
    });

    it('should appear with selected flat runs', async () => {
      setup(flatRuns);
      expect(await screen.findByText('Actions')).toBeInTheDocument();
    });

    it('should show action list', async () => {
      const { user } = setup(flatRuns);

      const actionButton = await screen.findByText('Actions');
      await user.click(actionButton);
      expect(await screen.findByText('Move')).toBeInTheDocument();
      expect(await screen.findByText('Archive')).toBeInTheDocument();
      expect(await screen.findByText('Unarchive')).toBeInTheDocument();
      expect(await screen.findByText('Delete')).toBeInTheDocument();
      expect(await screen.findByText('Kill')).toBeInTheDocument();
    });

    it('should kill runs', async () => {
      const { user, handler } = setup(flatRuns);
      const actionButton = await screen.findByText('Actions');
      await user.click(actionButton);
      await user.click(await screen.findByText('Kill'));
      expect(await screen.findByText('Confirm Batch Kill')).toBeInTheDocument();
      await user.click(await screen.findByRole('button', { name: 'Kill' }));
      expect(handler.onActionSuccess).toBeCalled();
    });
  });
});
