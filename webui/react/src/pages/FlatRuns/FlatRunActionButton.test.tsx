import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import dayjs from 'dayjs';
import UIProvider, { DefaultTheme } from 'hew/Theme';

import FlatRunActionButton from 'pages/FlatRuns/FlatRunActionButton';
import { FlatRun, RunState } from 'types';

const setup = (selectedFlatRuns: ReadonlyArray<Readonly<FlatRun>>) => {
  const user = userEvent.setup();

  render(
    <UIProvider theme={DefaultTheme.Light}>
      <FlatRunActionButton
        isMobile={false}
        projectId={1}
        selectedRuns={selectedFlatRuns}
        workspaceId={1}
        onActionComplete={vi.fn()}
      />
    </UIProvider>,
  );

  return {
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
  });
});
