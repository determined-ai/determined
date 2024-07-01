import { Modal, useModal } from 'hew/Modal';
import { observable, WritableObservable } from 'micro-observables';
import { forwardRef, useImperativeHandle, useRef } from 'react';

export type CloseReason = 'ok' | 'cancel' | 'manual';

import handleError from 'utils/error';

type RunMoveWarningProps = {
  onClose: (reason: 'ok' | 'cancel') => void;
};
const RunMoveWarningCopy = ({ onClose }: RunMoveWarningProps) => (
  <Modal
    cancel
    size="small"
    submit={{ handleError, handler: () => onClose('ok'), text: 'Move runs and searches' }}
    title="Move Run Dependency Alert"
    onClose={() => onClose('cancel')}>
    {`Some of the runs you're trying to move are part of a hyperparameter search. To preserve their
    contextual relationships, the associated search(es) will be moved along with the selected runs.`}
  </Modal>
);

export type RunMoveWarningFlowRef = {
  open: () => Promise<CloseReason>;
  close: () => void;
};

export const RunMoveWarningModalComponent = forwardRef<RunMoveWarningFlowRef>((_, ref) => {
  const RunMoveWarning = useModal(RunMoveWarningCopy);
  const closeReason = useRef<WritableObservable<CloseReason | null>>(observable(null));

  const { close: internalClose, open: internalOpen } = RunMoveWarning;

  const open = async () => {
    internalOpen();
    const reason = await closeReason.current.toPromise();
    if (reason === null) {
      return Promise.reject();
    }
    return reason;
  };

  const close = (reason: CloseReason = 'manual') => {
    internalClose(reason);
    closeReason.current.set(reason);
    closeReason.current = observable(null);
  };

  useImperativeHandle(ref, () => ({ close, open }));

  return <RunMoveWarning.Component onClose={close} />;
});

export default RunMoveWarningModalComponent;
