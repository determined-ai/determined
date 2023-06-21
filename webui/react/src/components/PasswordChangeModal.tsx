import React, { useState } from 'react';

import Form from 'components/kit/Form';
import Input from 'components/kit/Input';
import { Modal } from 'components/kit/Modal';
import { login, setUserPassword } from 'services/api';
import userStore from 'stores/users';
import { message } from 'utils/dialogApi';
import { ErrorType } from 'utils/error';
import handleError from 'utils/error';
import { Loadable } from 'utils/loadable';
import { useObservable } from 'utils/observable';

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

interface FormInputs {
  [OLD_PASSWORD_NAME]: string;
  [NEW_PASSWORD_NAME]: string;
  [CONFIRM_PASSWORD_NAME]: string;
}

const PasswordChangeModalComponent: React.FC = () => {
  const [form] = Form.useForm<FormInputs>();
  const currentUser = Loadable.getOrElse(undefined, useObservable(userStore.currentUser));
  const [disabled, setDisabled] = useState<boolean>(true);

  const submitValidatedFields = [OLD_PASSWORD_NAME];
  const requiredFields = [NEW_PASSWORD_NAME, CONFIRM_PASSWORD_NAME];

  const handleFieldsChange = () => {
    const fields = form.getFieldsError();
    const hasError = fields.some(
      (f) => f.name[0] && !submitValidatedFields.includes(f.name[0] as string) && f.errors.length,
    );
    const values = form.getFieldsValue();
    const missingRequiredFields = Object.entries(values).some(([key, value]) => {
      return requiredFields.includes(key) && !value;
    });
    setDisabled(hasError || missingRequiredFields);
  };

  const handleSubmit = async () => {
    await form.validateFields();

    try {
      const password = form.getFieldValue(NEW_PASSWORD_NAME);
      await setUserPassword({ password, userId: currentUser?.id ?? 0 });
      message.success(API_SUCCESS_MESSAGE);
      form.resetFields();
    } catch (e) {
      message.error(API_ERROR_MESSAGE);
      handleError(e, { silent: true, type: ErrorType.Input });

      // Re-throw error to prevent modal from getting dismissed.
      throw e;
    }
  };

  return (
    <Modal
      cancel
      size="small"
      submit={{
        disabled,
        handleError,
        handler: handleSubmit,
        text: OK_BUTTON_LABEL,
      }}
      title={MODAL_HEADER_LABEL}
      onClose={form.resetFields}>
      <Form form={form} onFieldsChange={handleFieldsChange}>
        <Form.Item
          label={OLD_PASSWORD_LABEL}
          name={OLD_PASSWORD_NAME}
          rules={[
            {
              message: INCORRECT_PASSWORD_MESSAGE,
              validator: async (rule, value) => {
                await login({ password: value ?? '', username: currentUser?.username ?? '' });
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
          Password must be at least 8 characters and contain an uppercase letter, a lowercase
          letter, and a number.
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default PasswordChangeModalComponent;
