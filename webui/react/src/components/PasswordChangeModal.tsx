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

const MODAL_HEADER_LABEL = 'Change Password';
export const OLD_PASSWORD_LABEL = 'Old Password';
const OLD_PASSWORD_NAME = 'oldPassword';
const NEW_PASSWORD_NAME = 'newPassword';
export const CONFIRM_PASSWORD_LABEL = 'Confirm Password';
const CONFIRM_PASSWORD_NAME = 'confirmPassword';
export const OK_BUTTON_LABEL = 'Change Password';
const INCORRECT_PASSWORD_MESSAGE = 'Incorrect password.';
const CONFIRM_PASSWORD_REQUIRED_MESSAGE = 'Confirmed password required.';
const PASSWORDS_NOT_MATCHING_MESSAGE = 'Passwords do not match.';
export const API_SUCCESS_MESSAGE = 'Password updated.';
const API_ERROR_MESSAGE = 'Could not update password.';

interface FormInputs {
  [OLD_PASSWORD_NAME]: string;
  [NEW_PASSWORD_NAME]: string;
  [CONFIRM_PASSWORD_NAME]: string;
}

interface Props {
  newPassword: string;
  onClose?: () => void;
}

const PasswordChangeModalComponent: React.FC<Props> = ({ newPassword, onClose }: Props) => {
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
      const password = newPassword;
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

  const handleClose = () => {
    onClose?.();
    form.resetFields();
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
      onClose={handleClose}>
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
          label={CONFIRM_PASSWORD_LABEL}
          name={CONFIRM_PASSWORD_NAME}
          rules={[
            { message: CONFIRM_PASSWORD_REQUIRED_MESSAGE, required: true },
            {
              message: PASSWORDS_NOT_MATCHING_MESSAGE,
              validator: (rule, value) => {
                return value === newPassword ? Promise.resolve() : Promise.reject();
              },
            },
          ]}>
          <Input.Password />
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default PasswordChangeModalComponent;
