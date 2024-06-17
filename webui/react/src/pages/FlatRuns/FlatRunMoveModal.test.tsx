import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import dayjs from 'dayjs';
import Button from 'hew/Button';
import { useModal } from 'hew/Modal';
import UIProvider, { DefaultTheme } from 'hew/Theme';
import { useMemo } from 'react';

import FlatRunMoveModalComponent from 'pages/FlatRuns/FlatRunMoveModal';
import { V1MoveRunsRequest } from 'services/api-ts-sdk';
import { BulkActionResult, FlatRun, RunState } from 'types';

const OPEN_MODAL_TEXT = 'Open Modal';

vi.mock('services/api', () => ({
  createGroup: vi.fn(),
  getWorkspaceProjects: vi.fn(() =>
    Promise.resolve({ projects: [{ id: 1, name: 'project_1', workspaceId: 1 }] }),
  ),
  getWorkspaces: vi.fn(() => Promise.resolve({ workspaces: [] })),
  moveRuns: vi.fn((params: V1MoveRunsRequest) => {
    return Promise.resolve({
      failed: [],
      successful: params.runIds,
    });
  }),
  searchRuns: vi.fn(() => Promise.resolve({ pagination: { total: 0 } })),
}));

const Container = ({
  onSubmit,
}: {
  onSubmit?: (results: BulkActionResult, destinationProjectId: number) => void;
}): JSX.Element => {
  const BASE_FLAT_RUNS: FlatRun[] = useMemo(() => {
    return [
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
        workspaceId: 1,
        workspaceName: 'test',
      },
    ];
  }, []);

  const flatRunMoveModal = useModal(FlatRunMoveModalComponent);

  return (
    <div>
      <Button onClick={flatRunMoveModal.open}>{OPEN_MODAL_TEXT}</Button>
      <flatRunMoveModal.Component
        flatRuns={BASE_FLAT_RUNS}
        sourceProjectId={1}
        onSubmit={onSubmit}
      />
    </div>
  );
};

const setup = () => {
  const onSubmit = vi.fn();
  const user = userEvent.setup();

  render(
    <UIProvider theme={DefaultTheme.Light}>
      <Container onSubmit={onSubmit} />
    </UIProvider>,
  );

  return {
    handlers: { onSubmit },
    user,
  };
};

describe('FlatRunMoveModalComponent', () => {
  it('should open modal', async () => {
    const { user } = setup();

    await user.click(screen.getByRole('button', { name: OPEN_MODAL_TEXT }));
    expect((await screen.findAllByText('Move Run')).length).toBe(2);
    expect(await screen.findByText('Workspace')).toBeInTheDocument();
    expect(await screen.findByRole('button', { name: 'Cancel' })).toBeInTheDocument();
    expect(await screen.findByRole('button', { name: 'Move Run' })).toBeInTheDocument();
  });

  it('should submit modal', async () => {
    const { user, handlers } = setup();

    await user.click(screen.getByRole('button', { name: OPEN_MODAL_TEXT }));
    expect((await screen.findAllByText('Move Run')).length).toBe(2);
    expect(await screen.findByText('Workspace')).toBeInTheDocument();
    expect(await screen.findByRole('button', { name: 'Move Run' })).not.toBeDisabled();
    await user.click(await screen.findByRole('button', { name: 'Move Run' }));
    expect(handlers.onSubmit).toBeCalled();
  });
});
