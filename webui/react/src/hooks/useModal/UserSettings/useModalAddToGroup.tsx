import { Form, message, Select } from 'antd';
import { FormInstance } from 'antd/lib/form/hooks/useForm';
import React, { useCallback } from 'react';

import useFeature from 'hooks/useFeature';
import {
  updateGroup,
} from 'services/api';
import { V1GroupSearchResult } from 'services/api-ts-sdk';
import Spinner from 'shared/components/Spinner';
import useModal, { ModalHooks as Hooks } from 'shared/hooks/useModal/useModal';
import { ErrorType } from 'shared/utils/error';
import { DetailedUser } from 'types';
import handleError from 'utils/error';

const FIELD_NAME = 'groups';

interface Props {
  form: FormInstance;
  groups: V1GroupSearchResult[];
}

const ModalForm: React.FC<Props> = ({ form, groups }) => {
  const rbacEnabled = useFeature().isOn('rbac');

  if (!rbacEnabled) {
    return null;
  }

  return (
    <Spinner spinning={!groups}>
      <Form form={form} labelCol={{ span: 24 }}>
        <Form.Item name={FIELD_NAME}>
          <Select
            mode="multiple"
            optionFilterProp="children"
            placeholder="Select Groups"
            showSearch>
            {groups.map((u) => (
              <Select.Option key={u.group.groupId} value={u.group.groupId}>
                {u.group.name}
              </Select.Option>
            ))}
          </Select>
        </Form.Item>
      </Form>
    </Spinner>
  );
};

interface ModalProps {
  groups: V1GroupSearchResult[];
  user: DetailedUser;
}

interface ModalHooks extends Omit<Hooks, 'modalOpen'> {
  modalOpen: () => void;
}

const useModalCreateUser = ({ groups, user }: ModalProps): ModalHooks => {
  const [form] = Form.useForm();

  const { modalOpen: openOrUpdate, ...modalHook } = useModal();

  const handleCancel = useCallback(() => {
    form.resetFields();
  }, [form]);

  const handleOk = useCallback(
    async () => {
      await form.validateFields();

      const formData = form.getFieldsValue();

      try {
        if (user) {
          const uid = user?.id;
          if (uid && formData[FIELD_NAME]) {
            (formData[FIELD_NAME] as number[]).forEach(async (gid) => {
              await updateGroup({ addUsers: [uid], groupId: gid });
            });
          }
          form.resetFields();
        }
      } catch (e) {
        message.error('Error adding user to groups');
        handleError(e, { silent: true, type: ErrorType.Input });

        // Re-throw error to prevent modal from getting dismissed.
        throw e;
      }
    },
    [
      form,
      user,
    ],
  );

  const modalOpen = useCallback(
    () => {
      openOrUpdate({
        closable: true,
        content: (
          <ModalForm
            form={form}
            groups={groups}
          />
        ),
        icon: null,
        okText: 'Save',
        onCancel: handleCancel,
        onOk: () => handleOk(),
        title: (
          <h5>Add to Groups</h5>
        ),
        width: 520,
      });
    },
    [form, handleCancel, handleOk, openOrUpdate, groups],
  );
  return { modalOpen, ...modalHook };
};

export default useModalCreateUser;
