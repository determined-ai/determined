import Form from 'hew/Form';
import { Modal } from 'hew/Modal';
import Select, { Option, RefSelectProps } from 'hew/Select';
import { makeToast } from 'hew/Toast';
import React, { useCallback, useEffect, useId, useRef, useState } from 'react';

import UserBadge from 'components/UserBadge';
import { updateGroup } from 'services/api';
import { V1GroupSearchResult } from 'services/api-ts-sdk';
import { DetailedUser } from 'types';
import handleError, { DetError, ErrorLevel, ErrorType } from 'utils/error';

const FORM_ID = 'add-members-to-group-modal-form';

interface Props {
  users: DetailedUser[];
  onClose?: () => void;
  group: V1GroupSearchResult;
}
type FormInputs = {
  userIds: number[];
};

const AddUsersToGroupModalComponent: React.FC<Props> = ({ users, onClose, group }: Props) => {
  const idPrefix = useId();
  const [filteredUsers, setFilteredUsers] = useState<DetailedUser[]>([]);
  const [form] = Form.useForm<FormInputs>();
  const ref = useRef<RefSelectProps>(null);

  useEffect(() => {
    ref.current?.focus();
  }, []);

  const handleSearch = useCallback(
    (value: string) => {
      if (value) {
        const regex = new RegExp(value, 'i');
        setFilteredUsers(
          users.filter((user) => {
            return (
              (user.displayName && regex.test(user.displayName)) ||
              regex.test(user.username) ||
              false
            );
          }),
        );
      } else {
        setFilteredUsers([]);
      }
    },
    [users],
  );

  const handleSubmit = useCallback(async () => {
    const values = await form.validateFields();
    try {
      const groupId = group.group.groupId;
      if (groupId === undefined) {
        throw new Error('groupId is undefined');
      }
      await updateGroup({ addUsers: values.userIds, groupId });
      if (values) {
        form.resetFields();
        onClose?.();
        makeToast({
          severity: 'Confirm',
          title: `${values.userIds.length} users added to group.`,
        });
      }
    } catch (e) {
      if (e instanceof DetError) {
        handleError(e, {
          level: e.level,
          publicMessage: e.publicMessage,
          publicSubject: 'Unable to add users to group.',
          silent: false,
          type: e.type,
        });
      } else {
        handleError(e, {
          level: ErrorLevel.Error,
          publicMessage: 'Please try again later.',
          publicSubject: 'Unable to add users to group.',
          silent: false,
          type: ErrorType.Server,
        });
      }
    }
  }, [form, group.group.groupId, onClose]);

  return (
    <Modal
      cancel
      size="small"
      submit={{
        disabled: !form.getFieldValue('userIds') || form.getFieldError('userIds').length > 0,
        form: idPrefix + FORM_ID,
        handleError,
        handler: handleSubmit,
        text: 'Add to Group',
      }}
      title={`Add members to ${group.group.name}`}>
      <Form autoComplete="off" form={form} id={idPrefix + FORM_ID} layout="vertical">
        <Form.Item
          label="Find and Select Users"
          name="userIds"
          rules={[{ message: 'User is required ', required: true }]}>
          <Select
            filterOption={false}
            mode="multiple"
            optionLabelProp="label"
            placeholder="Add Users"
            ref={ref}
            searchable={true}
            onBlur={() => setFilteredUsers([])}
            onSearch={handleSearch}>
            {filteredUsers.map((user) => (
              <Option key={user.id} label={user.displayName || user.username} value={user.id}>
                <UserBadge compact user={user} />
              </Option>
            ))}
          </Select>
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default AddUsersToGroupModalComponent;
