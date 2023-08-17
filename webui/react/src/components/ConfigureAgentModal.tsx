import React, { useEffect, useState } from 'react';

import Form from 'components/kit/Form';
import Input from 'components/kit/Input';
import InputNumber from 'components/kit/InputNumber';
import { Modal } from 'components/kit/Modal';
import Spinner from 'components/kit/Spinner';
import { patchUser } from 'services/api';
import { V1AgentUserGroup } from 'services/api-ts-sdk';
import { DetailedUser } from 'types';
import { message } from 'utils/dialogApi';
import handleError, { ErrorType } from 'utils/error';

interface Props {
  user: DetailedUser;
  onClose?: () => void;
}

const requiredFields = ['agentUid', 'agentUser', 'agentGid', 'agentGroup'];

const ConfigureAgentModalComponent: React.FC<Props> = ({ user, onClose }: Props) => {
  const [form] = Form.useForm();
  const [disabled, setDisabled] = useState<boolean>(true);

  const handleFieldsChange = () => {
    const values = form.getFieldsValue();
    const missingRequiredFields = requiredFields.map((rf) => values[rf]).some((v) => !v);
    setDisabled(missingRequiredFields);
  };

  const handleSubmit = async () => {
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
  };

  useEffect(() => {
    if (user.agentUserGroup) {
      // validate initial values, before onFieldsChange
      const missingRequiredFields = Object.entries(user.agentUserGroup).some(([key, value]) => {
        return requiredFields.includes(key) && !value;
      });
      setDisabled(missingRequiredFields);
    }
  }, [user]);

  return (
    <Modal
      cancel
      size="small"
      submit={{
        disabled,
        handleError,
        handler: handleSubmit,
        text: 'Save',
      }}
      title="Configure Agent"
      onClose={form.resetFields}>
      <Spinner spinning={!user}>
        <Form
          form={form}
          initialValues={
            user?.agentUserGroup
              ? {
                  agentGid: user?.agentUserGroup.agentGid,
                  agentGroup: user?.agentUserGroup.agentGroup,
                  agentUid: user?.agentUserGroup.agentUid,
                  agentUser: user?.agentUserGroup.agentUser,
                }
              : {
                  agentGid: undefined,
                  agentGroup: undefined,
                  agentUid: undefined,
                  agentUser: undefined,
                }
          }
          onFieldsChange={handleFieldsChange}>
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
