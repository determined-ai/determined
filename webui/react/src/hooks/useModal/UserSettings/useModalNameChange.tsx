import { Form, Input, message } from 'antd';
import { FormInstance } from 'antd/lib/form/hooks/useForm';
import React, { useCallback } from 'react';

import { StoreAction, useStore, useStoreDispatch } from 'contexts/Store';
import { patchUser } from 'services/api';
import { ErrorType } from 'shared/utils/error';
import handleError from 'utils/error';

import useModal, { ModalHooks } from '../../../shared/hooks/useModal/useModal';

interface Props {
  displayName?: string;
  form: FormInstance;
}

export const MODAL_HEADER_LABEL = 'Change Display Name';
export const DISPLAY_NAME_LABEL = 'Display Name';
export const DISPLAY_NAME_NAME = 'displayName';
export const CANCEL_BUTTON_LABEL = 'Cancel';
export const OK_BUTTON_LABEL = 'Update Display Name';
export const NAME_TOO_LONG_MESSAGE = 'Name can\'t be longer than 80 characters.';
export const API_SUCCESS_MESSAGE = 'Display name updated.';
export const API_ERROR_MESSAGE = 'Could not update display name.';

const ModalForm: React.FC<Props> = ({ displayName = '', form }) => (
  <Form form={form} layout="vertical">
    <Form.Item
      initialValue={displayName}
      label={DISPLAY_NAME_LABEL}
      name={DISPLAY_NAME_NAME}
      rules={[ { max: 80, message: NAME_TOO_LONG_MESSAGE } ]}
      validateTrigger={[ 'onBlur' ]}>
      <Input />
    </Form.Item>
  </Form>
);

const useModalNameChange = (): ModalHooks => {
  const [ form ] = Form.useForm();
  const { auth } = useStore();
  const storeDispatch = useStoreDispatch();
  const userId = auth.user?.id ?? 0;

  const { modalOpen: openOrUpdate, ...modalHook } = useModal();

  const handleCancel = useCallback(() => form.resetFields(), [ form ]);

  const handleOkay = useCallback(async () => {
    await form.validateFields();

    try {
      const user = await patchUser({
        userId,
        userParams: { displayName: form.getFieldValue(DISPLAY_NAME_NAME) },
      });
      storeDispatch({ type: StoreAction.SetCurrentUser, value: user });
      message.success(API_SUCCESS_MESSAGE);
    } catch (e) {
      message.error(API_ERROR_MESSAGE);
      handleError(e, { silent: true, type: ErrorType.Input });

      // Re-throw error to prevent modal from getting dismissed.
      throw e;
    }
  }, [ form, storeDispatch, userId ]);

  const modalOpen = useCallback(() => {
    openOrUpdate({
      closable: true,
      content: <ModalForm displayName={auth.user?.displayName} form={form} />,
      icon: null,
      okText: OK_BUTTON_LABEL,
      onCancel: handleCancel,
      onOk: handleOkay,
      title: <h5>{MODAL_HEADER_LABEL}</h5>,
    });
  }, [ auth.user?.displayName, form, handleCancel, handleOkay, openOrUpdate ]);

  return { modalOpen, ...modalHook };
};

export default useModalNameChange;
