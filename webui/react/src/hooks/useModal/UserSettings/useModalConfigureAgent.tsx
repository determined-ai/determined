import React, { useCallback, useEffect } from 'react';

import Form, { FormInstance } from 'components/kit/Form';
import Input from 'components/kit/Input';
import InputNumber from 'components/kit/InputNumber';
import { patchUser } from 'services/api';
import { V1AgentUserGroup } from 'services/api-ts-sdk';
import Spinner from 'shared/components/Spinner';
import useModal, { ModalHooks as Hooks } from 'shared/hooks/useModal/useModal';
import { ErrorType } from 'shared/utils/error';
import { DetailedUser } from 'types';
import { message } from 'utils/dialogApi';
import handleError from 'utils/error';

interface Props {
  form: FormInstance;
  user: DetailedUser;
}

const ModalForm: React.FC<Props> = ({ form, user }) => {
  useEffect(() => {
    if (user?.agentUserGroup) {
      form.setFieldsValue({
        agentGid: user?.agentUserGroup.agentGid,
        agentGroup: user?.agentUserGroup.agentGroup,
        agentUid: user?.agentUserGroup.agentUid,
        agentUser: user?.agentUserGroup.agentUser,
      });
    } else {
      form.setFieldsValue({
        agentGid: undefined,
        agentGroup: undefined,
        agentUid: undefined,
        agentUser: undefined,
      });
    }
  }, [form, user]);

  return (
    <Spinner spinning={!user}>
      <Form form={form} labelCol={{ span: 24 }}>
        <Form.Item
          label="Agent User ID"
          name="agentUid"
          rules={[{ message: 'Agent User ID is required ', required: true }]}>
          <InputNumber />
        </Form.Item>
        <Form.Item
          label="Agent User Name"
          name="agentUser"
          rules={[{ message: 'Agent User Name is required ', required: true }]}>
          <Input maxLength={100} />
        </Form.Item>
        <Form.Item
          label="Agent User Group ID"
          name="agentGid"
          rules={[{ message: 'Agent User Group ID is required ', required: true }]}>
          <InputNumber />
        </Form.Item>
        <Form.Item
          label="Agent Group Name"
          name="agentGroup"
          rules={[{ message: 'Agent Group Name is required ', required: true }]}>
          <Input maxLength={100} />
        </Form.Item>
      </Form>
    </Spinner>
  );
};

interface ModalProps {
  onClose?: () => void;
  user: DetailedUser;
}

interface ModalHooks extends Omit<Hooks, 'modalOpen'> {
  modalOpen: () => void;
}

const useModalConfigureAgent = ({ user, onClose }: ModalProps): ModalHooks => {
  const [form] = Form.useForm();

  const { modalOpen: openOrUpdate, ...modalHook } = useModal();

  const handleCancel = useCallback(() => {
    form.resetFields();
  }, [form]);

  const handleOk = useCallback(async () => {
    await form.validateFields();

    const formData = form.getFieldsValue();
    const { agentUid, agentUser, agentGid, agentGroup } = formData;
    const agentUserGroup: V1AgentUserGroup = { agentGid, agentGroup, agentUid, agentUser };
    formData.agentUserGroup = agentUserGroup;

    try {
      await patchUser({ userId: user.id, userParams: formData });
      onClose?.();
    } catch (e) {
      message.error('Error configuring agent');
      handleError(e, { silent: true, type: ErrorType.Input });

      // Re-throw error to prevent modal from getting dismissed.
      throw e;
    }
  }, [form, user, onClose]);

  const modalOpen = useCallback(() => {
    openOrUpdate({
      closable: true,
      content: <ModalForm form={form} user={user} />,
      icon: null,
      okText: 'Save',
      onCancel: handleCancel,
      onOk: () => handleOk(),
      title: <h5>Configure Agent</h5>,
      width: 520,
    });
  }, [form, handleCancel, handleOk, openOrUpdate, user]);
  return { modalOpen, ...modalHook };
};

export default useModalConfigureAgent;
