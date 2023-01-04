import { Form, Input, message, Select, Switch, Typography } from 'antd';
import { FormInstance } from 'antd/lib/form/hooks/useForm';
import { filter } from 'fp-ts/lib/Set';
import React, { useCallback, useEffect, useState } from 'react';

import useAuthCheck from 'hooks/useAuthCheck';
import useFeature from 'hooks/useFeature';
import usePermissions from 'hooks/usePermissions';
import {
  assignRolesToUser,
  getUserRoles,
  patchUser,
  postUser,
  removeRolesFromUser,
} from 'services/api';
import Spinner from 'shared/components/Spinner';
import useModal, { ModalHooks as Hooks } from 'shared/hooks/useModal/useModal';
import { ErrorType } from 'shared/utils/error';
import { initKnowRoles, useKnownRoles } from 'stores/knowRoles';
import { useCurrentUser } from 'stores/users';
import { DetailedUser, UserRole } from 'types';
import handleError from 'utils/error';
import { Loadable } from 'utils/loadable';

const ADMIN_NAME = 'admin';
export const ADMIN_LABEL = 'Admin';
export const API_SUCCESS_MESSAGE_CREATE =
  'New user with empty password has been created, advise user to reset password as soon as possible.';
const DISPLAY_NAME_NAME = 'displayName';
export const DISPLAY_NAME_LABEL = 'Display Name';
export const MODAL_HEADER_LABEL_CREATE = 'Add User';
const MODAL_HEADER_LABEL_VIEW = 'View User';
const MODAL_HEADER_LABEL_EDIT = 'Edit User';
const USER_NAME_NAME = 'username';
export const USER_NAME_LABEL = 'User Name';
const ROLE_LABEL = 'Roles';
const ROLE_NAME = 'roles';
export const BUTTON_NAME = 'Save';

interface Props {
  form: FormInstance;
  roles: UserRole[] | null;
  user?: DetailedUser;
  viewOnly?: boolean;
}

interface FormValues {
  ADMIN_NAME: boolean;
  DISPLAY_NAME_NAME?: string;
  USER_NAME_NAME: string;
}

const ModalForm: React.FC<Props> = ({ form, user, viewOnly, roles }) => {
  const rbacEnabled = useFeature().isOn('rbac');
  const { canAssignRoles, canModifyPermissions } = usePermissions();
  const knowRolesLoadable = useKnownRoles();
  const knownRoles = Loadable.getOrElse(initKnowRoles, knowRolesLoadable);

  useEffect(() => {
    form.setFieldsValue({
      [ADMIN_NAME]: user?.isAdmin,
      [DISPLAY_NAME_NAME]: user?.displayName,
      [ROLE_NAME]: roles?.map((r) => r.id),
    });
  }, [form, user, roles]);

  if (user !== undefined && roles === null && rbacEnabled && canAssignRoles({})) {
    return <Spinner tip="Loading roles..." />;
  }

  return (
    <Form<FormValues> form={form} labelCol={{ span: 24 }}>
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
      {!rbacEnabled && (
        <Form.Item label={ADMIN_LABEL} name={ADMIN_NAME} valuePropName="checked">
          <Switch disabled={viewOnly} />
        </Form.Item>
      )}
      {rbacEnabled && canModifyPermissions && (
        <>
          <Form.Item
            label={ROLE_LABEL}
            name={ROLE_NAME}>
            <Select
              disabled={(user !== undefined && roles === null) || viewOnly}
              mode="multiple"
              optionFilterProp="children"
              placeholder={viewOnly ? 'No Roles Added' : 'Add Roles'}
              showSearch>
              {Loadable.match(knowRolesLoadable, {
                Loaded: () =>
                  knownRoles.map((r: UserRole) => (
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
  onClose?: () => void;
  user?: DetailedUser;
}

interface ModalHooks extends Omit<Hooks, 'modalOpen'> {
  modalOpen: (viewOnly?: boolean) => void;
}

const useModalCreateUser = ({ onClose, user }: ModalProps): ModalHooks => {
  const [form] = Form.useForm();
  const { modalOpen: openOrUpdate, ...modalHook } = useModal();
  const rbacEnabled = useFeature().isOn('rbac');
  // Null means the roles have not yet loaded
  const [userRoles, setUserRoles] = useState<UserRole[] | null>(null);
  const { canAssignRoles, canModifyPermissions } = usePermissions();
  const loadableCurrentUser = useCurrentUser();
  const currentUser = Loadable.match(loadableCurrentUser, {
    Loaded: (cUser) => cUser,
    NotLoaded: () => undefined,
  });
  const checkAuth = useAuthCheck();

  const fetchUserRoles = useCallback(async () => {
    if (user !== undefined && rbacEnabled && canAssignRoles({})) {
      try {
        const roles = await getUserRoles({ userId: user.id });
        setUserRoles(roles);
      } catch (e) {
        handleError(e, { publicSubject: "Unable to fetch this user's roles." });
      }
    }
  }, [user, canAssignRoles, rbacEnabled]);

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

      const newRoles: Set<number> = new Set(formData[ROLE_NAME]);
      const oldRoles = new Set((userRoles ?? []).map((r) => r.id));
      const rolesToAdd = filter((r: number) => !oldRoles.has(r))(newRoles);
      const rolesToRemove = filter((r: number) => !newRoles.has(r))(oldRoles);

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
          if (currentUser && currentUser.id === user.id) checkAuth();
          message.success('User has been updated');
        } else {
          formData['active'] = true;
          const u = await postUser({ user: formData });
          const uid = u.user?.id;
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
    [
      form,
      onClose,
      user,
      handleCancel,
      userRoles,
      canModifyPermissions,
      fetchUserRoles,
      checkAuth,
      currentUser,
    ],
  );

  const modalOpen = useCallback(
    (viewOnly?: boolean) => {
      openOrUpdate({
        closable: true,
        // passing a default brandind due to changes on the initial state
        content: <ModalForm form={form} roles={userRoles} user={user} viewOnly={viewOnly} />,
        icon: null,
        okText: viewOnly ? 'Close' : BUTTON_NAME,
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
    [form, handleCancel, handleOk, openOrUpdate, user, userRoles],
  );

  return { modalOpen, ...modalHook };
};

export default useModalCreateUser;
