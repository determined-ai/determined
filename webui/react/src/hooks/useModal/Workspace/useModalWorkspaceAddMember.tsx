import { Form, Select } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { assignRoles } from 'services/api';
import { useStore } from 'contexts/Store';
import useModal, { ModalHooks } from 'shared/hooks/useModal/useModal';
import { UserOrGroup, Workspace, User } from 'types';
import { V1Group, V1GroupDetails} from 'services/api-ts-sdk';
import { getName, isUser } from 'utils/user';
import { DetError, ErrorLevel, ErrorType } from 'shared/utils/error';
import handleError from 'utils/error';

import css from './useModalWorkspaceAddMember.module.scss';

const getId = (obj: UserOrGroup | V1GroupDetails) => {
 if(isUser(obj)){
   const user = obj as User;
   return user.id
 }
 const group = obj as V1Group;
 return group.groupId
}
interface Props {
  onClose?: () => void;
  workspace: Workspace;
  groups: V1Group[];
}
interface FormInputs {
  role: string;
  id: number;
}

// Adding this lint rule to keep the reference to the workspace
// which will be needed when calling the API.
/* eslint-disable-next-line @typescript-eslint/no-unused-vars */
const useModalWorkspaceAddMember = ({ onClose, workspace, groups  }: Props): ModalHooks => {
  const {users} = useStore();
  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal({ onClose });
  const [selectedOption, setSelectedOption] = useState<UserOrGroup>();
  const [form] = Form.useForm<FormInputs>();

  const usersAndGroups = useMemo(() => [...users, ...groups], [users, groups]);
  const handleFilter = useCallback(
    (search: string, option): boolean => {
      const label = option.label as string;
      const userOrGroup = usersAndGroups.find((u) => {
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
        return (
          userOption?.displayName?.includes(search) || userOption?.username?.includes(search)
        );
      } else {
        const groupOption = userOrGroup as V1Group;
        return groupOption?.name?.includes(search) || false;
      }
    },
    [usersAndGroups],
  );
  
  const handleSelect = useCallback((value, option) => {
    const userOrGroup = usersAndGroups.find(u => {
      if(isUser(u)){
        const user = u as User;
        return (user?.displayName == option.label || user?.username == option.label)  && user.id == value
      } else {
        const group = u as V1Group;
        return group.name == option.label && group.groupId == value
      }
    }
    )
    setSelectedOption(userOrGroup);
  }, [usersAndGroups])

  const handleOk = useCallback(
    async () => {
      try {
        const values = await form.validateFields();
        if (values && selectedOption) {
          
          const roleAssignment = {
            role: {
              roleId: 0
            },
            scopeWorkspaceId: workspace.id
          }
          const assignment = isUser(selectedOption) ?
          { 
            userRoleAssignments: 
            [{
            userId: values.id,
            roleAssignment
          }]
        } :
          { 
            groupRoleAssignments:[{
            groupId: values.id,
            roleAssignment
          }]
        }
          await assignRoles(assignment);
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
            publicSubject: 'Unable to create project.',
            silent: false,
            type: ErrorType.Server,
          });
        }
      }},
    [form, selectedOption],
  );

  const modalContent = useMemo(() => {

  // Mock Data for potential roles
  const roles = ['Basic', 'Cluster Admin', 'Editor', 'Viewer', 'Restricted', 'Workspace Admin'];
    return (
      <div className={css.base}>
        <Form autoComplete="off" form={form} layout="vertical">
        <Form.Item
          name="id"
          rules={[{ message: 'User or group is required ', required: true }]}>
        <Select
          filterOption={handleFilter}
          options={usersAndGroups.map((option) => ({ label: getName(option), value: getId(option) }))}
          onSelect={handleSelect}
          placeholder="Find user or group by display name or username"
          showSearch
        />
        </Form.Item>
        <Form.Item
        name="role"
        rules={[{ message: 'Role is required ', required: true }]}>
        <Select placeholder="Role">
          {roles.map((r) => (
            <Select.Option key={r} value={r}>
              {r}
            </Select.Option>
          ))}
        </Select>
        </Form.Item>
        </Form>
      </div>
    );
  }, [handleFilter, usersAndGroups]);

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
