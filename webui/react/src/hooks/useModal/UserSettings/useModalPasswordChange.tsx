import { Form, Input, message } from 'antd';
import { FormInstance } from 'antd/lib/form/hooks/useForm';
import React, { useCallback } from 'react';

import { login, setUserPassword } from 'services/api';
import useModal, { ModalHooks } from 'shared/hooks/useModal/useModal';
import { ErrorType } from 'shared/utils/error';
import { useAuth } from 'stores/auth';
import handleError from 'utils/error';
import { Loadable } from 'utils/loadable';

interface Props {
  form: FormInstance;
  username?: string;
}

export const MODAL_HEADER_LABEL = 'Change Password';
export const OLD_PASSWORD_LABEL = 'Old Password';
export const OLD_PASSWORD_NAME = 'oldPassword';
export const NEW_PASSWORD_LABEL = 'New Password';
export const NEW_PASSWORD_NAME = 'newPassword';
export const CONFIRM_PASSWORD_LABEL = 'Confirm Password';
export const CONFIRM_PASSWORD_NAME = 'confirmPassword';
export const CANCEL_BUTTON_LABEL = 'Cancel';
export const OK_BUTTON_LABEL = 'Change Password';
export const INCORRECT_PASSWORD_MESSAGE = 'Incorrect password.';
export const NEW_PASSWORD_REQUIRED_MESSAGE = 'New password required.';
export const PASSWORD_TOO_SHORT_MESSAGE = "Password isn't long enough.";
export const PASSWORD_UPPERCASE_MESSAGE = 'Password must include a uppercase letter.';
export const PASSWORD_LOWERCASE_MESSAGE = 'Password must include a lowercase letter.';
export const PASSWORD_NUMBER_MESSAGE = 'Password must include a number.';
export const CONFIRM_PASSWORD_REQUIRED_MESSAGE = 'Confirmed password required.';
export const PASSWORDS_NOT_MATCHING_MESSAGE = 'Passwords do not match.';
export const API_SUCCESS_MESSAGE = 'Password updated.';
export const API_ERROR_MESSAGE = 'Could not update password.';

const ModalForm: React.FC<Props> = ({ form, username = '' }) => (
  <Form form={form} layout="vertical">
    <Form.Item
      label={OLD_PASSWORD_LABEL}
      name={OLD_PASSWORD_NAME}
      required
      rules={[
        {
          message: INCORRECT_PASSWORD_MESSAGE,
          validator: async (rule, value) => {
            await login({ password: value ?? '', username });
          },
        },
      ]}
      validateTrigger={['onSubmit']}>
      <Input.Password />
    </Form.Item>
    <Form.Item
      label={NEW_PASSWORD_LABEL}
      name={NEW_PASSWORD_NAME}
      rules={[
        { message: NEW_PASSWORD_REQUIRED_MESSAGE, required: true },
        { message: PASSWORD_TOO_SHORT_MESSAGE, min: 8 },
        {
          message: PASSWORD_UPPERCASE_MESSAGE,
          pattern: /[A-Z]+/,
        },
        {
          message: PASSWORD_LOWERCASE_MESSAGE,
          pattern: /[a-z]+/,
        },
        {
          message: PASSWORD_NUMBER_MESSAGE,
          pattern: /\d/,
        },
      ]}>
      <Input.Password />
    </Form.Item>
    <Form.Item
      dependencies={[NEW_PASSWORD_NAME]}
      label={CONFIRM_PASSWORD_LABEL}
      name={CONFIRM_PASSWORD_NAME}
      rules={[
        { message: CONFIRM_PASSWORD_REQUIRED_MESSAGE, required: true },
        {
          message: PASSWORDS_NOT_MATCHING_MESSAGE,
          validator: (rule, value) => {
            return value === form.getFieldValue(NEW_PASSWORD_NAME)
              ? Promise.resolve()
              : Promise.reject();
          },
        },
      ]}>
      <Input.Password />
    </Form.Item>
    <Form.Item>
      Password must be at least 8 characters and contain an uppercase letter, a lowercase letter,
      and a number.
    </Form.Item>
  </Form>
);

const useModalPasswordChange = (): ModalHooks => {
  const [form] = Form.useForm();
  const loadableAuth = useAuth();
  const authUser = Loadable.match(loadableAuth.auth, {
    Loaded: (auth) => auth.user,
    NotLoaded: () => undefined,
  });

  const { modalOpen: openOrUpdate, ...modalHook } = useModal();

  const handleCancel = useCallback(() => form.resetFields(), [form]);

  const handleOkay = useCallback(async () => {
    await form.validateFields();

    try {
      const password = form.getFieldValue(NEW_PASSWORD_NAME);
      await setUserPassword({ password, userId: authUser?.id ?? 0 });
      message.success(API_SUCCESS_MESSAGE);
      form.resetFields();
    } catch (e) {
      message.error(API_ERROR_MESSAGE);
      handleError(e, { silent: true, type: ErrorType.Input });

      // Re-throw error to prevent modal from getting dismissed.
      throw e;
    }
  }, [authUser?.id, form]);

  const modalOpen = useCallback(() => {
    openOrUpdate({
      closable: true,
      content: <ModalForm form={form} username={authUser?.username} />,
      icon: null,
      okText: OK_BUTTON_LABEL,
      onCancel: handleCancel,
      onOk: handleOkay,
      title: <h5>{MODAL_HEADER_LABEL}</h5>,
    });
  }, [authUser?.username, form, handleCancel, handleOkay, openOrUpdate]);

  return { modalOpen, ...modalHook };
};

export default useModalPasswordChange;
