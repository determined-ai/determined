import { Button, Form, Input } from 'antd';
import React, { useCallback } from 'react';

import { useStore } from 'contexts/Store';
import { login, setUserPassword } from 'services/api';
import handleError from 'utils/error';

import useModal, { ModalHooks } from './useModal';
import css from './useModalChangePassword.module.scss';

const useModalChangePassword = (): ModalHooks => {
  const { modalClose, modalOpen: openOrUpdate, modalRef } = useModal();
  const { auth } = useStore();
  const username = auth.user?.username || 'Anonymous';
  const [ form ] = Form.useForm();

  const handleFormCancel = useCallback(() => {
    form.resetFields();
    modalClose();
  }, [ form, modalClose ]);

  const handleFormSubmit = useCallback(async () => {
    try {
      await setUserPassword({
        password: form.getFieldValue('newPassword'),
        username,
      });
      modalClose();
    } catch (e) {
      handleError(e);
    }
  }, [ form, modalClose, username ]);

  const getModalContent = useCallback(() => {
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
                    password: value || '',
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
                message: 'Your new password must include an uppercase',
                pattern: /[A-Z]+/,
              },
              {
                message: 'Your new password must include a lowercase',
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
              <Button htmlType="submit" type="primary">
                Change Password
              </Button>
            </div>
          </Form.Item>
        </Form>
      </div>
    );
  }, [ form, username, handleFormSubmit, handleFormCancel ]);

  const modalOpen = useCallback(() => {
    openOrUpdate({
      className: css.noFooter,
      closable: true,
      content: getModalContent(),
      icon: null,
      okText: 'Change Password',
      title: 'Change Password',
    });
  }, [ getModalContent, openOrUpdate ]);

  return { modalClose, modalOpen, modalRef };
};

export default useModalChangePassword;
