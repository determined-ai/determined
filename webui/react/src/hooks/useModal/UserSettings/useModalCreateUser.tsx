import { Form, Input, InputNumber, message, Select, Switch, Typography } from 'antd';
import { FormInstance } from 'antd/lib/form/hooks/useForm';
import { filter } from 'fp-ts/lib/Set';
import React, { useCallback, useEffect, useState } from 'react';

import useFeature from 'hooks/useFeature';
import usePermissions from 'hooks/usePermissions';
import {
  assignRolesToUser,
  getUserRoles,
  patchUser,
  postUser,
  removeRolesFromUser,
  updateGroup,
} from 'services/api';
import { V1AgentUserGroup, V1GroupSearchResult } from 'services/api-ts-sdk';
import Spinner from 'shared/components/Spinner';
import useModal, { ModalHooks as Hooks } from 'shared/hooks/useModal/useModal';
import { ErrorType } from 'shared/utils/error';
import { initKnowRoles, useKnownRoles } from 'stores/knowRoles';
import { DetailedUser, UserRole } from 'types';
import handleError from 'utils/error';
import { Loadable } from 'utils/loadable';

export const ADMIN_NAME = 'admin';
export const ADMIN_LABEL = 'Admin';
export const API_SUCCESS_MESSAGE_CREATE = `New user with empty password has been created,
advise user to reset password as soon as possible.`;
export const API_SUCCESS_MESSAGE_EDIT = 'User has been updated';
export const DISPLAY_NAME_NAME = 'displayName';
export const DISPLAY_NAME_LABEL = 'Display Name';
export const MODAL_HEADER_LABEL_CREATE = 'Create User';
export const MODAL_HEADER_LABEL_EDIT = 'Edit User';
export const MODAL_HEADER_LABEL_VIEW = 'View User';
export const USER_NAME_NAME = 'username';
export const USER_NAME_LABEL = 'User Name';
export const GROUP_LABEL = 'Add to Groups';
export const GROUP_NAME = 'groups';
export const ROLE_LABEL = 'Roles';
export const ROLE_NAME = 'roles';

interface Props {
  form: FormInstance;
  groups: V1GroupSearchResult[];
  roles: UserRole[] | null;
  user?: DetailedUser;
  viewOnly?: boolean;
}

interface FormValues {
  ADMIN_NAME: boolean;
  DISPLAY_NAME_NAME?: string;
  GROUP_NAME?: number;
  USER_NAME_NAME: string;
}

const ModalForm: React.FC<Props> = ({ form, user, groups, viewOnly, roles }) => {
  const rbacEnabled = useFeature().isOn('rbac');
  const { canAssignRoles, canModifyPermissions } = usePermissions();
  const knowRolesLoadable = useKnownRoles();
  const knownRoles = Loadable.getOrElse(initKnowRoles, knowRolesLoadable);

  const useAgent = Form.useWatch<FormValues>('useAgent', form);

  useEffect(() => {
    form.setFieldsValue({
      [ADMIN_NAME]: user?.isAdmin,
      [DISPLAY_NAME_NAME]: user?.displayName,
      [ROLE_NAME]: roles?.map((r) => r.id),
    });
    if (user?.agentUserGroup) {
      form.setFieldsValue({
        agentGid: user?.agentUserGroup.agentGid,
        agentGroup: user?.agentUserGroup.agentGroup,
        agentUid: user?.agentUserGroup.agentUid,
        agentUser: user?.agentUserGroup.agentUser,
        useAgent: true,
      });
    } else {
      form.setFieldsValue({
        agentGid: undefined,
        agentGroup: undefined,
        agentUid: undefined,
        agentUser: undefined,
        useAgent: false,
      });
    }
  }, [form, user, roles]);

  if (user !== undefined && roles === null && rbacEnabled && canAssignRoles({})) {
    return <Spinner tip="Loading roles..." />;
  }

  return (
    <Form<FormValues> form={form} labelCol={{ span: 8 }} wrapperCol={{ span: 14 }}>
      <Form.Item
        initialValue={user?.username}
        label={USER_NAME_LABEL}
        name={USER_NAME_NAME}
        required
        rules={[
          {
            message: 'Please type in your username.',
            required: true,
          },
        ]}
        validateTrigger={['onSubmit']}>
        <Input autoFocus disabled={!!user} maxLength={128} placeholder="User Name" />
      </Form.Item>
      <Form.Item label={DISPLAY_NAME_LABEL} name={DISPLAY_NAME_NAME}>
        <Input disabled={viewOnly} maxLength={128} placeholder="Display Name" />
      </Form.Item>
      <Form.Item label="Configure Agent" name="useAgent" valuePropName="checked">
        <Switch disabled={viewOnly} />
      </Form.Item>
      {useAgent && (
        <>
          <Form.Item
            label="Agent User ID"
            name="agentUid"
            rules={[{ message: 'Agent User ID is required ', required: true }]}>
            <InputNumber disabled={viewOnly} />
          </Form.Item>
          <Form.Item
            label="Agent User Name"
            name="agentUser"
            rules={[{ message: 'Agent User Name is required ', required: true }]}>
            <Input disabled={viewOnly} maxLength={100} />
          </Form.Item>
          <Form.Item
            label="Agent User Group ID"
            name="agentGid"
            rules={[{ message: 'Agent User Group ID is required ', required: true }]}>
            <InputNumber disabled={viewOnly} />
          </Form.Item>
          <Form.Item
            label="Agent Group Name"
            name="agentGroup"
            rules={[{ message: 'Agent Group Name is required ', required: true }]}>
            <Input disabled={viewOnly} maxLength={100} />
          </Form.Item>
        </>
      )}
      {!rbacEnabled && (
        <Form.Item label={ADMIN_LABEL} name={ADMIN_NAME} valuePropName="checked">
          <Switch disabled={viewOnly} />
        </Form.Item>
      )}
      {!user && rbacEnabled && (
        <Form.Item label={GROUP_LABEL} name={GROUP_NAME}>
          <Select
            mode="multiple"
            optionFilterProp="children"
            placeholder="Select Groups"
            showSearch>
            {groups.map((u) => (
              <Select.Option key={u.group.groupId} value={u.group.groupId}>
                {u.group.name}
              </Select.Option>
            ))}
          </Select>
        </Form.Item>
      )}
      {rbacEnabled && canModifyPermissions && (
        <>
          <Form.Item label={ROLE_LABEL} name={ROLE_NAME}>
            <Select
              disabled={(user !== undefined && roles === null) || viewOnly}
              mode="multiple"
              optionFilterProp="children"
              placeholder={viewOnly ? 'No Roles Added' : 'Add Roles'}
              showSearch>
              {Loadable.match(knowRolesLoadable, {
                Loaded: () =>
                  knownRoles.map((r) => (
                    <Select.Option
                      disabled={
                        roles?.find((ro) => ro.id === r.id)?.fromGroup?.length ||
                        roles?.find((ro) => ro.id === r.id)?.fromWorkspace?.length
                      }
                      key={r.id}
                      value={r.id}>
                      {r.name}
                    </Select.Option>
                  )),
                NotLoaded: () => undefined, // TODO show spinner when this is loading
              })}
            </Select>
          </Form.Item>
          <Typography.Text type="secondary">
            Note that roles inherited from user groups or workspaces cannot be removed here.
          </Typography.Text>
        </>
      )}
    </Form>
  );
};

interface ModalProps {
  groups: V1GroupSearchResult[];
  onClose?: () => void;
  user?: DetailedUser;
}

interface ModalHooks extends Omit<Hooks, 'modalOpen'> {
  modalOpen: (viewOnly?: boolean) => void;
}

const useModalCreateUser = ({ groups, onClose, user }: ModalProps): ModalHooks => {
  const [form] = Form.useForm();
  const { modalOpen: openOrUpdate, ...modalHook } = useModal();
  const rbacEnabled = useFeature().isOn('rbac');
  // Null means the roles have not yet loaded
  const [userRoles, setUserRoles] = useState<UserRole[] | null>(null);
  const { canAssignRoles, canModifyPermissions } = usePermissions();

  const fetchUserRoles = useCallback(async () => {
    if (user !== undefined && rbacEnabled && canAssignRoles({})) {
      try {
        const roles = await getUserRoles({ userId: user.id });
        setUserRoles(roles);
      } catch (e) {
        handleError(e, { publicSubject: "Unable to fetch this user's roles." });
      }
    }
  }, [user, canAssignRoles]);

  useEffect(() => {
    fetchUserRoles();
  }, [fetchUserRoles]);

  const handleCancel = useCallback(() => {
    form.resetFields();
  }, [form]);

  const handleOk = useCallback(
    async (viewOnly?: boolean) => {
      if (viewOnly) {
        handleCancel();
        return;
      }
      await form.validateFields();

      const formData = form.getFieldsValue();

      const newRoles: Set<number> = new Set(formData.roles);
      const oldRoles = new Set((userRoles ?? []).map((r) => r.id));
      const rolesToAdd = filter((r: number) => !oldRoles.has(r))(newRoles);
      const rolesToRemove = filter((r: number) => !newRoles.has(r))(oldRoles);

      if (formData.useAgent || user) {
        const { agentUid, agentUser, agentGid, agentGroup } = formData;
        const agentUserGroup: V1AgentUserGroup = { agentGid, agentGroup, agentUid, agentUser };
        formData.agentUserGroup = agentUserGroup;
      }

      delete formData.useAgent;

      try {
        if (user) {
          await patchUser({ userId: user.id, userParams: formData });
          if (canModifyPermissions) {
            rolesToAdd.size > 0 &&
              (await assignRolesToUser({ roleIds: Array.from(rolesToAdd), userId: user.id }));
            rolesToRemove.size > 0 &&
              (await removeRolesFromUser({ roleIds: Array.from(rolesToRemove), userId: user.id }));
          }
          fetchUserRoles();
          message.success(API_SUCCESS_MESSAGE_EDIT);
        } else {
          formData['active'] = true;
          const u = await postUser({ user: formData });
          const uid = u.user?.id;
          if (uid && formData.groups) {
            (formData.groups as number[]).forEach(async (gid) => {
              await updateGroup({ addUsers: [uid], groupId: gid });
            });
          }
          if (uid && rolesToAdd.size > 0) {
            await assignRolesToUser({ roleIds: Array.from(rolesToAdd), userId: uid });
          }

          message.success(API_SUCCESS_MESSAGE_CREATE);
          form.resetFields();
        }
        onClose?.();
      } catch (e) {
        message.error(user ? 'Error updating user' : 'Error creating new user');
        handleError(e, { silent: true, type: ErrorType.Input });

        // Re-throw error to prevent modal from getting dismissed.
        throw e;
      }
    },
    [form, onClose, user, handleCancel, userRoles, canModifyPermissions, fetchUserRoles],
  );

  const modalOpen = useCallback(
    (viewOnly?: boolean) => {
      openOrUpdate({
        closable: true,
        // passing a default brandind due to changes on the initial state
        content: (
          <ModalForm
            form={form}
            groups={groups}
            roles={userRoles}
            user={user}
            viewOnly={viewOnly}
          />
        ),
        icon: null,
        okText: viewOnly ? 'Close' : user ? 'Update' : 'Create User',
        onCancel: handleCancel,
        onOk: () => handleOk(viewOnly),
        title: (
          <h5>
            {user
              ? viewOnly
                ? MODAL_HEADER_LABEL_VIEW
                : MODAL_HEADER_LABEL_EDIT
              : MODAL_HEADER_LABEL_CREATE}
          </h5>
        ),
        width: 520,
      });
    },
    [form, handleCancel, handleOk, openOrUpdate, user, groups, userRoles],
  );

  return { modalOpen, ...modalHook };
};

export default useModalCreateUser;
