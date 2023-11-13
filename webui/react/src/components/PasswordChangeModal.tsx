import Form from 'hew/Form';
import Input from 'hew/Input';
import { Modal } from 'hew/Modal';
import { useToast } from 'hew/Toast';
import { Loadable } from 'hew/utils/loadable';
import React, { useId, useState } from 'react';

import { login, setUserPassword } from 'services/api';
import userStore from 'stores/users';
import handleError, { ErrorType } from 'utils/error';
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

const FORM_ID = 'edit-password-form';

interface FormInputs {
  [OLD_PASSWORD_NAME]: string;
  [NEW_PASSWORD_NAME]: string;
  [CONFIRM_PASSWORD_NAME]: string;
}

interface Props {
  newPassword: string;
  onSubmit?: () => void;
}

const PasswordChangeModalComponent: React.FC<Props> = ({ newPassword, onSubmit }: Props) => {
  const idPrefix = useId();
  const [form] = Form.useForm<FormInputs>();
  const currentUser = Loadable.getOrElse(undefined, useObservable(userStore.currentUser));
  const [disabled, setDisabled] = useState<boolean>(true);
  const { openToast } = useToast();
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
    form.validateFields();

    try {
      const password = newPassword;
      await setUserPassword({ password, userId: currentUser?.id ?? 0 });
      openToast({ severity: 'Confirm', title: API_SUCCESS_MESSAGE });
      form.resetFields();
      onSubmit?.();
    } catch (e) {
      openToast({ severity: 'Error', title: API_ERROR_MESSAGE });
      handleError(e, { silent: true, type: ErrorType.Input });

      // Re-throw error to prevent modal from getting dismissed.
      throw e;
    }
  };

  const handleClose = () => {
    form.resetFields();
  };

  return (
    <Modal
      cancel
      size="small"
      submit={{
        disabled,
        form: idPrefix + FORM_ID,
        handleError,
        handler: handleSubmit,
        text: OK_BUTTON_LABEL,
      }}
      title={MODAL_HEADER_LABEL}
      onClose={handleClose}>
      <p>Please confirm your password change</p>
      <Form form={form} id={idPrefix + FORM_ID} onFieldsChange={handleFieldsChange}>
        <Form.Item
          label={OLD_PASSWORD_LABEL}
          name={OLD_PASSWORD_NAME}
          rules={[
            {
              message: INCORRECT_PASSWORD_MESSAGE,
              validator: async (_rule, value) => {
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
              validator: (_rule, value) => {
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
