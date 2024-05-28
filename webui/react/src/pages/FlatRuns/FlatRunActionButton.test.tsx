import { render, screen } from '@testing-library/react';
import dayjs from 'dayjs';
import UIProvider, { DefaultTheme } from 'hew/Theme';

import FlatRunActionButton from 'pages/FlatRuns/FlatRunActionButton';
import { FlatRun, Project, RunState, WorkspaceState } from 'types';

const setup = (selectedFlatRuns: ReadonlyArray<Readonly<FlatRun>>) => {
  const project: Readonly<Project> = {
    archived: false,
    id: 1,
    immutable: false,
    name: 'proj',
    notes: [],
    state: WorkspaceState.Unspecified,
    userId: 1,
    workspaceId: 1,
  };

  render(
    <UIProvider theme={DefaultTheme.Light}>
      <FlatRunActionButton
        isMobile={false}
        project={project}
        selectedRuns={selectedFlatRuns}
        onActionComplete={vi.fn()}
      />
    </UIProvider>,
  );
};

describe('canActionFlatRun function', () => {
  describe('Flat Run Action Button Visibility', () => {
    it('should not be appeard without selected flat runs', () => {
      setup([]);
      expect(screen.queryByText('Actions')).not.toBeInTheDocument();
    });

    it('should be appeard with selected flat runs', async () => {
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

      setup(flatRuns);
      expect(await screen.findByText('Actions')).toBeInTheDocument();
    });
  });
});
