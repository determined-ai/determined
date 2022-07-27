import { Form, Input, message, Switch } from 'antd';
import { FormInstance } from 'antd/lib/form/hooks/useForm';
import React, { useCallback } from 'react';

import { useStore } from 'contexts/Store';
import { login, postUser } from 'services/api';
import Icon from 'shared/components/Icon/Icon';
import useModal, { ModalHooks } from 'shared/hooks/useModal/useModal';
import { ErrorType } from 'shared/utils/error';
import { BrandingType } from 'types';
import handleError from 'utils/error';

interface Props {
  form: FormInstance;
  branding: BrandingType;
}

const ModalForm: React.FC<Props> = ({ form, branding }) => (
  <Form
    form={form}
    labelCol={{ span: 8 }}
    wrapperCol={{ span: 14 }}>
    <Form.Item
      label="User Name"
      name="username"
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
      label="Display Name"
      name="displayName">
      <Input maxLength={128} placeholder="display name" />
    </Form.Item>
    {branding === BrandingType.Determined ? (
      <Form.Item
        label="Admin"
        name="admin">
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
    form.resetFields()
}, [ form ]);

  const handleOkay = useCallback(async () => {
    await form.validateFields();

    try {
      const formData = form.getFieldsValue();
      formData.admin = !!formData.admin
        await postUser(formData);
      message.success('New user with empty password has been created, advice user to reset password as soon as possible.');
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
      content: <ModalForm branding={info.branding} form={form} />,
      icon: null,
      okText: 'Create User',
      onCancel: handleCancel,
      onOk: handleOkay,
      title: <h5>Create User</h5>,
    });
  }, [ form, handleCancel, handleOkay, openOrUpdate ]);

  return { modalOpen, ...modalHook };
};

export default useModalCreateUser;
