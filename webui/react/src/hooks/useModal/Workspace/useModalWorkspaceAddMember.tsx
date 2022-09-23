import { Form, message, Select } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import { useStore } from 'contexts/Store';
import { assignRoles } from 'services/api';
import { V1Group } from 'services/api-ts-sdk';
import useModal, { ModalHooks } from 'shared/hooks/useModal/useModal';
import { DetError, ErrorLevel, ErrorType } from 'shared/utils/error';
import { User, UserOrGroup, Workspace } from 'types';
import handleError from 'utils/error';
import { createAssignmentRequest, getIdFromUserOrGroup, getName, isUser } from 'utils/user';

import css from './useModalWorkspaceAddMember.module.scss';

interface Props {
  addableUsersAndGroups: UserOrGroup[];
  onClose?: () => void;
  workspace: Workspace;
}
interface FormInputs {
  userOrGroupId: number;
  roleId: number;
}

const useModalWorkspaceAddMember = ({
  addableUsersAndGroups,
  onClose,
  workspace,
}: Props): ModalHooks => {
  const { knownRoles } = useStore();

  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal({ onClose });
  const [selectedOption, setSelectedOption] = useState<UserOrGroup>();
  const [form] = Form.useForm<FormInputs>();

  const handleFilter = useCallback(
    (search: string, option): boolean => {
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
    (value, option) => {
      const userOrGroup = addableUsersAndGroups.find((u) => {
        if (isUser(u)) {
          const user = u as User;
          return (
            (user?.displayName === option.label || user?.username === option.label) &&
            user.id === value
          );
        } else {
          const group = u as V1Group;
          return group.name === option.label && group.groupId === value;
        }
      });
      setSelectedOption(userOrGroup);
    },
    [addableUsersAndGroups],
  );

  const handleOk = useCallback(async () => {
    try {
      const values = await form.validateFields();
      if (values && selectedOption) {
        await assignRoles(createAssignmentRequest(values.roleId, selectedOption, values.userOrGroupId, workspace.id));
        form.resetFields();
        setSelectedOption(undefined);
        message.success(`${getName(selectedOption)} added to workspace,`)
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
  }, [workspace.id, form, selectedOption]);

  const modalContent = useMemo(() => {
    return (
      <div className={css.base}>
        <Form autoComplete="off" form={form} layout="vertical">
          <Form.Item name="userOrGroupId" rules={[{ message: 'User or group is required ', required: true }]}>
            <Select
              filterOption={handleFilter}
              options={addableUsersAndGroups.map((option) => ({
                label: getName(option),
                value: getIdFromUserOrGroup(option),
              }))}
              placeholder="Find user or group by display name or username"
              showSearch
              onSelect={handleSelect}
            />
          </Form.Item>
          <Form.Item name="roleId" rules={[{ message: 'Role is required ', required: true }]}>
            <Select placeholder="Role">
              {knownRoles.map((role) => (
                <Select.Option key={role.id} value={role.id}>
                  {role.name}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
        </Form>
      </div>
    );
  }, [addableUsersAndGroups, form, handleFilter, handleSelect, knownRoles]);

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
