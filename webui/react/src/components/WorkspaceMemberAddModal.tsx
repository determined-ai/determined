import Select, { Option, RefSelectProps } from 'determined-ui/Select';
import Form from 'determined-ui/Form';
import Icon from 'determined-ui/Icon';
import { Modal } from 'determined-ui/Modal';
import Nameplate from 'determined-ui/Nameplate';
import { makeToast } from 'determined-ui/Toast';
import React, { useCallback, useId, useState } from 'react';

import UserBadge from 'components/UserBadge';
import { assignRolesToGroup, assignRolesToUser } from 'services/api';
import { V1Role } from 'services/api-ts-sdk';
import { User, UserOrGroup, Workspace } from 'types';
import handleError, { DetError, ErrorLevel, ErrorType } from 'utils/error';
import { getIdFromUserOrGroup, getName, isUser } from 'utils/user';

const FORM_ID = 'add-workspace-member-form';

interface Props {
  addableUsersAndGroups: UserOrGroup[];
  onClose?: () => void;
  rolesAssignableToScope: V1Role[];
  workspace: Workspace;
}
interface FormInputs {
  roleId: number;
  userOrGroupId: string;
}

const WorkspaceMemberAddModalComponent: React.FC<Props> = ({
  addableUsersAndGroups,
  rolesAssignableToScope,
  onClose,
  workspace,
}: Props) => {
  const idPrefix = useId();
  const [filteredOption, setFilteredOption] = useState<UserOrGroup[]>([]);
  const [form] = Form.useForm<FormInputs>();
  const ref = useRef<RefSelectProps>(null);

  useEffect(() => {
    ref.current?.focus();
  }, [ref]);

  const handleSearch = useCallback(
    (value: string) => {
      const regex = new RegExp(value, 'i');
      setFilteredOption(
        value
          ? _.filter(addableUsersAndGroups, (o) =>
              isUser(o)
                ? (o.displayName && regex.test(o.displayName)) || regex.test(o.username) || false
                : (o.name && regex.test(o.name)) || false,
            )
          : [],
      );
    },
    [addableUsersAndGroups],
  );

  const handleSubmit = useCallback(async () => {
    const values = await form.validateFields();
    try {
      if (values) {
        const userOrGroup = _.groupBy(values.userOrGroupId, (o) => o.substring(0, 2));
        const groupPayload = _.map(userOrGroup['g_'], (o) => ({
          groupId: Number(o.substring(2)),
          roleIds: [values.roleId],
          scopeWorkspaceId: workspace.id,
        }));
        groupPayload.length > 0 && (await assignRolesToGroup(groupPayload));

        const userPayload = _.map(userOrGroup['u_'], (o) => ({
          roleIds: [values.roleId],
          scopeWorkspaceId: workspace.id,
          userId: Number(o.substring(2)),
        }));
        userPayload.length > 0 && (await assignRolesToUser(userPayload));

        form.resetFields();
        onClose?.();
        makeToast({
          severity: 'Confirm',
          title: `${values.userOrGroupId.length} users or groups added to workspace.`,
        });
      }
    } catch (e) {
      if (e instanceof DetError) {
        handleError(e, {
          level: e.level,
          publicMessage: e.publicMessage,
          publicSubject: 'Unable to add user or group to workspace.',
          silent: false,
          type: e.type,
        });
      } else {
        handleError(e, {
          level: ErrorLevel.Error,
          publicMessage: 'Please try again later.',
          publicSubject: 'Unable to add user or group to workspace.',
          silent: false,
          type: ErrorType.Server,
        });
      }
    }
  }, [form, workspace.id, onClose]);

  return (
    <Modal
      cancel
      size="small"
      submit={{
        disabled:
          !form.getFieldValue('userOrGroupId') || form.getFieldError('userOrGroupId').length > 0,
        form: idPrefix + FORM_ID,
        handleError,
        handler: handleSubmit,
        text: 'Add to Workspace',
      }}
      title={`Add members to ${workspace.name}`}>
      <Form
        autoComplete="off"
        form={form}
        id={idPrefix + FORM_ID}
        initialValues={{ roleId: rolesAssignableToScope.find((r) => r.name === 'Editor')?.roleId }}
        layout="vertical">
        <Form.Item
          label="Find and Select Users or Groups"
          name="userOrGroupId"
          rules={[{ message: 'User or group is required ', required: true }]}>
          <Select
            filterOption={false}
            mode="multiple"
            optionLabelProp="label"
            placeholder="Add Users or Groups"
            ref={ref}
            searchable={true}
            onBlur={() => setFilteredOption([])}
            onSearch={handleSearch}>
            {_.map(filteredOption, (option) => (
              <Option
                key={(isUser(option) ? 'u_' : 'g_') + getIdFromUserOrGroup(option)}
                label={isUser(option) ? option.username : option.name}
                value={(isUser(option) ? 'u_' : 'g_') + getIdFromUserOrGroup(option)}>
                {isUser(option) ? (
                  <UserBadge compact user={option as User} />
                ) : (
                  <Nameplate
                    compact
                    icon={<Icon name="group" title="Group" />}
                    name={getName(option)}
                  />
                )}
              </Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item
          label="Assign Workspace Role"
          name="roleId"
          rules={[{ message: 'Role is required ', required: true }]}>
          <Select
            options={rolesAssignableToScope.map((role) => ({
              label: role.name,
              value: role.roleId,
            }))}
            placeholder="Role"
          />
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default WorkspaceMemberAddModalComponent;
