import { Button, Form, Input, message } from 'antd';
import { ModalStaticFunctions } from 'antd/es/modal/confirm';
import React, { useCallback, useState } from 'react';

import { useStore } from 'contexts/Store';
import { login, setUserPassword } from 'services/api';
import handleError from 'utils/error';

import useModal, { ModalHooks } from '../useModal';

import css from './useModalChangePassword.module.scss';

interface Props {
  onComplete: () => void;
}

const ChangePassword: React.FC<Props> = ({ onComplete }) => {
  const { auth } = useStore();
  const username = auth.user?.username ?? '';
  const [ form ] = Form.useForm();
  const [ isUpdating, setIsUpdating ] = useState(false);

  const handleFormCancel = useCallback(() => {
    form.resetFields();
    onComplete();
  }, [ form, onComplete ]);

  const handleFormSubmit = useCallback(async () => {
    setIsUpdating(true);
    try {
      await setUserPassword({
        password: form.getFieldValue('newPassword'),
        username,
      });
      message.success('Password updated');
      form.resetFields();
      onComplete();
    } catch (e) {
      message.error('Could not update password');
      handleError(e);
    }
    setIsUpdating(false);
  }, [ form, onComplete, username ]);

  return (
    <div className={css.base}>
      <Form form={form} layout="vertical" onFinish={handleFormSubmit}>
        <Form.Item
          label="Old Password"
          name="oldPassword"
          required
          rules={[
            {
              message: 'Incorrect password',
              validator: async (rule, value) => {
                return await login({
                  password: value ?? '',
                  username,
                });
              },
            },
          ]}
          validateTrigger={[ 'onBlur', 'onSubmit' ]}>
          <Input.Password />
        </Form.Item>
        <Form.Item
          label="New Password"
          name="newPassword"
          rules={[
            { message: 'New password required', required: true },
            { message: "Your new password isn't long enough", min: 8 },
            {
              message: 'Your new password must include an uppercase letter',
              pattern: /[A-Z]+/,
            },
            {
              message: 'Your new password must include a lowercase letter',
              pattern: /[a-z]+/,
            },
            {
              message: 'Your new password must include a number',
              pattern: /\d/,
            },
          ]}>
          <Input.Password />
        </Form.Item>
        <Form.Item
          dependencies={[ 'newPassword' ]}
          label="Confirm Password"
          name="confirmPassword"
          rules={[
            { message: 'Confirmed password required', required: true },
            {
              message: 'Your new passwords do not match',
              validator: (rule, value) => {
                return value === form.getFieldValue('newPassword')
                  ? Promise.resolve()
                  : Promise.reject();
              },
            },
          ]}>
          <Input.Password />
        </Form.Item>
        <Form.Item>
          <span>
            Password must be at least 8 characters
            and contain an uppercase letter, a lowercase letter, and a number.
          </span>
        </Form.Item>
        <Form.Item>
          {/* override modal buttons with form buttons
          to ensure form validation works as intended */}
          <div className={css.buttons}>
            <Button onClick={handleFormCancel}>Cancel</Button>
            <Button htmlType="submit" loading={isUpdating} type="primary">
              Change password
            </Button>
          </div>
        </Form.Item>
      </Form>
    </div>
  );
};

const useModalChangePassword = (modal: Omit<ModalStaticFunctions, 'warn'>): ModalHooks => {
  const { modalClose, modalOpen: openOrUpdate, modalRef } = useModal({ modal });

  const modalOpen = useCallback(() => {
    openOrUpdate({
      className: css.noFooter,
      closable: true,
      content: <ChangePassword onComplete={modalClose} />,
      icon: null,
      title: <h5>Change password</h5>,
    });
  }, [ modalClose, openOrUpdate ]);

  return { modalClose, modalOpen, modalRef };
};

export default useModalChangePassword;
