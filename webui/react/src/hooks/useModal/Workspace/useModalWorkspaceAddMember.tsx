import { Group } from '@storybook/api';
import { Select } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { useCallback, useEffect, useMemo } from 'react';

import { useStore } from 'contexts/Store';
import useModal, { ModalHooks } from 'shared/hooks/useModal/useModal';
import { UserOrGroup, Workspace, User } from 'types';
import { V1Group, V1GroupDetails } from 'services/api-ts-sdk';
import { getName, isUser } from 'utils/user';

import css from './useModalWorkspaceAddMember.module.scss';

const getId = (obj: UserOrGroup | V1GroupDetails) => {
 if(isUser(obj)){
   const user = obj as User;
   return user.id
 }
 const group = obj as V1Group;
 return group.groupId;
}
interface Props {
  onClose?: () => void;
  workspace: Workspace;
  groups: V1Group[];
}

// Adding this lint rule to keep the reference to the workspace
// which will be needed when calling the API.
/* eslint-disable-next-line @typescript-eslint/no-unused-vars */
const useModalWorkspaceAddMember = ({ onClose, workspace, groups  }: Props): ModalHooks => {
  const {users} = useStore();
  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal({ onClose });
  const usersAndGroups = [...users, ...groups];
  const handleFilter = useCallback(
    (search: string, option) => {
      const label = option.label as string;
      const userOrGroup = usersAndGroups.find((u) => {
        if (isUser(u)) {
          const member = u as User;
          return member?.displayName === label || member?.username === label;
        } else {
          const group = u as Group;
          return group.name === label;
        }
      });
      if (!userOrGroup) return false;
      if (isUser(userOrGroup)) {
        const userOption = userOrGroup as User;
        return (
          userOption?.displayName?.includes(search) || userOption?.username?.includes(search)
        );
      } else {
        const gOption: unknown = userOrGroup;
        const groupOption = gOption as Group;
        return groupOption?.name?.includes(search);
      }
    },
    [usersAndGroups],
  );

  const modalContent = useMemo(() => {

  // Mock Data for potential roles
  const roles = ['Basic', 'Cluster Admin', 'Editor', 'Viewer', 'Restricted', 'Workspace Admin'];
    return (
      <div className={css.base}>
        <Select
          filterOption={handleFilter}
          options={usersAndGroups.map((option) => ({ label: getName(option), value: getId(option) }))}
          placeholder="Find user or group by display name or username"
          showSearch
        />
        <Select placeholder="Role">
          {roles.map((r) => (
            <Select.Option key={r} value={r}>
              {r}
            </Select.Option>
          ))}
        </Select>
      </div>
    );
  }, [handleFilter, usersAndGroups]);

  const getModalProps = useCallback((): ModalFuncProps => {
    return {
      closable: true,
      content: modalContent,
      icon: null,
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
