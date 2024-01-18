import { Modal } from 'hew/Modal';

interface Props {
}

const ManageNodesModalComponent = ({}: Props): JSX.Element => {
  return (
    <Modal
      cancel
      size="medium"
      title="Manage Resource Pool Nodes">
      <div>Hello World</div>
    </Modal>
  );
};

export default ManageNodesModalComponent;
