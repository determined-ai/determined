import { Button } from 'antd';
import React, { useCallback, useState } from 'react';

import RegisterModelVersionModal from './RegisterModelVersionModal';

interface Props {
  checkpointUuid: string;
  onCloseAll?: () => void;
}

const RegisterModelVersionButton: React.FC<Props> = ({ checkpointUuid, onCloseAll }) => {
  const [ showRegisterVersionModal, setShowRegisterVersionModel ] = useState(false);

  const openModal = useCallback(() => {
    setShowRegisterVersionModel(true);
  }, []);

  const closeModal = useCallback(() => {
    setShowRegisterVersionModel(false);
  }, []);

  const closeAllModals = useCallback(() => {
    setShowRegisterVersionModel(false);
    onCloseAll?.();
  }, [ onCloseAll ]);

  return (
    <>
      <Button block onClick={openModal}>
      Register Model
      </Button>
      <RegisterModelVersionModal
        checkpointUuid={checkpointUuid}
        visible={showRegisterVersionModal}
        onClose={closeModal}
        onCloseAll={closeAllModals} />
    </>
  );
};

export default RegisterModelVersionButton;
