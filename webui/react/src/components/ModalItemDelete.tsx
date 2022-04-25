import { Modal } from 'antd';

const showModalItemCannotDelete = (): void => {
  Modal.confirm({
    closable: true,
    content: `Only the item creator or an admin can 
    delete items from the model registry.`,
    icon: null,
    maskClosable: true,
    okText: 'Ok',
    title: 'Unable to Delete',
  });
};

export default showModalItemCannotDelete;
