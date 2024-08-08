import { filter } from 'fp-ts/lib/Set';
import Form, { hasErrors } from 'hew/Form';
import Input from 'hew/Input';
import { Modal } from 'hew/Modal';
import Select, { Option } from 'hew/Select';
import Spinner from 'hew/Spinner';
import { useToast } from 'hew/Toast';
import Toggle from 'hew/Toggle';
import { Body } from 'hew/Typography';
import { Loadable } from 'hew/utils/loadable';
import { FormInstance } from 'rc-field-form';
import React, { useEffect, useId, useState } from 'react';

import Link from 'components/Link';
import { PASSWORD_RULES } from 'constants/passwordRules';
import useAuthCheck from 'hooks/useAuthCheck';
import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';
import { assignRolesToUser, patchUser, postUser, removeRolesFromUser } from 'services/api';
import { V1PatchUser } from 'services/api-ts-sdk';
import determinedStore from 'stores/determinedInfo';
import roleStore from 'stores/roles';
import userStore from 'stores/users';
import { DetailedUser, UserRole } from 'types';
import handleError, { ErrorType } from 'utils/error';
import { useObservable } from 'utils/observable';

import css from './CreateUserModal.module.scss';

const ADMIN_NAME = 'admin';
export const ADMIN_LABEL = 'Admin';
export const API_SUCCESS_MESSAGE_CREATE =
  'New user has been created; advise user to change password as soon as possible.';
export const API_SUCCESS_MESSAGE_CREATE_REMOTE =
  'New remote user has been created; please configure access in IdP.';
const DISPLAY_NAME_NAME = 'displayName';
export const DISPLAY_NAME_LABEL = 'Display Name';
export const MODAL_HEADER_LABEL_CREATE = 'Add User';
const MODAL_HEADER_LABEL_VIEW = 'View User';
const MODAL_HEADER_LABEL_EDIT = 'Edit User';
const USER_NAME_NAME = 'username';
export const USER_NAME_LABEL = 'User Name';
const USER_PASSWORD_NAME = 'password';
export const USER_PASSWORD_LABEL = 'User Password';
const USER_PASSWORD_CONFIRM_NAME = 'confirmPassword';
export const USER_PASSWORD_CONFIRM_LABEL = 'Confirm User Password';
const REMOTE_LABEL =
  'Remote (prevents password sign-on and requires user to sign-on using external IdP)';
const REMOTE_NAME = 'remote';
const ROLE_LABEL = 'Global Roles';
const ROLE_NAME = 'roles';
export const BUTTON_NAME = 'Save';
const ACTIVE_NAME = 'active';
const FORM_ID = 'create-user-form';

interface Props {
  user?: DetailedUser;
  viewOnly?: boolean;
  userRoles?: Loadable<UserRole[]>;
  onClose?: () => void;
}

interface FormInputs {
  [USER_NAME_NAME]: string;
  [USER_PASSWORD_NAME]: string;
  [DISPLAY_NAME_NAME]: string;
  [ADMIN_NAME]: boolean;
  [REMOTE_NAME]: boolean;
  [ROLE_NAME]: number[];
  [ACTIVE_NAME]: boolean;
}

const CreateUserModalComponent: React.FC<Props> = ({
  onClose,
  user,
  userRoles,
  viewOnly,
}: Props) => {
  const { openToast } = useToast();
  const idPrefix = useId();
  const [form] = Form.useForm<FormInputs>();
  const { rbacEnabled } = useObservable(determinedStore.info);
  // Null means the roles have not yet loaded
  const { canAssignRoles, canModifyPermissions } = usePermissions();
  const currentUser = Loadable.getOrElse(undefined, useObservable(userStore.currentUser));
  const checkAuth = useAuthCheck();

  const knownRoles = useObservable(roleStore.roles);

  const [canSubmit, setCanSubmit] = useState(!!user);
  const [isRemote, setIsRemote] = useState(false);

  const editPasswordRules = PASSWORD_RULES.map((rule) => {
    return (form: FormInstance) => (form.getFieldValue(USER_PASSWORD_NAME) ? rule : { min: 0 });
  });

  const handleSubmit = async () => {
    if (viewOnly) {
      form.resetFields();
      return;
    }

    const formData = await form.validateFields();

    const newRoles: Set<number> = new Set(formData[ROLE_NAME]);
    const oldRoles = new Set((userRoles?.getOrElse([]) ?? []).map((r) => r.id));
    const rolesToAdd = filter((r: number) => !oldRoles.has(r))(newRoles);
    const rolesToRemove = filter((r: number) => !newRoles.has(r))(oldRoles);

    try {
      if (user) {
        const patchParams: V1PatchUser = {
          active: formData[ACTIVE_NAME],
          admin: formData[ADMIN_NAME],
          displayName: formData[DISPLAY_NAME_NAME],
          remote: formData[REMOTE_NAME],
          username: formData[USER_NAME_NAME],
        };
        if (formData[USER_PASSWORD_NAME]?.length > 0) {
          patchParams.password = formData[USER_PASSWORD_NAME];
        }
        await patchUser({ userId: user.id, userParams: patchParams });
        if (canModifyPermissions) {
          rolesToAdd.size > 0 &&
            (await assignRolesToUser([{ roleIds: Array.from(rolesToAdd), userId: user.id }]));
          rolesToRemove.size > 0 &&
            (await removeRolesFromUser({ roleIds: Array.from(rolesToRemove), userId: user.id }));
        }
        if (currentUser?.id === user.id) checkAuth();
        openToast({ severity: 'Confirm', title: 'User has been updated' });
      } else {
        formData[ACTIVE_NAME] = true;
        const u = await postUser({
          password: formData[USER_PASSWORD_NAME],
          user: {
            active: formData[ACTIVE_NAME],
            admin: formData[ADMIN_NAME],
            displayName: formData[DISPLAY_NAME_NAME],
            remote: formData[REMOTE_NAME],
            username: formData[USER_NAME_NAME],
          },
        });
        const uid = u.user?.id;
        if (uid && rolesToAdd.size > 0) {
          await assignRolesToUser([{ roleIds: Array.from(rolesToAdd), userId: uid }]);
        }
        openToast({
          severity: 'Confirm',
          title: u.user?.remote ? API_SUCCESS_MESSAGE_CREATE_REMOTE : API_SUCCESS_MESSAGE_CREATE,
        });
        form.resetFields();
      }
      onClose?.();
    } catch (e) {
      openToast({
        severity: 'Error',
        title: user ? 'Error updating user' : 'Error creating new user',
      });
      handleError(e, { silent: true, type: ErrorType.Input });

      // Re-throw error to prevent modal from getting dismissed.
      throw e;
    }
  };

  useEffect(() => {
    form.setFieldsValue({
      [ADMIN_NAME]: user?.isAdmin,
      [DISPLAY_NAME_NAME]: user?.displayName,
      [ROLE_NAME]: userRoles?.getOrElse([]).map((r) => r.id),
    });
    setIsRemote(!!user?.remote);
  }, [form, user, userRoles]);

  return (
    <Modal
      cancel
      data-test-component="createUserModal"
      size="small"
      submit={{
        disabled: !canSubmit,
        form: idPrefix + FORM_ID,
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
        spinning={user !== undefined && userRoles?.isNotLoaded && rbacEnabled && canAssignRoles({})}
        tip="Loading roles...">
        <Form
          className={css.createUserModalForm}
          form={form}
          id={idPrefix + FORM_ID}
          onFieldsChange={() =>
            setCanSubmit(
              !hasErrors(form) &&
                form.getFieldValue(USER_NAME_NAME) &&
                (user ||
                  (form.getFieldValue(USER_PASSWORD_NAME) &&
                    form.getFieldValue(USER_PASSWORD_CONFIRM_NAME))),
            )
          }>
          <Form.Item
            initialValue={user?.username}
            label={USER_NAME_LABEL}
            name={USER_NAME_NAME}
            required>
            <Input
              autoFocus
              data-testid="username"
              disabled={!!user}
              maxLength={128}
              placeholder="User name"
            />
          </Form.Item>
          <Form.Item label={DISPLAY_NAME_LABEL} name={DISPLAY_NAME_NAME}>
            <Input
              data-testid="displayName"
              disabled={viewOnly}
              maxLength={128}
              placeholder="Display Name"
            />
          </Form.Item>
          {!rbacEnabled && (
            <Form.Item
              data-testid="isAdmin"
              label={ADMIN_LABEL}
              name={ADMIN_NAME}
              valuePropName="checked">
              <Toggle disabled={viewOnly} />
            </Form.Item>
          )}
          {rbacEnabled && canModifyPermissions && (
            <Form.Item
              initialValue={user?.remote}
              label={REMOTE_LABEL}
              name={REMOTE_NAME}
              valuePropName="checked">
              <Toggle
                checked={isRemote}
                data-testid="isRemote"
                disabled={viewOnly}
                onChange={setIsRemote}
              />
            </Form.Item>
          )}
          {!isRemote && (
            <>
              <Form.Item
                initialValue=""
                label={USER_PASSWORD_LABEL}
                name={USER_PASSWORD_NAME}
                required={!user && !isRemote}
                rules={editPasswordRules}>
                <Input.Password data-testid="password" disabled={viewOnly} placeholder="Password" />
              </Form.Item>
              <Form.Item
                dependencies={[USER_PASSWORD_NAME]}
                label={USER_PASSWORD_CONFIRM_LABEL}
                name={USER_PASSWORD_CONFIRM_NAME}
                required={!user && !isRemote}
                rules={[
                  ({ getFieldValue }) => ({
                    validator(_, value: string) {
                      if (!value || getFieldValue(USER_PASSWORD_NAME) === value) {
                        return Promise.resolve();
                      }
                      return Promise.reject(
                        new Error('The new password does not match the confirmation field'),
                      );
                    },
                  }),
                ]}>
                <Input.Password data-testid="confirmPassword" disabled={viewOnly} />
              </Form.Item>
            </>
          )}
          {rbacEnabled && canModifyPermissions && (
            <>
              <Form.Item label={ROLE_LABEL} name={ROLE_NAME}>
                <Select
                  data-testid="roles"
                  disabled={(user !== undefined && userRoles?.isNotLoaded) || viewOnly}
                  loading={Loadable.isNotLoaded(knownRoles)}
                  mode="multiple"
                  placeholder={viewOnly ? 'No Roles Added' : 'Add Roles'}>
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
              <Body inactive>
                Users may have additional inherited global or workspace roles not reflected here.
                &nbsp;
                <Link external path={paths.docs('/cluster-setup-guide/security/rbac.html')} popout>
                  Learn more
                </Link>
              </Body>
            </>
          )}
        </Form>
      </Spinner>
    </Modal>
  );
};

export default CreateUserModalComponent;
