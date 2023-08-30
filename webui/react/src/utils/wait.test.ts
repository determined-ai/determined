import { MaybeMocked } from '@vitest/spy';

import { CommandState, CommandTask, CommandType, RecordKey } from 'types';
import { generateUUID } from 'utils/string';
import * as utils from 'utils/wait';

const UUID = {
  [CommandType.JupyterLab]: generateUUID(),
  [CommandType.Shell]: generateUUID(),
  [CommandType.TensorBoard]: generateUUID(),
};
const UUID_REGEX = '[a-z\\d-]+';
const SHARED_TASK: Partial<CommandTask> = {
  misc: undefined,
  url: undefined,
};
const COMMAND_TASK: Record<RecordKey, CommandTask> = {
  [CommandType.JupyterLab]: {
    ...SHARED_TASK,
    displayName: 'lee sung hoon',
    id: UUID[CommandType.JupyterLab],
    name: 'JupyterLab (directly-sharp-penguin)',
    resourcePool: 'compute-pool',
    serviceAddress: `/proxy/${UUID[CommandType.JupyterLab]}`,
    startTime: '2022-08-01T00:12:24Z',
    state: CommandState.Queued,
    type: CommandType.JupyterLab,
    userId: 34,
    workspaceId: 0,
  },
  [CommandType.Shell]: {
    ...SHARED_TASK,
    displayName: 'lee sung hoon',
    id: UUID[CommandType.Shell],
    name: 'Shell (jolly-well-pheasant)',
    resourcePool: 'compute-pool',
    serviceAddress: `/proxy/${UUID[CommandType.Shell]}`,
    startTime: '2022-07-29T24:00:12Z',
    state: CommandState.Terminated,
    type: CommandType.Shell,
    userId: 34,
    workspaceId: 0,
  },
  [CommandType.TensorBoard]: {
    ...SHARED_TASK,
    displayName: 'racer kim',
    id: UUID[CommandType.TensorBoard],
    name: 'TensorBoard (ambiguously-happy-ear)',
    resourcePool: 'aux-pool',
    serviceAddress: `/proxy/${UUID[CommandType.TensorBoard]}`,
    startTime: '2022-08-02T12:24:00Z',
    state: CommandState.Running,
    type: CommandType.TensorBoard,
    userId: 16,
    workspaceId: 0,
  },
};

describe('Wait Page Utilities', () => {
  describe('openCommand', () => {
    let windowOpen: MaybeMocked<typeof global.open>;

    beforeEach(() => {
      vi.spyOn(global, 'open');
      windowOpen = vi.mocked(global.open);
      windowOpen.mockReset();
    });

    afterEach(() => {
      // Restore `global.open` to original function.
      vi.mocked(global.open).mockRestore();
    });

    it('should open window for JupyterLab task', () => {
      expect(windowOpen).not.toHaveBeenCalled();
      utils.openCommand(COMMAND_TASK[CommandType.JupyterLab]);
      // TODO: Expand this to use `toHaveBeenCalledWith`.
      expect(windowOpen).toHaveBeenCalled();
    });

    it('should open window for TensorBoard task', () => {
      expect(windowOpen).not.toHaveBeenCalled();
      utils.openCommand(COMMAND_TASK[CommandType.TensorBoard]);
      // TODO: Expand this to use `toHaveBeenCalledWith`.
      expect(windowOpen).toHaveBeenCalled();
    });

    it('should throw error for tasks that are not JupyterLabs or TensorBoards', () => {
      expect(() => {
        utils.openCommand(COMMAND_TASK[CommandType.Shell]);
      }).toThrow(utils.CANNOT_OPEN_COMMAND_ERROR);
    });
  });

  describe('waitPageUrl', () => {
    const REGEX: Record<RecordKey, RegExp> = {
      [CommandType.JupyterLab]: new RegExp(`wait/${CommandType.JupyterLab}/${UUID_REGEX}`, 'i'),
      [CommandType.TensorBoard]: new RegExp(`wait/${CommandType.TensorBoard}/${UUID_REGEX}`, 'i'),
    };

    it('should convert task to wait page url', () => {
      expect(utils.waitPageUrl(COMMAND_TASK[CommandType.JupyterLab])).toMatch(
        REGEX[CommandType.JupyterLab],
      );

      expect(utils.waitPageUrl(COMMAND_TASK[CommandType.TensorBoard])).toMatch(
        REGEX[CommandType.TensorBoard],
      );
    });

    it('should throw error for tasks that are not JupyterLabs or TensorBoards', () => {
      expect(() => {
        utils.waitPageUrl(COMMAND_TASK[CommandType.Shell]);
      }).toThrow(utils.CANNOT_OPEN_COMMAND_ERROR);
    });
  });
});
