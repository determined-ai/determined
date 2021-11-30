import { Button } from 'antd';
import React, { useCallback, useState } from 'react';

import RegisterModelVersionModal from './RegisterModelVersionModal';

interface Props {
  checkpointUuid: string;
}

const RegisterModelVersionButton: React.FC<Props> = ({ checkpointUuid }) => {
  const [ showRegisterVersionModal, setShowRegisterVersionModel ] = useState(false);

  const openModal = useCallback(() => {
    setShowRegisterVersionModel(true);
  }, []);

  const closeModal = useCallback(() => {
    setShowRegisterVersionModel(false);
  }, []);

  return (
    <>
      <Button onClick={openModal}>
      Register Model
      </Button>
      <RegisterModelVersionModal
        checkpointUuid={checkpointUuid}
        visible={showRegisterVersionModal}
        onClose={closeModal} />
    </>
  );
};

export default RegisterModelVersionButton;
