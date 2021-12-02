import { Button } from 'antd';
import React, { useCallback, useState } from 'react';

import NewModelModal from './NewModelModal';

const NewModelButton: React.FC = () => {
  const [ showNewModelModal, setShowNewModelModal ] = useState(false);

  const openModal = useCallback(() => {
    setShowNewModelModal(true);
  }, []);

  const closeModal = useCallback(() => {
    setShowNewModelModal(false);
  }, []);

  return (
    <>
      <Button onClick={openModal}>
      New Model
      </Button>
      <NewModelModal visible={showNewModelModal} onClose={closeModal} />
    </>
  );
};

export default NewModelButton;
