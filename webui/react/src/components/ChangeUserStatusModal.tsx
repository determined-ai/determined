import Form from 'components/kit/Form';
import { Modal } from 'components/kit/Modal';
import Select, { Option } from 'components/kit/Select';
import { makeToast } from 'components/kit/Toast';
import { patchUsers } from 'services/api';
import handleError from 'utils/error';

type FormInputs = {
  status: 'activate' | 'deactivate';
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

  const onSubmit = async () => {
    const values = await form.validateFields();

    try {
      await patchUsers({ activate: values.status === 'activate', userIds });
      makeToast({
        title: 'Successfully changed status',
      });
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
      <Form form={form} layout="vertical" name="control-hooks">
        <Form.Item label="Status" name="status" rules={[{ required: true }]}>
          <Select allowClear placeholder="Select a option and change input text above">
            <Option value="activate">Activate</Option>
            <Option value="deactivate">Deactivate</Option>
          </Select>
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default ChangeUserStatusModalComponent;
