import { Form, Select } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { assignRoles } from 'services/api';
import { useStore } from 'contexts/Store';
import useModal, { ModalHooks } from 'shared/hooks/useModal/useModal';
import { UserOrGroup, Workspace, User } from 'types';
import { V1Group, V1GroupDetails } from 'services/api-ts-sdk';
import { createAssignmentRequest, getName, getIdFromUserOrGroup, isUser } from 'utils/user';
import { DetError, ErrorLevel, ErrorType } from 'shared/utils/error';
import handleError from 'utils/error';

import css from './useModalWorkspaceAddMember.module.scss';

interface Props {
  onClose?: () => void;
  workspace: Workspace;
  addableUsersAndGroups: UserOrGroup[];
}
interface FormInputs {
  role: string;
  id: number;
}

const useModalWorkspaceAddMember = ({
  onClose,
  workspace,
  addableUsersAndGroups,
}: Props): ModalHooks => {
  const { knownRoles, users } = useStore();

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
            (user?.displayName == option.label || user?.username == option.label) &&
            user.id == value
          );
        } else {
          const group = u as V1Group;
          return group.name == option.label && group.groupId == value;
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
        await assignRoles(createAssignmentRequest(selectedOption, values.id, 0, workspace.id));
        form.resetFields();
        setSelectedOption(undefined);
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
  }, [form, selectedOption]);

  const modalContent = useMemo(() => {
    return (
      <div className={css.base}>
        <Form autoComplete="off" form={form} layout="vertical">
          <Form.Item name="id" rules={[{ message: 'User or group is required ', required: true }]}>
            <Select
              filterOption={handleFilter}
              options={addableUsersAndGroups.map((option) => ({
                label: getName(option),
                value: getIdFromUserOrGroup(option),
              }))}
              onSelect={handleSelect}
              placeholder="Find user or group by display name or username"
              showSearch
            />
          </Form.Item>
          <Form.Item name="role" rules={[{ message: 'Role is required ', required: true }]}>
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
  }, [handleFilter, addableUsersAndGroups]);

  const getModalProps = useCallback((): ModalFuncProps => {
    return {
      closable: true,
      content: modalContent,
      icon: null,
      onOk: handleOk,
      okText: 'Add Member',
      title: 'Add Member',
    };
  }, [modalContent]);

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
