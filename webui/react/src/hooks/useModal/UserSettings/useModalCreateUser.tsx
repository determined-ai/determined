import { Form, Input, message, Switch } from 'antd';
import { FormInstance } from 'antd/lib/form/hooks/useForm';
import React, { useCallback, useEffect, useState } from 'react';

import { useStore } from 'contexts/Store';
import { getUser, patchUser, postUser } from 'services/api';
import useModal, { ModalHooks } from 'shared/hooks/useModal/useModal';
import { ErrorType } from 'shared/utils/error';
import { BrandingType, DetailedUser } from 'types';
import handleError from 'utils/error';

export const MODAL_HEADER_LABEL_CREATE = 'Create User';
export const MODAL_HEADER_LABEL_EDIT = 'Edit User';
export const USER_NAME_NAME = 'username';
export const USER_NAME_LABEL = 'User Name';
export const DISPLAY_NAME_NAME = 'displayName';
export const DISPLAY_NAME_LABEL = 'Display Name';
export const ADMIN_NAME = 'admin';
export const ADMIN_LABEL = 'Admin';
export const API_SUCCESS_MESSAGE_CREATE = `New user with empty password has been created, 
advice user to reset password as soon as possible.`;
export const API_SUCCESS_MESSAGE_EDIT = 'User has been updated';
interface Props {
  branding: BrandingType;
  form: FormInstance;
  user?: DetailedUser
}

const ModalForm: React.FC<Props> = ({ form, branding, user }) => {
  useEffect(() => {
    form.setFieldsValue({
      [ADMIN_NAME]: user?.isAdmin,
      [DISPLAY_NAME_NAME]: user?.displayName,
    });
  }, [ user, form ]);
  return (
    <Form
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
        <Input autoFocus disabled={!!user} maxLength={128} placeholder="user name" />
      </Form.Item>
      <Form.Item
        label={DISPLAY_NAME_LABEL}
        name={DISPLAY_NAME_NAME}>
        <Input maxLength={128} placeholder="display name" />
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
  // const [ user, setUser ] = useState<DetailedUser>()
  const [ form ] = Form.useForm();
  const { info } = useStore();
  const [ updatedUser, setUpdatedUser ] = useState<DetailedUser>();
  const { modalOpen: openOrUpdate, ...modalHook } = useModal();

  const fetchUser = useCallback(async () => {
    if (!user) return;
    const res = await getUser({ userId: user.id });
    setUpdatedUser(res);
  }, [ user ]);

  const handleCancel = useCallback(() => {
    form.resetFields();
    fetchUser();
  }, [ form, fetchUser ]);

  const handleOkay = useCallback(async () => {
    await form.validateFields();

    const formData = form.getFieldsValue();
    formData.admin = !!formData.admin;
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
      fetchUser();
    } catch (e) {
      message.error(user ? 'error updating user' : 'error creating new user');
      handleError(e, { silent: true, type: ErrorType.Input });

      // Re-throw error to prevent modal from getting dismissed.
      throw e;
    }
  }, [ form, onClose, fetchUser, user ]);

  const modalOpen = useCallback(() => {
    openOrUpdate({
      closable: true,
      content: <ModalForm branding={info.branding} form={form} user={updatedUser || user} />,
      icon: null,
      okText: user ? 'Update' : 'Create User',
      onCancel: handleCancel,
      onOk: handleOkay,
      title: <h5>{user ? MODAL_HEADER_LABEL_EDIT : MODAL_HEADER_LABEL_CREATE}</h5>,
    });
  }, [ form, handleCancel, handleOkay, openOrUpdate, info, updatedUser, user ]);

  return { modalOpen, ...modalHook };
};

export default useModalCreateUser;
