import { useObservable } from 'micro-observables';

import Form from 'components/kit/Form';
import { Modal } from 'components/kit/Modal';
import Select, { Option } from 'components/kit/Select';
import { makeToast } from 'components/kit/Toast';
import { Loadable } from 'components/kit/utils/loadable';
import { assignRolesToUser } from 'services/api';
import roleStore from 'stores/roles';
import { UserRole } from 'types';
import handleError from 'utils/error';

const ROLE_LABEL = 'Global Roles';
const ROLE_NAME = 'roles';

type FormInputs = {
  [ROLE_NAME]: number[];
};

interface Props {
  userIds: number[];
  clearTableSelection: () => void;
  fetchUsers: () => void;
}

const SetUserRolesModalComponent = ({
  userIds,
  clearTableSelection,
  fetchUsers,
}: Props): JSX.Element => {
  const [form] = Form.useForm<FormInputs>();
  const knownRoles = useObservable(roleStore.roles);

  const onSubmit = async () => {
    const values = await form.validateFields();

    try {
      const roleIds = Array.from(new Set(values[ROLE_NAME]));
      const params = userIds.map((userId) => ({ roleIds, userId }));
      await assignRolesToUser(params);
      makeToast({
        title: 'Successfully set roles',
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
        form: 'SetUserRolesModalComponent',
        handleError,
        handler: onSubmit,
        text: 'Submit',
      }}
      title="Set Selected Users' Roles">
      <Form form={form} layout="vertical">
        <Form.Item
          label={ROLE_LABEL}
          name={ROLE_NAME}
          rules={[{ message: `${ROLE_LABEL} is required`, required: true }]}>
          <Select loading={Loadable.isNotLoaded(knownRoles)} mode="multiple">
            {Loadable.isLoaded(knownRoles) ? (
              <>
                {knownRoles.data.map((r: UserRole) => (
                  <Option key={r.id} value={r.id}>
                    {r.name}
                  </Option>
                ))}
              </>
            ) : undefined}
          </Select>
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default SetUserRolesModalComponent;
