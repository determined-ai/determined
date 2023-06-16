import { Select, Switch, Typography } from 'antd';
import { filter } from 'fp-ts/lib/Set';
import React, { useCallback, useEffect, useState } from 'react';

import Form from 'components/kit/Form';
import Input from 'components/kit/Input';
import { Modal } from 'components/kit/Modal';
import Link from 'components/Link';
import Spinner from 'components/Spinner';
import useAuthCheck from 'hooks/useAuthCheck';
import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';
import {
  assignRolesToUser,
  getUserRoles,
  patchUser,
  postUser,
  removeRolesFromUser,
} from 'services/api';
import determinedStore from 'stores/determinedInfo';
import roleStore from 'stores/roles';
import userStore from 'stores/users';
import { DetailedUser, UserRole } from 'types';
import { message } from 'utils/dialogApi';
import { ErrorType } from 'utils/error';
import handleError from 'utils/error';
import { Loadable } from 'utils/loadable';
import { useObservable } from 'utils/observable';

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
const ROLE_LABEL = 'Global Roles';
const ROLE_NAME = 'roles';
export const BUTTON_NAME = 'Save';
const ACTIVE_NAME = 'active';

interface Props {
  user?: DetailedUser;
  viewOnly?: boolean;
  onClose?: () => void;
}

interface FormInputs {
  [USER_NAME_NAME]: string;
  [DISPLAY_NAME_NAME]: string;
  [ADMIN_NAME]: boolean;
  [ROLE_NAME]: number[];
  [ACTIVE_NAME]: boolean;
}

const CreateUserModalComponent: React.FC<Props> = ({ onClose, user, viewOnly }: Props) => {
  const [form] = Form.useForm<FormInputs>();
  const { rbacEnabled } = useObservable(determinedStore.info);
  // Null means the roles have not yet loaded
  const [userRoles, setUserRoles] = useState<UserRole[] | null>(null);
  const { canAssignRoles, canModifyPermissions } = usePermissions();
  const canAssignRolesFlag: boolean = canAssignRoles({});
  const currentUser = Loadable.getOrElse(undefined, useObservable(userStore.currentUser));
  const checkAuth = useAuthCheck();

  const username = Form.useWatch(USER_NAME_NAME, form);

  const knownRoles = useObservable(roleStore.roles);
  const fetchUserRoles = useCallback(async () => {
    if (user !== undefined && rbacEnabled && canAssignRolesFlag) {
      try {
        const roles = await getUserRoles({ userId: user.id });
        setUserRoles(roles?.filter((r) => r.fromUser));
      } catch (e) {
        handleError(e, { publicSubject: "Unable to fetch this user's roles." });
      }
    }
  }, [user, canAssignRolesFlag, rbacEnabled]);

  useEffect(() => {
    fetchUserRoles();
  }, [fetchUserRoles]);

  const handleSubmit = async (viewOnly?: boolean) => {
    if (viewOnly) {
      form.resetFields();
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
        formData[ACTIVE_NAME] = true;
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
  };

  useEffect(() => {
    form.setFieldsValue({
      [ADMIN_NAME]: user?.isAdmin,
      [DISPLAY_NAME_NAME]: user?.displayName,
      [ROLE_NAME]: userRoles?.map((r) => r.id),
    });
  }, [form, user, userRoles]);

  return (
    <Modal
      cancel
      size="small"
      submit={{
        disabled: !username,
        handleError,
        handler: handleSubmit,
        text: viewOnly ? 'Close' : BUTTON_NAME,
      }}
      title={
        user
          ? viewOnly
            ? MODAL_HEADER_LABEL_VIEW
            : MODAL_HEADER_LABEL_EDIT
          : MODAL_HEADER_LABEL_CREATE
      }
      onClose={form.resetFields}>
      <Spinner
        spinning={user !== undefined && userRoles === null && rbacEnabled && canAssignRoles({})}
        tip="Loading roles...">
        <Form form={form}>
          <Form.Item
            initialValue={user?.username}
            label={USER_NAME_LABEL}
            name={USER_NAME_NAME}
            required
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
              <Form.Item label={ROLE_LABEL} name={ROLE_NAME}>
                <Select
                  disabled={(user !== undefined && userRoles === null) || viewOnly}
                  loading={Loadable.isLoading(knownRoles)}
                  mode="multiple"
                  optionFilterProp="children"
                  placeholder={viewOnly ? 'No Roles Added' : 'Add Roles'}
                  showSearch>
                  {Loadable.isLoaded(knownRoles) ? (
                    <>
                      {knownRoles.data.map((r: UserRole) => (
                        <Select.Option key={r.id} value={r.id}>
                          {r.name}
                        </Select.Option>
                      ))}
                    </>
                  ) : undefined}
                </Select>
              </Form.Item>
              <Typography.Text type="secondary">
                Users may have additional inherited global or workspace roles not reflected here.
                &nbsp;
                <Link external path={paths.docs('/cluster-setup-guide/security/rbac.html')} popout>
                  Learn more
                </Link>
              </Typography.Text>
            </>
          )}
        </Form>
      </Spinner>
    </Modal>
  );
};

export default CreateUserModalComponent;
