import Form from 'hew/Form';
import { Modal } from 'hew/Modal';
import Select, { Option } from 'hew/Select';
import Spinner from 'hew/Spinner';
import { useToast } from 'hew/Toast';
import React, { useEffect, useId } from 'react';

import { updateGroup } from 'services/api';
import { V1GroupSearchResult } from 'services/api-ts-sdk';
import determinedStore from 'stores/determinedInfo';
import { DetailedUser } from 'types';
import handleError, { ErrorType } from 'utils/error';
import { useObservable } from 'utils/observable';

const GROUPS_NAME = 'groups';
const FORM_ID = 'manage-groups-form';

interface Props {
  groupOptions: V1GroupSearchResult[];
  user: DetailedUser;
  userGroups: V1GroupSearchResult[];
}

interface FormInputs {
  [GROUPS_NAME]: number[];
}

const ManageGroupsModalComponent: React.FC<Props> = ({ user, groupOptions, userGroups }: Props) => {
  const idPrefix = useId();
  const [form] = Form.useForm<FormInputs>();

  const { openToast } = useToast();

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
    const formData = await form.validateFields();
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
      openToast({ severity: 'Error', title: 'Error adding user to groups' });
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
        form: idPrefix + FORM_ID,
        handleError,
        handler: handleSubmit,
        text: 'Save',
      }}
      title="Manage Groups">
      <Spinner spinning={!groupOptions}>
        <Form form={form} id={idPrefix + FORM_ID}>
          <Form.Item name={GROUPS_NAME}>
            <Select mode="multiple" placeholder="Select Groups">
              {groupOptions.map((go) => (
                <Option key={go.group.groupId} value={go.group.groupId}>
                  {go.group.name}
                </Option>
              ))}
            </Select>
          </Form.Item>
        </Form>
      </Spinner>
    </Modal>
  );
};

export default ManageGroupsModalComponent;
