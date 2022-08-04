import { Form, Input, message, Select } from 'antd';
import { FormInstance } from 'antd/lib/form/hooks/useForm';
import React, { useCallback, useEffect, useState } from 'react';

import { useStore } from 'contexts/Store';
import { createGroup, deleteGroup, getGroup } from 'services/api';
import { V1GroupDetails, V1GroupSearchResult } from 'services/api-ts-sdk';
import useModal, { ModalHooks } from 'shared/hooks/useModal/useModal';
import { isEqual } from 'shared/utils/data';
import { ErrorType } from 'shared/utils/error';
import { BrandingType, DetailedUser } from 'types';
import handleError from 'utils/error';

export const MODAL_HEADER_LABEL_CREATE = 'Create Group';
export const MODAL_HEADER_LABEL_EDIT = 'Edit Group';

export const GROUP_NAME_NAME = 'name';
export const GROUP_NAME_LABEL = 'Group Name';
export const USER_ADD_NAME = 'addUsers';
export const USER_LABEL = 'Users';
export const API_SUCCESS_MESSAGE_CREATE = 'New group has been created.';
export const API_SUCCESS_MESSAGE_EDIT = 'Group has been updated.';

interface Props {
  users: DetailedUser[];
  form: FormInstance;
  group?: V1GroupDetails;
  isLoading?: boolean
}

const ModalForm: React.FC<Props> = ({ form, users, group, isLoading }) => {
  useEffect(() => {
    console.log(group);
    form.setFieldsValue({ [GROUP_NAME_NAME]: group?.name });
  }, [ group, form ]);
  return (
    <Form
      form={form}
      labelCol={{ span: 8 }}
      wrapperCol={{ span: 14 }}>
      <Form.Item
        label={GROUP_NAME_LABEL}
        name={GROUP_NAME_NAME}
        required
        rules={[
          {
            message: 'Please type in your group name.',
            required: true,
          },
        ]}
        validateTrigger={[ 'onSubmit' ]}>
        <Input autoFocus maxLength={128} placeholder="Group Name" />
      </Form.Item>
      <Form.Item
        label={USER_LABEL}
        name={USER_ADD_NAME}>
        <Select mode="multiple" optionFilterProp="children" placeholder="Add Users" showSearch>{
          users.filter((u) => !group?.users?.map((gu) => gu.id).includes(u.id)).map((u) => (
            <Select.Option key={u.id} value={u.id}>{u.displayName || u.username}</Select.Option>
          ))
        }
        </Select>
      </Form.Item>
    </Form>
  );
};

interface ModalProps {
  onClose?: () => void;
  users: DetailedUser[];
  group?: V1GroupSearchResult;
}

const useModalCreateGroup = ({ onClose, users, group }: ModalProps): ModalHooks => {

  const [ form ] = Form.useForm();
  const { info } = useStore();
  const [ groupDetail, setGroupDetail ] = useState<V1GroupDetails>();
  const [ isLoading, setIsLoading ] = useState(true);

  const { modalOpen: openOrUpdate, ...modalHook } = useModal();

  const fetchGroup = useCallback(async () => {
    if (group?.group.groupId) {
      try {
        const response = await getGroup({ groupId: group?.group.groupId });
        setGroupDetail(response.group);
      } catch (e) {
        handleError(e, { publicSubject: 'Unable to fetch groups.' });
      } finally {
        setIsLoading(false);
      }
    }

  }, [ group ]);

  useEffect(() => {
    fetchGroup();
  }, [ group ]);

  const handleCancel = useCallback(() => {
    form.resetFields();
  }, [ form ]);

  const handleOkay = useCallback(async () => {
    await form.validateFields();

    try {
      const formData = form.getFieldsValue();
      await createGroup(formData);
      message.success(API_SUCCESS_MESSAGE_CREATE);
      form.resetFields();
      onClose?.();
    } catch (e) {
      message.error('error creating new group');
      handleError(e, { silent: true, type: ErrorType.Input });

      // Re-throw error to prevent modal from getting dismissed.
      throw e;
    }
  }, [ form, onClose ]);

  const modalOpen = useCallback(() => {
    openOrUpdate({
      closable: true,
      content: <ModalForm form={form} group={groupDetail} isLoading={isLoading} users={users} />,
      icon: null,
      okText: group ? 'Edit Group' : 'Create Group',
      onCancel: handleCancel,
      onOk: handleOkay,
      title: <h5>{group ? MODAL_HEADER_LABEL_EDIT : MODAL_HEADER_LABEL_CREATE}</h5>,
    });
  }, [ form, handleCancel, handleOkay, openOrUpdate, info, users, groupDetail ]);

  return { modalOpen, ...modalHook };
};

export default useModalCreateGroup;
