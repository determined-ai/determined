import { Typography } from 'antd';
import { filter } from 'fp-ts/lib/Set';
import Form from 'hew/Form';
import Input from 'hew/Input';
import { Modal } from 'hew/Modal';
import Select, { Option } from 'hew/Select';
import Spinner from 'hew/Spinner';
import { useToast } from 'hew/Toast';
import { Loadable } from 'hew/utils/loadable';
import _ from 'lodash';
import { useObservable } from 'micro-observables';
import React, { useCallback, useEffect, useId, useState } from 'react';

import Link from 'components/Link';
import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';
import {
  assignRolesToGroup,
  createGroup,
  getGroup,
  getGroupRoles,
  removeRolesFromGroup,
  updateGroup,
} from 'services/api';
import { V1GroupDetails, V1GroupSearchResult } from 'services/api-ts-sdk';
import determinedStore from 'stores/determinedInfo';
import roleStore from 'stores/roles';
import { UserRole } from 'types';
import handleError, { ErrorType } from 'utils/error';

export const MODAL_HEADER_LABEL_CREATE = 'Create Group';
export const MODAL_HEADER_LABEL_EDIT = 'Edit Group';
const GROUP_NAME_NAME = 'name';
export const GROUP_NAME_LABEL = 'Group Name';
const GROUP_ROLE_NAME = 'roles';
export const GROUP_ROLE_LABEL = 'Select Global Roles';
export const API_SUCCESS_MESSAGE_CREATE = 'New group has been created.';
const API_SUCCESS_MESSAGE_EDIT = 'Group has been updated.';
const API_FAILURE_MESSAGE_CREATE = 'Error creating new group.';
const API_FAILURE_MESSAGE_EDIT = 'Error editing group.';
const FORM_ID = 'create-group-form';

interface Messages {
  API_FAILURE_MESSAGE: string;
  API_SUCCESS_MESSAGE: string;
  MODAL_HEADER_LABEL: string;
}

const CREATE_VALUES: Messages = {
  API_FAILURE_MESSAGE: API_FAILURE_MESSAGE_CREATE,
  API_SUCCESS_MESSAGE: API_SUCCESS_MESSAGE_CREATE,
  MODAL_HEADER_LABEL: MODAL_HEADER_LABEL_CREATE,
};

const EDIT_VALUES: Messages = {
  API_FAILURE_MESSAGE: API_FAILURE_MESSAGE_EDIT,
  API_SUCCESS_MESSAGE: API_SUCCESS_MESSAGE_EDIT,
  MODAL_HEADER_LABEL: MODAL_HEADER_LABEL_EDIT,
};

interface Props {
  group?: V1GroupSearchResult;
  onClose?: () => void;
}

const CreateGroupModalComponent: React.FC<Props> = ({ onClose, group }: Props) => {
  const idPrefix = useId();
  const [form] = Form.useForm();
  const { rbacEnabled } = useObservable(determinedStore.info);
  const { canModifyPermissions } = usePermissions();
  const [groupRoles, setGroupRoles] = useState<UserRole[]>([]);
  const [groupDetail, setGroupDetail] = useState<V1GroupDetails>();
  const [isLoading, setIsLoading] = useState(true);
  const isCreateModal = !group;
  const messages = isCreateModal ? CREATE_VALUES : EDIT_VALUES;

  const { openToast } = useToast();

  const roles = useObservable(roleStore.roles);
  const groupName = Form.useWatch(GROUP_NAME_NAME, form);

  const fetchGroupDetail = useCallback(async () => {
    if (group?.group.groupId) {
      try {
        const response = await getGroup({ groupId: group?.group.groupId });
        const groupDetail = response.group;
        setGroupDetail(groupDetail);
        form.setFieldsValue({
          [GROUP_NAME_NAME]: groupDetail.name,
        });
      } catch (e) {
        handleError(e, { publicSubject: 'Unable to fetch group data.' });
      }
    }
  }, [group, form]);

  const fetchGroupRoles = useCallback(async () => {
    if (group?.group.groupId && rbacEnabled) {
      try {
        const roles = await getGroupRoles({ groupId: group.group.groupId });
        const groupRoles = roles.filter((r) => r.scopeCluster);
        setGroupRoles(groupRoles);
        form.setFieldValue(GROUP_ROLE_NAME, groupRoles?.map((r) => r.id));
      } catch (e) {
        handleError(e, { publicSubject: "Unable to fetch this group's roles." });
      }
    }
  }, [form, group, rbacEnabled]);

  const fetchData = useCallback(async () => {
    await fetchGroupDetail();
    await fetchGroupRoles();
    setIsLoading(false);
  }, [fetchGroupDetail, fetchGroupRoles]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleSubmit = async () => {
    try {
      const formData = await form.validateFields();

      if (group) {
        const nameUpdated = !_.isEqual(formData.name, groupDetail?.name);
        const rolesUpdated = !_.isEqual(
          formData.roles,
          groupRoles.map((r) => r.id),
        );
        if (!nameUpdated && !rolesUpdated) {
          openToast({ title: 'No changes to save.' });
          return;
        }

        await updateGroup({ groupId: group.group.groupId, ...formData });
        if (canModifyPermissions && group.group.groupId) {
          const newRoles: Set<number> = new Set(formData.roles);
          const oldRoles = new Set((groupRoles ?? []).map((r) => r.id));

          const rolesToAdd = filter((r: number) => !oldRoles.has(r))(newRoles);
          const rolesToRemove = filter((r: number) => !newRoles.has(r))(oldRoles);

          if (rolesToAdd.size > 0) {
            await assignRolesToGroup([
              {
                groupId: group.group.groupId,
                roleIds: Array.from(rolesToAdd),
              },
            ]);
          }
          if (rolesToRemove.size > 0) {
            await removeRolesFromGroup({
              groupId: group.group.groupId,
              roleIds: Array.from(rolesToRemove),
            });
          }
          await fetchGroupRoles();
        }
      } else {
        const newGroup = await createGroup(formData);
        if (canModifyPermissions && newGroup.group.groupId) {
          const newRoles: Array<number> = formData.roles ?? [];

          if (newRoles.length > 0) {
            await assignRolesToGroup([
              {
                groupId: newGroup.group.groupId,
                roleIds: newRoles,
              },
            ]);
          }
        }
      }
      openToast({ severity: 'Confirm', title: messages.API_SUCCESS_MESSAGE });
      form.resetFields();
      onClose?.();
    } catch (e) {
      openToast({ severity: 'Error', title: messages.API_FAILURE_MESSAGE });
      handleError(e, { silent: true, type: ErrorType.Input });

      // Re-throw error to prevent modal from getting dismissed.
      throw e;
    }
  };

  return (
    <Modal
      cancel
      size="small"
      submit={{
        disabled: !groupName,
        form: idPrefix + FORM_ID,
        handleError,
        handler: handleSubmit,
        text: messages.MODAL_HEADER_LABEL,
      }}
      title={messages.MODAL_HEADER_LABEL}
      onClose={form.resetFields}>
      <Spinner spinning={isLoading}>
        <Form form={form} id={idPrefix + FORM_ID}>
          <Form.Item
            label={GROUP_NAME_LABEL}
            name={GROUP_NAME_NAME}
            required
            rules={[{ whitespace: true }]}
            validateTrigger={['onSubmit', 'onChange']}>
            <Input autoComplete="off" autoFocus maxLength={128} placeholder={GROUP_NAME_LABEL} />
          </Form.Item>
          {rbacEnabled && canModifyPermissions && (
            <>
              <Form.Item label={GROUP_ROLE_LABEL} name={GROUP_ROLE_NAME}>
                <Select
                  loading={Loadable.isNotLoaded(roles)}
                  mode="multiple"
                  placeholder={'Add Roles'}>
                  {roles
                    .getOrElse([])
                    .sort((r1, r2) => r1.id - r2.id)
                    .map((r) => (
                      <Option key={r.id} value={r.id}>
                        {r.name}
                      </Option>
                    ))}
                </Select>
              </Form.Item>
              <Typography.Text type="secondary">
                Groups may have additional inherited workspace roles not reflected here. &nbsp;
                <Link external path={paths.docs('/cluster-setup-guide/security/rbac.html')} popout>
                  Learn more
                </Link>
              </Typography.Text>
            </>
          )}
        </Form>
      </Spinner>
    </Modal>
  );
};

export default CreateGroupModalComponent;
