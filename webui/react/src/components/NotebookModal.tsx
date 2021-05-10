import { Modal } from 'antd';
import React, { } from 'react';

interface Props {
  forceVisible?: boolean
}

const NotebookModal: React.FC<Props> = (
  { forceVisible = false }: Props,
) => {
  return <Modal visible={forceVisible}>
    <p>test</p>
  </Modal>;
};

export default NotebookModal;
