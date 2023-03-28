import { Select } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';

import Form from 'components/kit/Form';
import { Modal } from 'components/kit/Modal';
import useFeature from 'hooks/useFeature';
import { getGroups, updateGroup } from 'services/api';
import { V1GroupSearchResult } from 'services/api-ts-sdk';
import Spinner from 'shared/components/Spinner';
import { ErrorType } from 'shared/utils/error';
import { DetailedUser } from 'types';
import { message } from 'utils/dialogApi';
import handleError from 'utils/error';

const FIELD_NAME = 'groups';

interface Props {
  groups: V1GroupSearchResult[];
  user: DetailedUser;
}

const ManageGroupsModalComponent: React.FC<Props> = ({ user, groups }: Props) => {
  const [form] = Form.useForm();

  const [userGroupIds, setUserGroupIds] = useState<(number | undefined)[]>();

  const fetchUserGroups = useCallback(async () => {
    try {
      const response = await getGroups({ userId: user.id });
      const groupIds = response.groups?.map((ug) => ug.group.groupId);
      if (groupIds?.length) setUserGroupIds(groupIds);
    } catch (e) {
      handleError(e, { publicSubject: "Unable to fetch this user's groups." });
    }
  }, [user.id]);

  useEffect(() => {
    fetchUserGroups();
  }, [fetchUserGroups]);

  const rbacEnabled = useFeature().isOn('rbac');

  useEffect(() => {
    if (userGroupIds) {
      form.setFieldsValue({
        [FIELD_NAME]: userGroupIds,
      });
    }
  }, [form, userGroupIds]);

  const handleSubmit = useCallback(
    async (userGroupIds?: (number | undefined)[]) => {
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
    },
    [form, user],
  );

  if (!rbacEnabled) {
    return null;
  }

  return (
    <Modal
      cancel
      submit={{
        handler: handleSubmit,
        text: 'Save',
      }}
      title="Manage Groups">
      <Spinner spinning={!groups}>
        <Form form={form} labelCol={{ span: 24 }}>
          <Form.Item name={FIELD_NAME}>
            <Select
              mode="multiple"
              optionFilterProp="children"
              placeholder="Select Groups"
              showSearch>
              {groups.map((go) => (
                <Select.Option key={go.group.groupId} value={go.group.groupId}>
                  {go.group.name}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
        </Form>
      </Spinner>
    </Modal>
  );
};

export default ManageGroupsModalComponent;
