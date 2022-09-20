import { Group } from '@storybook/api';
import { Select } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { useCallback, useEffect, useMemo } from 'react';

import { useStore } from 'contexts/Store';
import useModal, { ModalHooks } from 'shared/hooks/useModal/useModal';
import { Member, MemberOrGroup, Workspace } from 'types';
import { getName, isMember } from 'utils/user';

import css from './useModalWorkspaceAddMember.module.scss';

interface Props {
  onClose?: () => void;
  workspace: Workspace;
}

// Adding this lint rule to keep the reference to the workspace
// which will be needed when calling the API.
/* eslint-disable-next-line @typescript-eslint/no-unused-vars */
const useModalWorkspaceAddMember = ({ onClose, workspace }: Props): ModalHooks => {
  const { users } = useStore();
  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal({ onClose });

  // Mock Data for potential roles
  const roles = [ 'Basic', 'Cluster Admin', 'Editor', 'Viewer', 'Restricted', 'Workspace Admin' ];

  const members: Member[] = [];

  // Assign a mock role to users
  users.forEach((u) => {
    const m: Member = u;
    m.role = roles[2];
    members.push(m);
  });

  // Create mock groups to show the UI renders correctly
  const groups: MemberOrGroup[] = [
    { id: 999, name: 'Group One', role: roles[0] },
    { id: 1000, name: 'Group Two', role: roles[1] },
    { id: 1001 * 1000, name: 'Group Three', role: roles[5] },
  ];

  // Mock table row data
  const membersAndGroups = groups.concat(members);

  const handleFilter = useCallback((search: string, option) => {
    const label = option.label as string;
    const memberOrGroup = membersAndGroups.find((m) => {
      if (isMember(m)){
        const member = m as Member;
        return member?.displayName === label || member?.username === label;
      } else {
        const g: unknown = m;
        const group = g as Group;
        return group.name === label;
      }
    });
    if (!memberOrGroup) return false;
    if (isMember(memberOrGroup)){
        const memberOption = memberOrGroup as Member;
        return memberOption?.displayName?.includes(search) || memberOption?.username?.includes(search);
      } else {
        const gOption: unknown = memberOrGroup;
        const groupOption = gOption as Group;
        return groupOption?.name?.includes(search);
      }
    }, [membersAndGroups]);
  
  const modalContent = useMemo(() => {
    return (
      <div className={css.base}>
        <Select
          filterOption={handleFilter}
          options={membersAndGroups.map((option) => ({ label: getName(option), value: option.id }))}
          placeholder="Find user or group by display name or username"
          showSearch
        />
        <Select
        placeholder="Role">
          {roles.map((r) => (
            <Select.Option key={r} value={r}>
              {r}
            </Select.Option>
            ))}
        </Select>
      </div>
    );
  }, []);

  const getModalProps = useCallback((): ModalFuncProps => {
    return {
      closable: true,
      content: modalContent,
      icon: null,
      okText: 'Add Member',
      title: 'Add Member',
    };
  }, [ modalContent ]);

  const modalOpen = useCallback((initialModalProps: ModalFuncProps = {}) => {
    openOrUpdate({ ...getModalProps(), ...initialModalProps });
  }, [ getModalProps, openOrUpdate ]);

  /**
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal.
   */
  useEffect(() => {
    if (modalRef.current) openOrUpdate(getModalProps());
  }, [ getModalProps, modalRef, openOrUpdate ]);

  return { modalOpen, modalRef, ...modalHook };
};

export default useModalWorkspaceAddMember ;
