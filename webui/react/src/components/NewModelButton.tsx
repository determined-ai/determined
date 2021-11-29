import { Button } from 'antd';
import React, { useCallback, useState } from 'react';

import NewModelModal from './NewModelModal';

const NewModelButton: React.FC = () => {
  const [ showNewModelModal, setShowNewModelModel ] = useState(false);

  const openModal = useCallback(() => {
    setShowNewModelModel(true);
  }, []);

  const closeModal = useCallback(() => {
    setShowNewModelModel(false);
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
