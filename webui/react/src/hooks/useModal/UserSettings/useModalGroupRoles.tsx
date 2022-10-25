import { Form, message, Select, Table } from 'antd';
import { FormInstance } from 'antd/lib/form/hooks/useForm';
import React, { useCallback, useEffect, useState } from 'react';

import { useStore } from 'contexts/Store';
import { assignRolesToGroup, getGroupRoles } from 'services/api';
import { V1GroupSearchResult } from 'services/api-ts-sdk';
import useModal, { ModalHooks } from 'shared/hooks/useModal/useModal';
import { ErrorType } from 'shared/utils/error';
import { UserRole } from 'types';
import handleError from 'utils/error';

interface Props {
  form: FormInstance;
  group: V1GroupSearchResult;
}

const ModalForm: React.FC<Props> = ({ form, group }) => {
  const [groupRoles, setGroupRoles] = useState<UserRole[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const { knownRoles } = useStore();

  const fetchGroupRoles = useCallback(async () => {
    if (group?.group.groupId) {
      try {
        const roles = await getGroupRoles({ groupId: group.group.groupId });
        setGroupRoles(roles);
      } catch (e) {
        handleError(e, { publicSubject: "Unable to fetch this group's roles." });
      } finally {
        setIsLoading(false);
      }
    } else {
      setIsLoading(false);
    }
  }, [group]);
  useEffect(() => {
    fetchGroupRoles();
  }, [fetchGroupRoles]);

  const roleColumns = [
    {
      dataIndex: 'name',
      key: 'name',
      title: 'Role Name',
    },
  ];

  return (
    <Form form={form} labelCol={{ span: 8 }} wrapperCol={{ span: 14 }}>
      {groupRoles.length ? (
        <Table
          columns={roleColumns}
          dataSource={groupRoles}
          pagination={{ hideOnSinglePage: true, size: 'small' }}
        />
      ) : null}
      <Form.Item label="Roles" name="roles">
        <Select
          loading={isLoading}
          mode="multiple"
          optionFilterProp="children"
          placeholder={`Add Roles to: ${group.group.name}`}
          showSearch>
          {knownRoles.map((r) => (
            <Select.Option key={r.id} value={r.id}>
              {r.name}
            </Select.Option>
          ))}
        </Select>
      </Form.Item>
    </Form>
  );
};

interface ModalProps {
  group: V1GroupSearchResult;
  onClose?: () => void;
}

const useModalGroupRoles = ({ onClose, group }: ModalProps): ModalHooks => {
  const [form] = Form.useForm();

  const { modalOpen: openOrUpdate, ...modalHook } = useModal();

  const handleCancel = useCallback(() => {
    form.resetFields();
  }, [form]);

  const onOk = useCallback(async () => {
    if (!group.group.groupId) {
      return;
    }
    await form.validateFields();

    try {
      const formData = form.getFieldsValue();
      await assignRolesToGroup({
        groupId: group.group.groupId,
        roleIds: formData.roles,
      });
      message.success('Updated group roles.');
      form.resetFields();
      onClose?.();
    } catch (e) {
      message.error('Error updating group roles.');
      handleError(e, { silent: true, type: ErrorType.Input });

      // Re-throw error to prevent modal from getting dismissed.
      throw e;
    }
  }, [form, onClose, group]);

  const modalOpen = useCallback(() => {
    openOrUpdate({
      closable: true,
      content: <ModalForm form={form} group={group} />,
      icon: null,
      okText: 'Update Roles',
      onCancel: handleCancel,
      onOk: onOk,
      title: <h5>Update Roles</h5>,
    });
  }, [form, handleCancel, onOk, openOrUpdate, group]);

  return { modalOpen, ...modalHook };
};

export default useModalGroupRoles;
