import { Select, Typography } from 'antd';
import { filter } from 'fp-ts/lib/Set';
import _ from 'lodash';
import { useObservable } from 'micro-observables';
import React, { useCallback, useEffect, useId, useState } from 'react';

import Form from 'components/kit/Form';
import Input from 'components/kit/Input';
import { Modal } from 'components/kit/Modal';
import Spinner from 'components/kit/Spinner';
import { makeToast } from 'components/kit/Toast';
import { Loadable } from 'components/kit/utils/loadable';
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
import { DetailedUser, UserRole } from 'types';
import handleError, { ErrorType } from 'utils/error';
import { getDisplayName } from 'utils/user';

export const MODAL_HEADER_LABEL_CREATE = 'Create Group';
export const MODAL_HEADER_LABEL_EDIT = 'Edit Group';
const GROUP_NAME_NAME = 'name';
export const GROUP_NAME_LABEL = 'Group Name';
const GROUP_ROLE_NAME = 'roles';
const GROUP_ROLE_LABEL = 'Global Roles';
const USERS_NAME = 'users';
export const USERS_LABEL = 'Users';
const ADD_USERS = 'addUsers';
const REMOVE_USERS = 'removeUsers';
export const API_SUCCESS_MESSAGE_CREATE = 'New group has been created.';
const API_SUCCESS_MESSAGE_EDIT = 'Group has been updated.';
const FORM_ID = 'create-group-form';

interface Props {
  group?: V1GroupSearchResult;
  onClose?: () => void;
  users: DetailedUser[];
}

const CreateGroupModalComponent: React.FC<Props> = ({ onClose, users, group }: Props) => {
  const idPrefix = useId();
  const [form] = Form.useForm();
  const { rbacEnabled } = useObservable(determinedStore.info);
  const { canModifyPermissions } = usePermissions();
  const [groupRoles, setGroupRoles] = useState<UserRole[]>([]);
  const [groupDetail, setGroupDetail] = useState<V1GroupDetails>();
  const [isLoading, setIsLoading] = useState(true);

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
          [USERS_NAME]: groupDetail?.users?.map((u) => u.id),
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
        form.setFieldValue(
          GROUP_ROLE_NAME,
          groupRoles?.map((r) => r.id),
        );
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
        const usersUpdated = !_.isEqual(
          formData.users,
          groupDetail?.users?.map((u) => u.id),
        );
        const rolesUpdated = !_.isEqual(
          formData.roles,
          groupRoles.map((r) => r.id),
        );
        if (!nameUpdated && !usersUpdated && !rolesUpdated) {
          makeToast({ title: 'No changes to save.' });
          return;
        }

        const oldUserIds = groupDetail?.users?.map((u) => u.id) ?? [];
        const usersToAdd = formData[USERS_NAME].filter(
          (userId: number) => !oldUserIds.includes(userId),
        );
        const usersToRemove = oldUserIds.filter((userId) => !formData[USERS_NAME].includes(userId));
        formData[ADD_USERS] = usersToAdd;
        formData[REMOVE_USERS] = usersToRemove;

        await updateGroup({ groupId: group.group.groupId, ...formData });
        if (canModifyPermissions && group.group.groupId) {
          const newRoles: Set<number> = new Set(formData.roles);
          const oldRoles = new Set((groupRoles ?? []).map((r) => r.id));

          const rolesToAdd = filter((r: number) => !oldRoles.has(r))(newRoles);
          const rolesToRemove = filter((r: number) => !newRoles.has(r))(oldRoles);

          rolesToAdd.size > 0 &&
            (await assignRolesToGroup({
              groupId: group.group.groupId,
              roleIds: Array.from(rolesToAdd),
            }));
          rolesToRemove.size > 0 &&
            (await removeRolesFromGroup({
              groupId: group.group.groupId,
              roleIds: Array.from(rolesToRemove),
            }));
          await fetchGroupRoles();
        }
        makeToast({ severity: 'Confirm', title: API_SUCCESS_MESSAGE_EDIT });
      } else {
        if (formData[USERS_NAME]) formData[ADD_USERS] = formData[USERS_NAME];
        await createGroup(formData);
        makeToast({ severity: 'Confirm', title: API_SUCCESS_MESSAGE_CREATE });
      }
      form.resetFields();
      onClose?.();
    } catch (e) {
      if (group) {
        makeToast({ severity: 'Error', title: 'Error editing group.' });
      } else {
        makeToast({ severity: 'Error', title: 'Error creating new group.' });
      }
      handleError(e, { silent: true, type: ErrorType.Input });

      // Re-throw error to prevent modal from getting dismissed.
      throw e;
    }
  };

  const currentGroupMembers = form.getFieldValue(USERS_NAME);

  return (
    <Modal
      cancel
      size="small"
      submit={{
        disabled: !groupName,
        form: idPrefix + FORM_ID,
        handleError,
        handler: handleSubmit,
        text: group ? MODAL_HEADER_LABEL_EDIT : MODAL_HEADER_LABEL_CREATE,
      }}
      title={group ? MODAL_HEADER_LABEL_EDIT : MODAL_HEADER_LABEL_CREATE}
      onClose={form.resetFields}>
      <Spinner spinning={isLoading}>
        <Form form={form} id={idPrefix + FORM_ID}>
          <Form.Item
            label={GROUP_NAME_LABEL}
            name={GROUP_NAME_NAME}
            required
            validateTrigger={['onSubmit', 'onChange']}>
            <Input autoComplete="off" autoFocus maxLength={128} placeholder="Group Name" />
          </Form.Item>
          <Form.Item label={USERS_LABEL} name={USERS_NAME}>
            <Select mode="multiple" optionFilterProp="children" placeholder="Add Users" showSearch>
              {users
                ?.filter((u) => u.isActive || currentGroupMembers?.includes(u.id))
                ?.map((u) => (
                  <Select.Option key={u.id} value={u.id}>
                    {getDisplayName(u)}
                  </Select.Option>
                ))}
            </Select>
          </Form.Item>
          {rbacEnabled && canModifyPermissions && group && (
            <>
              <Form.Item label={GROUP_ROLE_LABEL} name={GROUP_ROLE_NAME}>
                <Select
                  loading={Loadable.isNotLoaded(roles)}
                  mode="multiple"
                  optionFilterProp="children"
                  placeholder={'Add Roles'}
                  showSearch>
                  {Loadable.isLoaded(roles) && (
                    <>
                      {roles.data.map((r) => (
                        <Select.Option key={r.id} value={r.id}>
                          {r.name}
                        </Select.Option>
                      ))}
                    </>
                  )}
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
