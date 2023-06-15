import { Action } from 'components/Table/TableBulkActions';
import { openOrCreateTensorBoard } from 'services/api';
import { ValueOf } from 'types';
import { ErrorLevel, ErrorType } from 'utils/error';
import handleError from 'utils/error';
import { openCommandResponse } from 'utils/wait';

import { TrialsSelectionOrCollection } from '../Collections/collections';

export const TrialAction = {
  AddTags: 'Add Tags',
  OpenTensorBoard: 'View in TensorBoard',
  TagAndCollect: 'Tag and Collect',
} as const;

export type TrialAction = ValueOf<typeof TrialAction>;

type trials = { trials: TrialsSelectionOrCollection; workspaceId: number };

export type TrialsActionHandler = (t: trials) => Promise<void> | void;

export const openTensorBoard = async ({ trials, workspaceId }: trials): Promise<void> => {
  if ('trialIds' in trials) {
    const result = await openOrCreateTensorBoard({
      trialIds: trials.trialIds,
      workspaceId: workspaceId,
    });
    if (result) openCommandResponse(result);
  }
};

export const trialActionDefs: Record<TrialAction, Action<TrialAction>> = {
  [TrialAction.AddTags]: {
    bulk: true,
    label: TrialAction.AddTags,
    value: TrialAction.AddTags,
  },
  [TrialAction.TagAndCollect]: {
    bulk: false,
    label: TrialAction.TagAndCollect,
    value: TrialAction.TagAndCollect,
  },
  [TrialAction.OpenTensorBoard]: {
    bulk: false,
    label: TrialAction.OpenTensorBoard,
    value: TrialAction.OpenTensorBoard,
  },
};

export const dispatchTrialAction = async (
  action: TrialAction,
  trials: TrialsSelectionOrCollection,
  handler: TrialsActionHandler,
  workspaceId: number,
): Promise<void> => {
  try {
    await handler({ trials, workspaceId });
  } catch (e) {
    const publicSubject =
      action === TrialAction.OpenTensorBoard
        ? 'Unable to View TensorBoard for Selected Trials'
        : `Unable to ${action} Selected Trials`;
    handleError(e, {
      level: ErrorLevel.Error,
      publicMessage: 'Please try again later.',
      publicSubject,
      silent: false,
      type: ErrorType.Server,
    });
  }
};
