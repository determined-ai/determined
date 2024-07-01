import { Modal } from 'hew/Modal';
import Spinner from 'hew/Spinner';
import { Loadable } from 'hew/utils/loadable';
import { useCallback, useEffect } from 'react';

export type onInterstitialCloseActionType = (reason: 'ok' | 'close' | 'failed') => void;

interface Props<T> {
  onCloseAction: onInterstitialCloseActionType;
  loadableData: Loadable<T>;
}

function InterstitialModalComponent<T>({ onCloseAction, loadableData }: Props<T>): JSX.Element {
  useEffect(() => {
    if (loadableData.isLoaded) {
      onCloseAction('ok');
    } else if (loadableData.isFailed) {
      onCloseAction('failed');
    }
  }, [onCloseAction, loadableData.isLoaded, loadableData.isFailed]);

  const onClose = useCallback(() => {
    onCloseAction('close');
  }, [onCloseAction]);

  return (
    <Modal footer={<></>} size="small" title="Loading" onClose={onClose}>
      <Spinner center />
    </Modal>
  );
}

export default InterstitialModalComponent;
