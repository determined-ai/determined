import React, { useCallback, useEffect, useState } from 'react';

import Form from 'components/kit/Form';
import Input from 'components/kit/Input';
import InputNumber from 'components/kit/InputNumber';
import { Modal } from 'components/kit/Modal';
import { patchUser } from 'services/api';
import { V1AgentUserGroup } from 'services/api-ts-sdk';
import Spinner from 'shared/components/Spinner';
import { ErrorType } from 'shared/utils/error';
import { DetailedUser } from 'types';
import { message } from 'utils/dialogApi';
import handleError from 'utils/error';

interface Props {
  user: DetailedUser;
  onClose?: () => void;
}

const requiredFields = ['agentUid', 'agentUser', 'agentGid', 'agentGroup'];

const ConfigureAgentModalComponent: React.FC<Props> = ({ user, onClose }: Props) => {
  const [form] = Form.useForm();
  const [disabled, setDisabled] = useState<boolean>(true);

  const handleFieldsChange = useCallback(() => {
    const values = form.getFieldsValue();
    const missingRequiredFields = requiredFields.map((rf) => values[rf]).some((v) => !v);
    setDisabled(missingRequiredFields);
  }, [form, setDisabled]);

  const handleCancel = useCallback(() => {
    form.resetFields();
  }, [form]);

  const handleSubmit = useCallback(async () => {
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
    <Modal
      cancel
      size="small"
      submit={{
        disabled,
        handler: handleSubmit,
        text: 'Save',
      }}
      title="Configure Agent"
      onClose={handleCancel}>
      <Spinner spinning={!user}>
        <Form form={form} labelCol={{ span: 24 }} onFieldsChange={handleFieldsChange}>
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
    </Modal>
  );
};

export default ConfigureAgentModalComponent;
