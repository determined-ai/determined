import { Form, Input, message, Switch } from 'antd';
import { FormInstance } from 'antd/lib/form/hooks/useForm';
import React, { useCallback } from 'react';

import { useStore } from 'contexts/Store';
import { postUser } from 'services/api';
import useModal, { ModalHooks } from 'shared/hooks/useModal/useModal';
import { ErrorType } from 'shared/utils/error';
import { BrandingType } from 'types';
import handleError from 'utils/error';

export const MODAL_HEADER_LABEL = 'Create User';
export const USER_NAME_NAME = 'username';
export const USER_NAME_LABEL = 'User Name';
export const DISPLAY_NAME_NAME = 'displayName';
export const DISPLAY_NAME_LABEL = 'Display Name';
export const ADMIN_NAME = 'admin';
export const ADMIN_LABEL = 'Admin';
export const API_SUCCESS_MESSAGE = `New user with empty password has been created, 
advice user to reset password as soon as possible.`;

interface Props {
  branding: BrandingType;
  form: FormInstance;
}

const ModalForm: React.FC<Props> = ({ form, branding }) => (
  <Form
    form={form}
    labelCol={{ span: 8 }}
    wrapperCol={{ span: 14 }}>
    <Form.Item
      label={USER_NAME_LABEL}
      name={USER_NAME_NAME}
      required
      rules={[
        {
          message: 'Please type in your username.',
          required: true,
        },
      ]}
      validateTrigger={[ 'onSubmit' ]}>
      <Input autoFocus maxLength={128} placeholder="user name" />
    </Form.Item>
    <Form.Item
      label={DISPLAY_NAME_LABEL}
      name={DISPLAY_NAME_NAME}>
      <Input maxLength={128} placeholder="display name" />
    </Form.Item>
    {branding === BrandingType.Determined ? (
      <Form.Item
        label={ADMIN_LABEL}
        name={ADMIN_NAME}>
        <Switch defaultChecked={false} />
      </Form.Item>
    ) : null }
  </Form>
);

interface ModalProps {
  onClose?: () => void;
}

const useModalCreateUser = ({ onClose }: ModalProps): ModalHooks => {
  const [ form ] = Form.useForm();
  const { info } = useStore();

  const { modalOpen: openOrUpdate, ...modalHook } = useModal();

  const handleCancel = useCallback(() => {
    form.resetFields();
  }, [ form ]);

  const handleOkay = useCallback(async () => {
    await form.validateFields();

    try {
      const formData = form.getFieldsValue();
      formData.admin = !!formData.admin;
      await postUser(formData);
      message.success(API_SUCCESS_MESSAGE);
      form.resetFields();
      onClose?.();
    } catch (e) {
      message.error('error creating new user');
      handleError(e, { silent: true, type: ErrorType.Input });

      // Re-throw error to prevent modal from getting dismissed.
      throw e;
    }
  }, [ form, onClose ]);

  const modalOpen = useCallback(() => {
    openOrUpdate({
      closable: true,
      // passing a default brandind due to changes on the initial state
      content: <ModalForm branding={info.branding || BrandingType.Determined} form={form} />,
      icon: null,
      okText: 'Create User',
      onCancel: handleCancel,
      onOk: handleOkay,
      title: <h5>{MODAL_HEADER_LABEL}</h5>,
    });
  }, [ form, handleCancel, handleOkay, openOrUpdate, info ]);

  return { modalOpen, ...modalHook };
};

export default useModalCreateUser;
