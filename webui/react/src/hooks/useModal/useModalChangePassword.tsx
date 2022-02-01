import { Button, Form, Input } from 'antd';
import React from 'react';

import { useStore } from 'contexts/Store';
import { login, setUserPassword } from 'services/api';

import useModal, { ModalHooks } from './useModal';
import css from './useModalChangePassword.module.scss';

const useModalChangePassword = (): ModalHooks => {
  const { modalClose, modalOpen: openOrUpdate, modalRef } = useModal();
  const { auth } = useStore();
  const username = auth.user?.username || 'Anonymous';
  const [ form ] = Form.useForm();

  const getModalContent = () => {
    const submitForm = async (): Promise<void> => {
      await setUserPassword({
        password: form.getFieldValue('newPassword'),
        username,
      });
      modalClose();
    };

    return (
      <div className={css.base}>
        <Form form={form} layout="vertical" onFinish={submitForm}>
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
                    username: username || '',
                  });
                },
              },
            ]}
            validateTrigger="onSubmit">
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
              Password must be at least 8 charactes and contain one uppercase,
              lowercase.
            </span>
          </Form.Item>
          <Form.Item>
            <div className={css.buttons}>
              <Button onClick={() => onCancel()}>Cancel</Button>
              <Button htmlType="submit" type="primary">
                Change Password
              </Button>
            </div>
          </Form.Item>
        </Form>
      </div>
    );
  };

  const onCancel = () => {
    form.resetFields();
    modalClose();
  };

  const modalOpen = () => {
    openOrUpdate({
      className: css.noFooter,
      closable: true,
      content: getModalContent(),
      icon: null,
      onCancel,
      title: 'Change Password',
    });
  };

  return { modalClose, modalOpen, modalRef };
};

export default useModalChangePassword;
