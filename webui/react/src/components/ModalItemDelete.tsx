import { Modal } from 'antd';

const showModalItemCannotDelete = (): void => {
  Modal.confirm({
    closable: true,
    content: 'Only the item creator or an admin can delete this item.',
    icon: null,
    maskClosable: true,
    okText: 'Ok',
    title: 'Unable to Delete',
  });
};

export default showModalItemCannotDelete;
