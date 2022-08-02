import { Form, Input, message, Switch } from 'antd';
import { FormInstance } from 'antd/lib/form/hooks/useForm';
import React, { useCallback, useEffect } from 'react';

import { useStore } from 'contexts/Store';
import { patchUser, postUser } from 'services/api';
import useModal, { ModalHooks } from 'shared/hooks/useModal/useModal';
import { ErrorType } from 'shared/utils/error';
import { BrandingType, DetailedUser } from 'types';
import handleError from 'utils/error';

export const ADMIN_NAME = 'admin';
export const ADMIN_LABEL = 'Admin';
export const API_SUCCESS_MESSAGE_CREATE = `New user with empty password has been created, 
advise user to reset password as soon as possible.`;
export const API_SUCCESS_MESSAGE_EDIT = 'User has been updated';
export const DISPLAY_NAME_NAME = 'displayName';
export const DISPLAY_NAME_LABEL = 'Display Name';
export const MODAL_HEADER_LABEL_CREATE = 'Create User';
export const MODAL_HEADER_LABEL_EDIT = 'Edit User';
export const USER_NAME_NAME = 'username';
export const USER_NAME_LABEL = 'User Name';

interface Props {
  branding: BrandingType;
  form: FormInstance;
  user?: DetailedUser
}

interface FormValues {
  ADMIN_NAME: boolean;
  DISPLAY_NAME_NAME?: string;
  USER_NAME_NAME: string;
}

const ModalForm: React.FC<Props> = ({ form, branding, user }) => {
  useEffect(() => {
    form.setFieldsValue({
      [ADMIN_NAME]: user?.isAdmin,
      [DISPLAY_NAME_NAME]: user?.displayName,
    });
  }, [ user, form ]);
  return (
    <Form<FormValues>
      form={form}
      labelCol={{ span: 8 }}
      wrapperCol={{ span: 14 }}>
      <Form.Item
        initialValue={user?.username}
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
        <Input autoFocus disabled={!!user} maxLength={128} placeholder="User Name" />
      </Form.Item>
      <Form.Item
        label={DISPLAY_NAME_LABEL}
        name={DISPLAY_NAME_NAME}>
        <Input maxLength={128} placeholder="Display Name" />
      </Form.Item>
      {branding === BrandingType.Determined ? (
        <Form.Item
          label={ADMIN_LABEL}
          name={ADMIN_NAME}
          valuePropName="checked">
          <Switch />
        </Form.Item>
      ) : null }
    </Form>
  );
};

interface ModalProps {
  onClose?: () => void;
  user?: DetailedUser
}

const useModalCreateUser = ({ onClose, user }: ModalProps): ModalHooks => {
  const [ form ] = Form.useForm();
  const { info } = useStore();
  const { modalOpen: openOrUpdate, ...modalHook } = useModal();

  const handleCancel = useCallback(() => {
    form.resetFields();
  }, [ form ]);

  const handleOk = useCallback(async () => {
    await form.validateFields();

    const formData = form.getFieldsValue();
    try {
      if (user) {
        await patchUser({ userId: user.id, userParams: formData });
        message.success(API_SUCCESS_MESSAGE_EDIT);
      } else {
        await postUser(formData);
        message.success(API_SUCCESS_MESSAGE_CREATE);
      }

      form.resetFields();
      onClose?.();
    } catch (e) {
      message.error(user ? 'Error updating user' : 'Error creating new user');
      handleError(e, { silent: true, type: ErrorType.Input });

      // Re-throw error to prevent modal from getting dismissed.
      throw e;
    }
  }, [ form, onClose, user ]);

  const modalOpen = useCallback(() => {
    openOrUpdate({
      closable: true,
      content: <ModalForm branding={info.branding} form={form} user={user} />,
      icon: null,
      okText: user ? 'Update' : 'Create User',
      onCancel: handleCancel,
      onOk: handleOk,
      title: <h5>{user ? MODAL_HEADER_LABEL_EDIT : MODAL_HEADER_LABEL_CREATE}</h5>,
    });
  }, [ form, handleCancel, handleOk, openOrUpdate, info, user ]);

  return { modalOpen, ...modalHook };
};

export default useModalCreateUser;
