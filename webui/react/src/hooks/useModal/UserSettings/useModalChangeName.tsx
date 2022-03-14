import { Button, Form, Input, message } from 'antd';
import React, { useCallback } from 'react';

import { StoreAction, useStore, useStoreDispatch } from 'contexts/Store';
import { patchUser } from 'services/api';
import handleError from 'utils/error';

import useModal, { ModalHooks } from '../useModal';

import css from './useModalChangeName.module.scss';

const useModalChangeName = (): ModalHooks => {
  const { modalClose, modalOpen: openOrUpdate, modalRef } = useModal();
  const { auth } = useStore();
  const username = auth.user?.username ?? '';
  const existingDisplayName = auth.user?.displayName;
  const [ form ] = Form.useForm();
  const storeDispatch = useStoreDispatch();

  const handleFormCancel = useCallback(() => {
    form.resetFields();
    modalClose();
  }, [ form, modalClose ]);

  const handleFormSubmit = useCallback(async () => {
    try {
      const user = await patchUser({
        username,
        userParams: { displayName: form.getFieldValue('displayName') },
      });
      storeDispatch({ type: StoreAction.SetCurrentUser, value: user });
      message.success('Display name updated');
      modalClose();
    } catch (e) {
      message.error('Could not update display name');
      handleError(e);
    }
  }, [ form, modalClose, username, storeDispatch ]);

  const getModalContent = useCallback(() => {
    return (
      <div className={css.base}>
        <Form form={form} layout="vertical" onFinish={handleFormSubmit}>
          <Form.Item
            initialValue={existingDisplayName}
            label="Display name"
            name="displayName"
            required
            rules={[
              {
                max: 80,
                message: 'Name can\'t be longer than 80 characters',
              },
            ]}
            validateTrigger={[ 'onBlur' ]}>
            <Input />
          </Form.Item>
          <Form.Item>
            {/* override modal buttons with form buttons
            to ensure form validation works as intended */}
            <div className={css.buttons}>
              <Button onClick={handleFormCancel}>Cancel</Button>
              <Button htmlType="submit" type="primary">
                Change name
              </Button>
            </div>
          </Form.Item>
        </Form>
      </div>
    );
  }, [ form, handleFormSubmit, handleFormCancel, existingDisplayName ]);

  const modalOpen = useCallback(() => {
    openOrUpdate({
      className: css.noFooter,
      closable: true,
      content: getModalContent(),
      icon: null,
      title: 'Change name',
    });
  }, [ getModalContent, openOrUpdate ]);

  return { modalClose, modalOpen, modalRef };
};

export default useModalChangeName;
