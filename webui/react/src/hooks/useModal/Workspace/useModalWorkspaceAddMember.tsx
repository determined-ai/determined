import { Select } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import GroupAvatar from 'components/GroupAvatar';
import Form from 'components/kit/Form';
import UserBadge from 'components/kit/UserBadge';
import { assignRolesToGroup, assignRolesToUser } from 'services/api';
import { V1Group, V1Role } from 'services/api-ts-sdk';
import useModal, { ModalHooks } from 'shared/hooks/useModal/useModal';
import { DetError, ErrorLevel, ErrorType } from 'shared/utils/error';
import { User, UserOrGroup } from 'types';
import { message } from 'utils/dialogApi';
import handleError from 'utils/error';
import { getIdFromUserOrGroup, getName, isUser, UserNameFields } from 'utils/user';

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

interface SearchProp {
  label: {
    props: {
      groupName?: string;
      user?: User;
    };
  };
}

const useModalWorkspaceAddMember = ({
  addableUsersAndGroups,
  rolesAssignableToScope,
  onClose,
  workspaceId,
}: Props): ModalHooks => {
  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal();
  const [selectedOption, setSelectedOption] = useState<UserOrGroup>();
  const [form] = Form.useForm<FormInputs>();

  const handleFilter = useCallback((search: string, option?: SearchProp): boolean => {
    if (!option) return false;
    const label = option.label;
    return (
      label.props.groupName?.includes(search) ||
      label.props.user?.username?.includes(search) ||
      label.props.user?.displayName?.includes(search) ||
      false
    );
  }, []);

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
                  <UserBadge compact user={option as UserNameFields} />
                ) : (
                  <GroupAvatar groupName={getName(option)} />
                ),
                value: (isUser(option) ? 'u_' : 'g_') + getIdFromUserOrGroup(option),
              }))}
              placeholder="User or Group"
              showSearch
              size="large"
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
