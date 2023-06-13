import { Select } from 'antd';
import React, { useCallback, useState } from 'react';

import Form from 'components/kit/Form';
import Icon from 'components/kit/Icon';
import { Modal } from 'components/kit/Modal';
import Nameplate from 'components/kit/Nameplate';
import UserBadge from 'components/UserBadge';
import { assignRolesToGroup, assignRolesToUser } from 'services/api';
import { V1Role } from 'services/api-ts-sdk';
import { User, UserOrGroup } from 'types';
import { message } from 'utils/dialogApi';
import { DetError, ErrorLevel, ErrorType } from 'utils/error';
import handleError from 'utils/error';
import { getIdFromUserOrGroup, getName, isUser } from 'utils/user';

interface Props {
  addableUsersAndGroups: UserOrGroup[];
  onClose?: () => void;
  rolesAssignableToScope: V1Role[];
  workspaceId: number;
}
interface FormInputs {
  roleId: number;
  userOrGroupId: string;
}

interface SearchProp {
  label: {
    props: {
      name?: string;
      user?: User;
    };
  };
}

const WorkspaceMemberAddModalComponent: React.FC<Props> = ({
  addableUsersAndGroups,
  rolesAssignableToScope,
  onClose,
  workspaceId,
}: Props) => {
  const [selectedOption, setSelectedOption] = useState<UserOrGroup>();
  const [form] = Form.useForm<FormInputs>();

  const handleFilter = useCallback((search: string, option?: SearchProp): boolean => {
    if (!option) return false;
    const label = option.label;
    return (
      label.props.name?.includes(search) ||
      label.props.user?.username?.includes(search) ||
      label.props.user?.displayName?.includes(search) ||
      false
    );
  }, []);

  const handleSelect = useCallback(
    (value: string) => {
      const userOrGroup = addableUsersAndGroups.find((u) => {
        if (isUser(u) && value.substring(0, 2) === 'u_') {
          const user = u;
          return user.id === Number(value.substring(2));
        } else if (!isUser(u) && value.substring(0, 2) === 'g_') {
          const group = u;
          return group.groupId === Number(value.substring(2));
        }
      });
      setSelectedOption(userOrGroup);
    },
    [addableUsersAndGroups],
  );

  const handleSubmit = useCallback(async () => {
    const values = await form.validateFields();
    try {
      if (values && selectedOption) {
        isUser(selectedOption)
          ? await assignRolesToUser({
              roleIds: [values.roleId],
              scopeWorkspaceId: workspaceId,
              userId: Number(values.userOrGroupId.substring(2)),
            })
          : await assignRolesToGroup({
              groupId: Number(values.userOrGroupId.substring(2)),
              roleIds: [values.roleId],
              scopeWorkspaceId: workspaceId,
            });
        form.resetFields();
        setSelectedOption(undefined);
        onClose?.();
        message.success(`${getName(selectedOption)} added to workspace.`);
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
  }, [form, selectedOption, workspaceId, onClose]);

  return (
    <Modal
      cancel
      size="small"
      submit={{
        handleError,
        handler: handleSubmit,
        text: 'Add Member',
      }}
      title="Add Member">
      <Form autoComplete="off" form={form} layout="vertical">
        <Form.Item
          label="User or Group"
          name="userOrGroupId"
          rules={[{ message: 'User or group is required ', required: true }]}>
          <Select
            filterOption={handleFilter}
            options={addableUsersAndGroups.map((option) => ({
              label: isUser(option) ? (
                <UserBadge compact user={option as User} />
              ) : (
                <Nameplate
                  compact
                  icon={<Icon name="group" title="Group" />}
                  name={getName(option)}
                />
              ),
              value: (isUser(option) ? 'u_' : 'g_') + getIdFromUserOrGroup(option),
            }))}
            placeholder="User or Group"
            showSearch
            size="large"
            onSelect={handleSelect}
          />
        </Form.Item>
        <Form.Item
          label="Role"
          name="roleId"
          rules={[{ message: 'Role is required ', required: true }]}>
          <Select placeholder="Role">
            {rolesAssignableToScope.map((role) => (
              <Select.Option key={role.roleId} value={role.roleId}>
                {role.name}
              </Select.Option>
            ))}
          </Select>
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default WorkspaceMemberAddModalComponent;
