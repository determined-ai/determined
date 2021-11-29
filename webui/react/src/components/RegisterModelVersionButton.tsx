import { Button } from 'antd';
import React, { useCallback, useState } from 'react';

import RegisterModelVersionModal from './RegisterModelVersionModal';

const RegisterModelVersionButton: React.FC = () => {
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
      <RegisterModelVersionModal visible={showRegisterVersionModal} onClose={closeModal} />
    </>
  );
};

export default RegisterModelVersionButton;
