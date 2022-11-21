import { Form, Input, message, Select, Typography } from 'antd';
import { FormInstance } from 'antd/lib/form/hooks/useForm';
import { filter } from 'fp-ts/lib/Set';
import React, { useCallback, useEffect, useState } from 'react';

import { useStore } from 'contexts/Store';
import useFeature from 'hooks/useFeature';
import usePermissions from 'hooks/usePermissions';
import {
  assignRolesToGroup,
  createGroup,
  getGroup,
  getGroupRoles,
  removeRolesFromGroup,
  updateGroup,
} from 'services/api';
import { V1GroupDetails, V1GroupSearchResult } from 'services/api-ts-sdk/models';
import useModal, { ModalHooks } from 'shared/hooks/useModal/useModal';
import { ErrorType } from 'shared/utils/error';
import { DetailedUser, UserRole } from 'types';
import handleError from 'utils/error';
import { getDisplayName } from 'utils/user';

export const MODAL_HEADER_LABEL_CREATE = 'Create Group';
export const MODAL_HEADER_LABEL_EDIT = 'Edit Group';
export const GROUP_NAME_NAME = 'name';
export const GROUP_NAME_LABEL = 'Group Name';
export const GROUP_ROLE_NAME = 'roles';
export const GROUP_ROLE_LABEL = 'Roles';
export const USER_ADD_NAME = 'addUsers';
export const USER_ADD_LABEL = 'Add Users';
export const USER_REMOVE_LABEL = 'Remove Users';
export const USER_REMOVE_NAME = 'removeUsers';
export const USER_LABEL = 'Users';
export const API_SUCCESS_MESSAGE_CREATE = 'New group has been created.';
export const API_SUCCESS_MESSAGE_EDIT = 'Group has been updated.';

interface Props {
  form: FormInstance;
  group?: V1GroupSearchResult;
  groupRoles?: UserRole[];
  users: DetailedUser[];
}

const ModalForm: React.FC<Props> = ({ form, users, group, groupRoles }) => {
  const rbacEnabled = useFeature().isOn('rbac');
  const { canModifyPermissions } = usePermissions();
  const [isLoading, setIsLoading] = useState(true);

  const { knownRoles } = useStore();

  const [groupDetail, setGroupDetail] = useState<V1GroupDetails>();

  const fetchGroup = useCallback(async () => {
    if (group?.group.groupId) {
      try {
        const response = await getGroup({ groupId: group?.group.groupId });
        setGroupDetail(response.group);
        form.setFieldsValue({
          [GROUP_NAME_NAME]: group.group.name,
          [GROUP_ROLE_NAME]: groupRoles?.map((r) => r.id),
        });
      } catch (e) {
        handleError(e, { publicSubject: 'Unable to fetch groups.' });
      } finally {
        setIsLoading(false);
      }
    } else {
      setIsLoading(false);
    }
  }, [group, form, groupRoles]);

  useEffect(() => {
    fetchGroup();
  }, [fetchGroup]);

  return (
    <Form form={form} labelCol={{ span: 8 }} wrapperCol={{ span: 14 }}>
      <Form.Item
        label={GROUP_NAME_LABEL}
        name={GROUP_NAME_NAME}
        required
        rules={[
          {
            message: 'Please type in your group name.',
            required: true,
          },
        ]}
        validateTrigger={['onSubmit', 'onChange']}>
        <Input autoFocus maxLength={128} placeholder="Group Name" />
      </Form.Item>
      {group ? (
        <Form.Item label={USER_ADD_LABEL} name={USER_ADD_NAME}>
          <Select
            loading={isLoading}
            mode="multiple"
            optionFilterProp="children"
            placeholder="Add Users"
            showSearch>
            {users
              .filter((u) => !groupDetail?.users?.map((gu) => gu.id).includes(u.id))
              .map((u) => (
                <Select.Option key={u.id} value={u.id}>
                  {getDisplayName(u)}
                </Select.Option>
              ))}
          </Select>
        </Form.Item>
      ) : (
        <Form.Item label={USER_LABEL} name={USER_ADD_NAME}>
          <Select mode="multiple" optionFilterProp="children" placeholder="Add Users" showSearch>
            {users.map((u) => (
              <Select.Option key={u.id} value={u.id}>
                {getDisplayName(u)}
              </Select.Option>
            ))}
          </Select>
        </Form.Item>
      )}
      {rbacEnabled && canModifyPermissions && group && (
        <>
          <Form.Item label={GROUP_ROLE_LABEL} name={GROUP_ROLE_NAME}>
            <Select
              mode="multiple"
              optionFilterProp="children"
              placeholder={'Add Roles'}
              showSearch>
              {knownRoles.map((r) => (
                <Select.Option
                  disabled={groupRoles?.find((gr) => gr.id === r.id)?.fromWorkspace?.length}
                  key={r.id}
                  value={r.id}>
                  {r.name}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Typography.Text type="secondary">
            Note that roles inherited from workspaces cannot be removed here.
          </Typography.Text>
        </>
      )}
    </Form>
  );
};

interface ModalProps {
  group?: V1GroupSearchResult;
  onClose?: () => void;
  users: DetailedUser[];
}

const useModalCreateGroup = ({ onClose, users, group }: ModalProps): ModalHooks => {
  const [form] = Form.useForm();
  const rbacEnabled = useFeature().isOn('rbac');
  const { canModifyPermissions } = usePermissions();
  const [groupRoles, setGroupRoles] = useState<UserRole[]>([]);
  const { modalOpen: openOrUpdate, ...modalHook } = useModal();

  const handleCancel = useCallback(() => {
    form.resetFields();
  }, [form]);

  const fetchGroupRoles = useCallback(async () => {
    if (group?.group.groupId && rbacEnabled) {
      try {
        const roles = await getGroupRoles({ groupId: group.group.groupId });
        setGroupRoles(roles);
      } catch (e) {
        handleError(e, { publicSubject: "Unable to fetch this group's roles." });
      }
    }
  }, [group, rbacEnabled]);

  useEffect(() => {
    fetchGroupRoles();
  }, [fetchGroupRoles]);

  const onOk = useCallback(async () => {
    await form.validateFields();

    try {
      const formData = form.getFieldsValue();
      if (group) {
        await updateGroup({ groupId: group.group.groupId, ...formData });
        if (canModifyPermissions && group.group.groupId) {
          const newRoles: Set<number> = new Set(formData.roles);
          const oldRoles = new Set((groupRoles ?? []).map((r) => r.id));

          const rolesToAdd = filter((r: number) => !oldRoles.has(r))(newRoles);
          const rolesToRemove = filter((r: number) => !newRoles.has(r))(oldRoles);

          rolesToAdd.size > 0 &&
            (await assignRolesToGroup({
              groupId: group.group.groupId,
              roleIds: Array.from(rolesToAdd),
            }));
          rolesToRemove.size > 0 &&
            (await removeRolesFromGroup({
              groupId: group.group.groupId,
              roleIds: Array.from(rolesToRemove),
            }));
          await fetchGroupRoles();
        }
        message.success(API_SUCCESS_MESSAGE_EDIT);
      } else {
        await createGroup(formData);
        message.success(API_SUCCESS_MESSAGE_CREATE);
      }
      form.resetFields();
      onClose?.();
    } catch (e) {
      message.error('Error creating new group.');
      handleError(e, { silent: true, type: ErrorType.Input });

      // Re-throw error to prevent modal from getting dismissed.
      throw e;
    }
  }, [form, onClose, group, canModifyPermissions, fetchGroupRoles, groupRoles]);

  const modalOpen = useCallback(() => {
    openOrUpdate({
      closable: true,
      content: <ModalForm form={form} group={group} groupRoles={groupRoles} users={users} />,
      icon: null,
      okText: group ? MODAL_HEADER_LABEL_EDIT : MODAL_HEADER_LABEL_CREATE,
      onCancel: handleCancel,
      onOk: onOk,
      title: <h5>{group ? MODAL_HEADER_LABEL_EDIT : MODAL_HEADER_LABEL_CREATE}</h5>,
    });
  }, [form, handleCancel, onOk, openOrUpdate, users, group, groupRoles]);

  return { modalOpen, ...modalHook };
};

export default useModalCreateGroup;
