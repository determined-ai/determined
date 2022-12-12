import { Form, message, Select } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import { DefaultOptionType } from 'antd/es/select';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import useFeature from 'hooks/useFeature';
import { assignRolesToGroup, assignRolesToUser } from 'services/api';
import { V1Group, V1Role } from 'services/api-ts-sdk';
import Icon from 'shared/components/Icon/Icon';
import useModal, { ModalHooks } from 'shared/hooks/useModal/useModal';
import { DetError, ErrorLevel, ErrorType } from 'shared/utils/error';
import { User, UserOrGroup } from 'types';
import handleError from 'utils/error';
import { getIdFromUserOrGroup, getName, isUser } from 'utils/user';

import css from './useModalWorkspaceAddMember.module.scss';

interface Props {
  addableUsersAndGroups: UserOrGroup[];
  onClose?: () => void;
  rolesAssignableToScope: V1Role[];
  workspaceId: number;
}
interface FormInputs {
  roleId: number;
  userOrGroupId: string;
}

const useModalWorkspaceAddMember = ({
  addableUsersAndGroups,
  rolesAssignableToScope,
  onClose,
  workspaceId,
}: Props): ModalHooks => {
  let knownRoles = rolesAssignableToScope;
  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal();
  const [selectedOption, setSelectedOption] = useState<UserOrGroup>();
  const [form] = Form.useForm<FormInputs>();
  const mockWorkspaceMembers = useFeature().isOn('mock_workspace_members');

  knownRoles = useMemo(
    () =>
      mockWorkspaceMembers
        ? [
            {
              name: 'Editor',
              permissions: [],
              roleId: 1,
            },
            {
              name: 'Viewer',
              permissions: [],
              roleId: 2,
            },
          ]
        : knownRoles,
    [knownRoles, mockWorkspaceMembers],
  );

  const handleFilter = useCallback(
    (search: string, option: DefaultOptionType): boolean => {
      const label = option.label as string;
      const userOrGroup = addableUsersAndGroups.find((u) => {
        if (isUser(u)) {
          const user = u as User;
          return user?.displayName === label || user?.username === label;
        } else {
          const group = u as V1Group;
          return group.name === label;
        }
      });
      if (!userOrGroup) return false;
      if (isUser(userOrGroup)) {
        const userOption = userOrGroup as User;
        return userOption?.displayName?.includes(search) || userOption?.username?.includes(search);
      } else {
        const groupOption = userOrGroup as V1Group;
        return groupOption?.name?.includes(search) || false;
      }
    },
    [addableUsersAndGroups],
  );

  const handleSelect = useCallback(
    (value: string) => {
      const userOrGroup = addableUsersAndGroups.find((u) => {
        if (isUser(u) && value.substring(0, 2) === 'u_') {
          const user = u as User;
          return user.id === Number(value.substring(2));
        } else if (!isUser(u) && value.substring(0, 2) === 'g_') {
          const group = u as V1Group;
          return group.groupId === Number(value.substring(2));
        }
      });
      setSelectedOption(userOrGroup);
    },
    [addableUsersAndGroups],
  );

  const handleOk = useCallback(async () => {
    const values = await form.validateFields();
    try {
      if (values && selectedOption) {
        isUser(selectedOption)
          ? await assignRolesToUser({
              roleIds: [values.roleId],
              scopeWorkspaceId: workspaceId,
              userId: Number(values.userOrGroupId.substring(2)),
            })
          : await assignRolesToGroup({
              groupId: Number(values.userOrGroupId.substring(2)),
              roleIds: [values.roleId],
              scopeWorkspaceId: workspaceId,
            });
        form.resetFields();
        setSelectedOption(undefined);
        onClose?.();
        message.success(`${getName(selectedOption)} added to workspace,`);
      }
    } catch (e) {
      if (e instanceof DetError) {
        handleError(e, {
          level: e.level,
          publicMessage: e.publicMessage,
          publicSubject: 'Unable to add user or group to workspace.',
          silent: false,
          type: e.type,
        });
      } else {
        handleError(e, {
          level: ErrorLevel.Error,
          publicMessage: 'Please try again later.',
          publicSubject: 'Unable to add user or group to workspace.',
          silent: false,
          type: ErrorType.Server,
        });
      }
    }
  }, [form, selectedOption, workspaceId, onClose]);

  const modalContent = useMemo(() => {
    return (
      <div className={css.base}>
        <Form autoComplete="off" form={form} layout="vertical">
          <Form.Item
            label="User or Group"
            name="userOrGroupId"
            rules={[{ message: 'User or group is required ', required: true }]}>
            <Select
              filterOption={handleFilter}
              options={addableUsersAndGroups.map((option) => ({
                label: isUser(option) ? (
                  getName(option)
                ) : (
                  <span>
                    {getName(option)}&nbsp;&nbsp;
                    <Icon name="group" />
                  </span>
                ),
                value: (isUser(option) ? 'u_' : 'g_') + getIdFromUserOrGroup(option),
              }))}
              placeholder="Find user or group by display name or username"
              showSearch
              onSelect={handleSelect}
            />
          </Form.Item>
          <Form.Item
            label="Role"
            name="roleId"
            rules={[{ message: 'Role is required ', required: true }]}>
            <Select placeholder="Role">
              {rolesAssignableToScope.map((role) => (
                <Select.Option key={role.roleId} value={role.roleId}>
                  {role.name}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
        </Form>
      </div>
    );
  }, [addableUsersAndGroups, form, handleFilter, handleSelect, rolesAssignableToScope]);

  const getModalProps = useCallback((): ModalFuncProps => {
    return {
      closable: true,
      content: modalContent,
      icon: null,
      okText: 'Add Member',
      onOk: handleOk,
      title: 'Add Member',
    };
  }, [handleOk, modalContent]);

  const modalOpen = useCallback(
    (initialModalProps: ModalFuncProps = {}) => {
      openOrUpdate({ ...getModalProps(), ...initialModalProps });
    },
    [getModalProps, openOrUpdate],
  );

  /**
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal.
   */
  useEffect(() => {
    if (modalRef.current) openOrUpdate(getModalProps());
  }, [getModalProps, modalRef, openOrUpdate]);

  return { modalOpen, modalRef, ...modalHook };
};

export default useModalWorkspaceAddMember;
