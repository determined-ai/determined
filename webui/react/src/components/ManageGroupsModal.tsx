import { Select } from 'antd';
import React, { useEffect } from 'react';

import Form from 'components/kit/Form';
import { Modal } from 'components/kit/Modal';
import Spinner from 'components/kit/Spinner';
import { updateGroup } from 'services/api';
import { V1GroupSearchResult } from 'services/api-ts-sdk';
import determinedStore from 'stores/determinedInfo';
import { DetailedUser } from 'types';
import { message } from 'utils/dialogApi';
import { ErrorType } from 'utils/error';
import handleError from 'utils/error';
import { useObservable } from 'utils/observable';

const GROUPS_NAME = 'groups';

interface Props {
  groupOptions: V1GroupSearchResult[];
  user: DetailedUser;
  userGroups: V1GroupSearchResult[];
}

interface FormInputs {
  [GROUPS_NAME]: number[];
}

const ManageGroupsModalComponent: React.FC<Props> = ({ user, groupOptions, userGroups }: Props) => {
  const [form] = Form.useForm<FormInputs>();

  const groupsValue = Form.useWatch(GROUPS_NAME, form);

  const { rbacEnabled } = useObservable(determinedStore.info);

  useEffect(() => {
    if (userGroups) {
      form.setFieldsValue({
        [GROUPS_NAME]: userGroups?.map((ug) => ug.group.groupId),
      });
    }
  }, [form, userGroups]);

  const handleSubmit = async () => {
    await form.validateFields();

    const formData = form.getFieldsValue();
    const userGroupIds = userGroups.map((ug) => ug.group.groupId);

    try {
      if (user) {
        const uid = user.id;
        if (uid) {
          (formData[GROUPS_NAME] as number[]).forEach(async (gid) => {
            if (!userGroupIds?.includes(gid)) {
              await updateGroup({ addUsers: [uid], groupId: gid });
            }
          });
          (userGroupIds as number[])?.forEach(async (gid) => {
            if (!formData[GROUPS_NAME].includes(gid)) {
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
  };

  if (!rbacEnabled) {
    return null;
  }

  return (
    <Modal
      cancel
      size="small"
      submit={{
        disabled: !groupsValue?.length,
        handleError,
        handler: handleSubmit,
        text: 'Save',
      }}
      title="Manage Groups">
      <Spinner spinning={!groupOptions}>
        <Form form={form}>
          <Form.Item name={GROUPS_NAME}>
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
    </Modal>
  );
};

export default ManageGroupsModalComponent;
