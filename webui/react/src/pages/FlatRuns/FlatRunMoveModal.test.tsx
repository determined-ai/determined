import { render, screen } from '@testing-library/react';
import userEvent, { UserEvent } from '@testing-library/user-event';
import dayjs from 'dayjs';
import Button from 'hew/Button';
import { useModal } from 'hew/Modal';
import UIProvider, { DefaultTheme } from 'hew/Theme';
import { useMemo } from 'react';

import {
  Conjunction,
  FilterFormSetWithoutId,
  FormKind,
} from 'components/FilterForm/components/type';
import { V1MoveRunsRequest } from 'services/api-ts-sdk';
import { FlatRun, RunState } from 'types';

import FlatRunMoveModalComponent from './FlatRunMoveModal';

const OPEN_MODAL_TEXT = 'Open Modal';

vi.mock('services/api', () => ({
  createGroup: vi.fn(),
  getWorkspaceProjects: vi.fn(() =>
    Promise.resolve({ projects: [{ id: 1, name: 'project_1', workspaceId: 1 }] }),
  ),
  moveRuns: (params: V1MoveRunsRequest) => {
    return Promise.resolve({
      failed: [],
      successful: params.runIds,
    });
  },
}));

const Container = (): JSX.Element => {
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
        workspaceId: 10,
        workspaceName: 'test',
      },
    ];
  }, []);

  const filterFormSetWithoutId: FilterFormSetWithoutId = useMemo(() => {
    return {
      filterGroup: { children: [], conjunction: Conjunction.Or, kind: FormKind.Group },
      showArchived: false,
    };
  }, []);

  const flatRunMoveModal = useModal(FlatRunMoveModalComponent);

  return (
    <div>
      <Button onClick={flatRunMoveModal.open}>{OPEN_MODAL_TEXT}</Button>
      <flatRunMoveModal.Component
        filterFormSetWithoutId={filterFormSetWithoutId}
        flatRuns={BASE_FLAT_RUNS}
        sourceProjectId={1}
      />
    </div>
  );
};

const setup = (): { user: UserEvent } => {
  const user = userEvent.setup();

  render(
    <UIProvider theme={DefaultTheme.Light}>
      <Container />
    </UIProvider>,
  );

  return {
    user,
  };
};

describe('FlatRunMoveModalComponent', () => {
  it('should open modal', async () => {
    const { user } = setup();

    await user.click(screen.getByRole('button', { name: OPEN_MODAL_TEXT }));
    expect((await screen.findAllByText('Move Runs')).length).toBe(2);
    expect(await screen.findByText('Workspace')).toBeInTheDocument();
    expect(await screen.findByRole('button', { name: 'Cancel' })).toBeInTheDocument();
    expect(await screen.findByRole('button', { name: 'Move Runs' })).toBeInTheDocument();
  });
});
