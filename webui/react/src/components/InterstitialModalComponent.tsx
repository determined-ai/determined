import { Modal, ModalCloseReason } from 'hew/Modal';
import Spinner from 'hew/Spinner';
import { Loadable } from 'hew/utils/loadable';
import { useCallback, useEffect } from 'react';

interface Props<T> {
  close: (reason: ModalCloseReason) => void;
  loadableData: Loadable<T>;
  then: () => void;
}

function InterstitialModalComponent<T>({ close, loadableData, then }: Props<T>): JSX.Element {
  useEffect(() => {
    if (loadableData.isLoaded) {
      close('ok');
      then();
    } else if (loadableData.isFailed) {
      close('failed');
    }
  }, [close, loadableData.isFailed, loadableData.isLoaded, then]);

  const onClose = useCallback(() => {
    close('close');
  }, [close]);

  return (
    <Modal footer={<></>} size="small" title="Loading" onClose={onClose}>
      <Spinner center />
    </Modal>
  );
}

export default InterstitialModalComponent;
