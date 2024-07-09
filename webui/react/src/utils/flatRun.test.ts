import dayjs from 'dayjs';

import {
  deletableRunStates,
  killableRunStates,
  pausableRunStates,
  runStateList,
  terminalRunStates,
} from 'constants/states';
import { FlatRun, FlatRunAction, RunState } from 'types';
import { canActionFlatRun } from 'utils/flatRun';

describe('Flat Run Utilities', () => {
  const BASE_FLAT_RUN: Readonly<FlatRun> = {
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
  };

  describe('canActionFlatRun function', () => {
    const terminatedRunStates: Set<RunState> = new Set(
      Object.values(RunState).filter((v) => terminalRunStates.has(v)),
    );

    describe('Archive Action', () => {
      it.each(Array.from(terminatedRunStates))(
        'should be archivable (%s)',
        (terminatedRunState) => {
          const flatRun: FlatRun = {
            ...BASE_FLAT_RUN,
            state: terminatedRunState,
          };
          expect(canActionFlatRun(FlatRunAction.Archive, flatRun)).toBeTruthy();
        },
      );

      it.each(Array.from(terminatedRunStates))(
        'should not be archivable (%s)',
        (terminatedRunState) => {
          // just parentArchived is true
          const parentArchivedFlatRun: FlatRun = {
            ...BASE_FLAT_RUN,
            parentArchived: true,
            state: terminatedRunState,
          };
          expect(canActionFlatRun(FlatRunAction.Archive, parentArchivedFlatRun)).toBeFalsy();

          // just archived is true
          const archivedFlatRun: FlatRun = {
            ...BASE_FLAT_RUN,
            archived: true,
            state: terminatedRunState,
          };
          expect(canActionFlatRun(FlatRunAction.Archive, archivedFlatRun)).toBeFalsy();

          // parentArchived and archived are true
          const bothArchivedFlatRun: FlatRun = {
            ...BASE_FLAT_RUN,
            archived: true,
            parentArchived: true,
            state: terminatedRunState,
          };
          expect(canActionFlatRun(FlatRunAction.Archive, bothArchivedFlatRun)).toBeFalsy();
        },
      );
    });

    describe('Unarchive Action', () => {
      it.each(Array.from(terminatedRunStates))(
        'should be unarchivable (%s)',
        (terminatedRunState) => {
          const archivedFlatRun: FlatRun = {
            ...BASE_FLAT_RUN,
            archived: true,
            parentArchived: false,
            state: terminatedRunState,
          };
          expect(canActionFlatRun(FlatRunAction.Unarchive, archivedFlatRun)).toBeTruthy();

          const bothArchivedFlatRun: FlatRun = {
            ...BASE_FLAT_RUN,
            archived: true,
            parentArchived: true,
            state: terminatedRunState,
          };
          expect(canActionFlatRun(FlatRunAction.Unarchive, bothArchivedFlatRun)).toBeTruthy();
        },
      );

      it.each(Array.from(terminatedRunStates))(
        'should not be unarchivable with Terminated Run State (%s)',
        (terminatedRunState) => {
          const flatRun: FlatRun = {
            ...BASE_FLAT_RUN,
            archived: false,
            state: terminatedRunState,
          };
          expect(canActionFlatRun(FlatRunAction.Unarchive, flatRun)).toBeFalsy();

          const parentArchivedFlatRun: FlatRun = {
            ...BASE_FLAT_RUN,
            parentArchived: true,
            state: terminatedRunState,
          };
          expect(canActionFlatRun(FlatRunAction.Unarchive, parentArchivedFlatRun)).toBeFalsy();
        },
      );

      it.each(Array.from(Object.values(RunState).filter((v) => !terminatedRunStates.has(v))))(
        'should not be unarchivable with non Terminated Run States (%s)',
        (nonTerminatedRunState) => {
          const flatRun: FlatRun = {
            ...BASE_FLAT_RUN,
            state: nonTerminatedRunState,
          };
          expect(canActionFlatRun(FlatRunAction.Unarchive, flatRun)).toBeFalsy();

          const archivedFlatRun: FlatRun = {
            ...BASE_FLAT_RUN,
            archived: true,
            state: nonTerminatedRunState,
          };
          expect(canActionFlatRun(FlatRunAction.Unarchive, archivedFlatRun)).toBeFalsy();
        },
      );
    });

    describe('Delete Action', () => {
      it.each(runStateList)('should be deletable (%s)', (runState) => {
        const flatRun: FlatRun = {
          ...BASE_FLAT_RUN,
          state: runState,
        };
        expect(canActionFlatRun(FlatRunAction.Delete, flatRun)).toBeTruthy();
      });

      it.each(Object.values(RunState).filter((state) => !deletableRunStates.has(state)))(
        'should not be deletable',
        (nonDeletableRunState) => {
          const flatRun: FlatRun = {
            ...BASE_FLAT_RUN,
            state: nonDeletableRunState,
          };
          expect(canActionFlatRun(FlatRunAction.Delete, flatRun)).toBeFalsy();
        },
      );
    });

    describe('Kill Action', () => {
      const killRunStates: Set<RunState> = new Set(
        Object.values(RunState).filter((v) => killableRunStates.includes(v)),
      );

      it.each(Array.from(killRunStates))('should be killable (%s)', (killRunState) => {
        const flatRun: FlatRun = {
          ...BASE_FLAT_RUN,
          state: killRunState,
        };
        expect(canActionFlatRun(FlatRunAction.Kill, flatRun)).toBeTruthy();
      });

      it.each(Object.values(RunState).filter((v) => !killRunStates.has(v)))(
        'should not be killable (%s)',
        (nonKillRunState) => {
          const flatRun: FlatRun = {
            ...BASE_FLAT_RUN,
            state: nonKillRunState,
          };
          expect(canActionFlatRun(FlatRunAction.Kill, flatRun)).toBeFalsy();
        },
      );
    });

    describe('Move Action', () => {
      it.each(Object.values(RunState))('should be movable (%s)', (runState) => {
        const flatRun: FlatRun = {
          ...BASE_FLAT_RUN,
          state: runState,
        };
        expect(canActionFlatRun(FlatRunAction.Move, flatRun)).toBeTruthy();
      });

      it.each(Object.values(RunState))('should not be movable (%s)', (runState) => {
        // just parentArchived is true
        const parentArchivedFlatRun: FlatRun = {
          ...BASE_FLAT_RUN,
          parentArchived: true,
          state: runState,
        };
        expect(canActionFlatRun(FlatRunAction.Move, parentArchivedFlatRun)).toBeFalsy();

        // just archived is true
        const archivedFlatRun: FlatRun = {
          ...BASE_FLAT_RUN,
          archived: true,
          state: runState,
        };
        expect(canActionFlatRun(FlatRunAction.Move, archivedFlatRun)).toBeFalsy();

        // both archived and parentArchived are true
        const bothArchivedFlatRun: FlatRun = {
          ...BASE_FLAT_RUN,
          archived: true,
          parentArchived: true,
          state: runState,
        };
        expect(canActionFlatRun(FlatRunAction.Move, bothArchivedFlatRun)).toBeFalsy();
      });
    });

    describe('Pause Action', () => {
      it.each(Object.values(RunState).filter((v) => pausableRunStates.has(v)))(
        'should be pausable (%s)',
        (runState) => {
          const flatRun: FlatRun = {
            ...BASE_FLAT_RUN,
            state: runState,
          };
          expect(canActionFlatRun(FlatRunAction.Pause, flatRun)).toBeTruthy();

          const flatRunWithSingletrial: FlatRun = {
            ...flatRun,
            experiment: {
              description: '',
              id: 1,
              isMultitrial: false,
              name: 'name',
              progress: 1,
              resourcePool: 'default',
              searcherMetric: 'validation_loss',
              searcherType: 'adaptive_asha',
              unmanaged: false,
            },
          };
          expect(canActionFlatRun(FlatRunAction.Pause, flatRunWithSingletrial)).toBeTruthy();
        },
      );

      it.each(Object.values(RunState).filter((v) => pausableRunStates.has(v)))(
        'should not be pausable with pausable run states (%s)',
        (runState) => {
          const flatRunWithMultitrial: FlatRun = {
            ...BASE_FLAT_RUN,
            experiment: {
              description: '',
              id: 1,
              isMultitrial: true,
              name: 'name',
              progress: 1,
              resourcePool: 'default',
              searcherMetric: 'validation_loss',
              searcherType: 'adaptive_asha',
              unmanaged: false,
            },
            state: runState,
          };
          expect(canActionFlatRun(FlatRunAction.Pause, flatRunWithMultitrial)).toBeFalsy();
        },
      );

      it.each(Object.values(RunState).filter((v) => !pausableRunStates.has(v)))(
        'should not be pausable with nonpausable run states (%s)',
        (runState) => {
          const flatRun: FlatRun = {
            ...BASE_FLAT_RUN,
            state: runState,
          };
          expect(canActionFlatRun(FlatRunAction.Pause, flatRun)).toBeFalsy();

          const flatRunWithSingletrial: FlatRun = {
            ...flatRun,
            experiment: {
              description: '',
              id: 1,
              isMultitrial: false,
              name: 'name',
              progress: 1,
              resourcePool: 'default',
              searcherMetric: 'validation_loss',
              searcherType: 'single',
              unmanaged: false,
            },
          };
          expect(canActionFlatRun(FlatRunAction.Pause, flatRunWithSingletrial)).toBeFalsy();

          const flatRunWithMultitrial: FlatRun = {
            ...flatRun,
            experiment: {
              description: '',
              id: 1,
              isMultitrial: true,
              name: 'name',
              progress: 1,
              resourcePool: 'default',
              searcherMetric: 'validation_loss',
              searcherType: 'adaptive_asha',
              unmanaged: false,
            },
          };
          expect(canActionFlatRun(FlatRunAction.Pause, flatRunWithMultitrial)).toBeFalsy();
        },
      );
    });
  });
});
