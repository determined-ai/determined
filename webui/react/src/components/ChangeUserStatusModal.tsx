import Form from 'hew/Form';
import { Modal } from 'hew/Modal';
import Select, { Option } from 'hew/Select';
import { useToast } from 'hew/Toast';
import { ValueOf } from 'hew/utils/types';

import { patchUsers } from 'services/api';
import handleError from 'utils/error';

const STATUS_NAME = 'status';

const StatusType = {
  Activate: 'activate',
  Deactivate: 'deactivate',
} as const;

type StatusType = ValueOf<typeof StatusType>;

type FormInputs = {
  [STATUS_NAME]: StatusType;
};

interface Props {
  userIds: number[];
  clearTableSelection: () => void;
  fetchUsers: () => void;
}

const ChangeUserStatusModalComponent = ({
  userIds,
  clearTableSelection,
  fetchUsers,
}: Props): JSX.Element => {
  const [form] = Form.useForm<FormInputs>();
  const { openToast } = useToast();
  const onSubmit = async () => {
    const values = await form.validateFields();

    try {
      await patchUsers({ activate: values[STATUS_NAME] === StatusType.Activate, userIds });
      openToast({ title: 'Successfully changed status' });
      clearTableSelection();
    } catch (e) {
      handleError(e);
    } finally {
      fetchUsers();
    }
  };

  return (
    <Modal
      cancel
      size="small"
      submit={{
        form: 'ChangeUserStatusModalComponent',
        handleError,
        handler: onSubmit,
        text: 'Submit',
      }}
      title="Change Selected Users' Status">
      <Form form={form} layout="vertical">
        <Form.Item
          label="Status"
          name={STATUS_NAME}
          rules={[{ message: 'This field is required', required: true }]}>
          <Select allowClear placeholder="Select Status">
            <Option value={StatusType.Activate}>Activate</Option>
            <Option value={StatusType.Deactivate}>Deactivate</Option>
          </Select>
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default ChangeUserStatusModalComponent;
