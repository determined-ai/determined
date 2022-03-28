import { Button, Form, Input, message } from 'antd';
import { ModalStaticFunctions } from 'antd/es/modal/confirm';
import React, { useCallback, useState } from 'react';

import { StoreAction, useStore, useStoreDispatch } from 'contexts/Store';
import { patchUser } from 'services/api';
import handleError from 'utils/error';

import useModal, { ModalHooks } from '../useModal';

import css from './useModalChangeName.module.scss';

interface Props {
  onComplete: () => void;
}

const ChangeName: React.FC<Props> = ({ onComplete }) => {
  const { auth } = useStore();
  const userId = auth.user?.id ?? 0;
  const existingDisplayName = auth.user?.displayName;
  const [ form ] = Form.useForm();
  const [ isUpdating, setIsUpdating ] = useState(false);
  const storeDispatch = useStoreDispatch();

  const handleFormCancel = useCallback(() => {
    form.resetFields();
    onComplete();
  }, [ form, onComplete ]);

  const handleFormSubmit = useCallback(async () => {
    setIsUpdating(true);
    try {
      const user = await patchUser({
        userId,
        userParams: { displayName: form.getFieldValue('displayName') },
      });
      storeDispatch({ type: StoreAction.SetCurrentUser, value: user });
      message.success('Display name updated');
      onComplete();
    } catch (e) {
      message.error('Could not update display name');
      handleError(e);
    }
    setIsUpdating(false);
  }, [ form, onComplete, userId, storeDispatch ]);

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
            <Button htmlType="submit" loading={isUpdating} type="primary">
              Change name
            </Button>
          </div>
        </Form.Item>
      </Form>
    </div>
  );
};

const useModalChangeName = (modal: Omit<ModalStaticFunctions, 'warn'>): ModalHooks => {
  const { modalClose, modalOpen: openOrUpdate, modalRef } = useModal({ modal });

  const modalOpen = useCallback(() => {
    openOrUpdate({
      className: css.noFooter,
      closable: true,
      content: <ChangeName onComplete={modalClose} />,
      icon: null,
      title: <h5>Change name</h5>,
    });
  }, [ modalClose, openOrUpdate ]);

  return { modalClose, modalOpen, modalRef };
};

export default useModalChangeName;
