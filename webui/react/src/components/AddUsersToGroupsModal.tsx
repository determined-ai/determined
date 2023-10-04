import Form from 'components/kit/Form';
import { Modal } from 'components/kit/Modal';
import Select, { Option } from 'components/kit/Select';
import { makeToast } from 'components/kit/Toast';
import { assignMultipleGroups } from 'services/api';
import { V1GroupSearchResult } from 'services/api-ts-sdk';
import handleError from 'utils/error';

const GROUPS_NAME = 'groups';

type FormInputs = {
  [GROUPS_NAME]: number[];
};

interface Props {
  userIds: number[];
  groupOptions: V1GroupSearchResult[];
  clearTableSelection: () => void;
  fetchUsers: () => void;
}

const AddUsersToGroupsModalComponent = ({
  userIds,
  groupOptions,
  clearTableSelection,
  fetchUsers,
}: Props): JSX.Element => {
  const [form] = Form.useForm<FormInputs>();

  const onSubmit = async () => {
    const values = await form.validateFields();

    try {
      const groupIds = Array.from(
        new Set(values[GROUPS_NAME].flatMap((v) => (v !== undefined ? [v] : []))),
      );
      await assignMultipleGroups({ addGroups: groupIds, removeGroups: [], userIds });
      makeToast({
        title: 'Successfully added to groups',
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
        form: 'AddUsersToGroupsModalComponent',
        handleError,
        handler: onSubmit,
        text: 'Submit',
      }}
      title="Add Selected to Groups">
      <Form form={form} layout="vertical">
        <Form.Item
          label="Groups"
          name={GROUPS_NAME}
          rules={[{ message: 'This field is required', required: true }]}>
          <Select mode="multiple" placeholder="Select Groups">
            {groupOptions.map((go) => (
              <Option key={go.group.groupId} value={go.group.groupId}>
                {go.group.name}
              </Option>
            ))}
          </Select>
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default AddUsersToGroupsModalComponent;
