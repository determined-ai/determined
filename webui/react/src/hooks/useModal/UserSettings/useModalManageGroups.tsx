import { Form, message, Select } from 'antd';
import { FormInstance } from 'antd/lib/form/hooks/useForm';
import React, { useCallback, useEffect, useState } from 'react';

import useFeature from 'hooks/useFeature';
import { getGroups, updateGroup } from 'services/api';
import { V1GroupSearchResult } from 'services/api-ts-sdk';
import Spinner from 'shared/components/Spinner';
import useModal, { ModalHooks as Hooks } from 'shared/hooks/useModal/useModal';
import { ErrorType } from 'shared/utils/error';
import { DetailedUser } from 'types';
import handleError from 'utils/error';

const FIELD_NAME = 'groups';

interface Props {
  form: FormInstance;
  groupOptions: V1GroupSearchResult[];
  userGroupIds?: (number | undefined)[];
}

const ModalForm: React.FC<Props> = ({ form, groupOptions, userGroupIds }) => {
  const rbacEnabled = useFeature().isOn('rbac');

  useEffect(() => {
    if (userGroupIds) {
      form.setFieldsValue({
        [FIELD_NAME]: userGroupIds,
      });
    }
  }, [form, userGroupIds]);

  if (!rbacEnabled) {
    return null;
  }

  return (
    <Spinner spinning={!groupOptions}>
      <Form form={form} labelCol={{ span: 24 }}>
        <Form.Item name={FIELD_NAME}>
          <Select
            mode="multiple"
            optionFilterProp="children"
            placeholder="Select Groups"
            showSearch>
            {groupOptions.map((go) => (
              <Select.Option key={go.group.groupId} value={go.group.groupId}>
                {go.group.name}
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

const useModalManageGroups = ({ groups, user }: ModalProps): ModalHooks => {
  const [form] = Form.useForm();

  const { modalOpen: openOrUpdate, ...modalHook } = useModal();

  const handleOk = useCallback(async (userGroupIds?: (number | undefined)[]) => {
    await form.validateFields();

    const formData = form.getFieldsValue();

    try {
      if (user) {
        const uid = user?.id;
        if (uid) {
          (formData[FIELD_NAME] as number[]).forEach(async (gid) => {
            if (!userGroupIds?.includes(gid)) {
              await updateGroup({ addUsers: [uid], groupId: gid });
            }
          });
          (userGroupIds as number[])?.forEach(async (gid) => {
            if (!formData[FIELD_NAME].includes(gid)) {
              await updateGroup({ groupId: gid, removeUsers: [uid] });
            }
          });
        }
      }
    } catch (e) {
      message.error('Error adding user to groups');
      handleError(e, { silent: true, type: ErrorType.Input });

      // Re-throw error to prevent modal from getting dismissed.
      throw e;
    }
  }, [form, user]);

  const fetchUserGroups = useCallback(async () => {
    try {
      const response = await getGroups({ userId: user.id });
      const groupIds = response.groups?.map((ug) => ug.group.groupId);
      return groupIds;
    } catch (e) {
      handleError(e, { publicSubject: "Unable to fetch this user's groups." });
    }
  }, [user.id]);

  const modalOpen = useCallback(async () => {
    const userGroupIds = await fetchUserGroups();
    openOrUpdate({
      closable: true,
      content: <ModalForm form={form} groupOptions={groups} userGroupIds={userGroupIds} />,
      icon: null,
      okText: 'Save',
      onOk: () => handleOk(userGroupIds),
      title: <h5>Manage Groups</h5>,
      width: 520,
    });
  }, [form,
    handleOk,
    openOrUpdate,
    fetchUserGroups,
    groups]);
  return { modalOpen, ...modalHook };
};

export default useModalManageGroups;
