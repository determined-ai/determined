import Form from 'hew/Form';
import Icon from 'hew/Icon';
import { Modal } from 'hew/Modal';
import Nameplate from 'hew/Nameplate';
import Select, { Option, RefSelectProps } from 'hew/Select';
import { useToast } from 'hew/Toast';
import _ from 'lodash';
import React, { useCallback, useEffect, useId, useRef, useState } from 'react';

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

const USER_PREFIX = 'u_';
const GROUP_PREFIX = 'g_';

const WorkspaceMemberAddModalComponent: React.FC<Props> = ({
  addableUsersAndGroups,
  rolesAssignableToScope,
  onClose,
  workspace,
}: Props) => {
  const idPrefix = useId();
  const [filteredOption, setFilteredOption] = useState<UserOrGroup[]>([]);
  const { openToast } = useToast();
  const [form] = Form.useForm<FormInputs>();
  const ref = useRef<RefSelectProps>(null);

  useEffect(() => {
    ref.current?.focus();
  }, []);

  const handleSearch = useCallback(
    (value: string) => {
      // Only show options for select if user had typed in something.
      if (value) {
        const regex = new RegExp(value, 'i');
        setFilteredOption(
          _.filter(addableUsersAndGroups, (o) =>
            isUser(o)
              ? (o.displayName && regex.test(o.displayName)) || regex.test(o.username) || false
              : (o.name && regex.test(o.name)) || false,
          ),
        );
      } else {
        setFilteredOption([]);
      }
    },
    [addableUsersAndGroups],
  );

  const handleSubmit = useCallback(async () => {
    const values = await form.validateFields();
    try {
      if (values) {
        const userOrGroup = _.groupBy(values.userOrGroupId, (o) => o.substring(0, 2));
        const groupPayload = _.map(userOrGroup[GROUP_PREFIX], (o) => ({
          groupId: Number(o.substring(USER_PREFIX.length)),
          roleIds: [values.roleId],
          scopeWorkspaceId: workspace.id,
        }));
        groupPayload.length > 0 && (await assignRolesToGroup(groupPayload));

        const userPayload = _.map(userOrGroup[USER_PREFIX], (o) => ({
          roleIds: [values.roleId],
          scopeWorkspaceId: workspace.id,
          userId: Number(o.substring(USER_PREFIX.length)),
        }));
        userPayload.length > 0 && (await assignRolesToUser(userPayload));

        form.resetFields();
        onClose?.();
        openToast({
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
  }, [form, workspace.id, onClose, openToast]);

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
            {filteredOption.map((option) => (
              <Option
                key={(isUser(option) ? USER_PREFIX : GROUP_PREFIX) + getIdFromUserOrGroup(option)}
                label={isUser(option) ? option.username : option.name}
                value={
                  (isUser(option) ? USER_PREFIX : GROUP_PREFIX) + getIdFromUserOrGroup(option)
                }>
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
