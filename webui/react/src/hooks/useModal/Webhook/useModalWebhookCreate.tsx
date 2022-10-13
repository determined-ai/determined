import { Form, Input, Select } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { useCallback, useEffect, useMemo } from 'react';

import { paths } from 'routes/utils';
import { createWebhook } from 'services/api';
import { V1TriggerType, V1WebhookType } from 'services/api-ts-sdk/api';
import useModal, { ModalHooks } from 'shared/hooks/useModal/useModal';
import { DetError, ErrorLevel, ErrorType } from 'shared/utils/error';
import { routeToReactUrl } from 'shared/utils/routes';
import { RunState } from 'types';
import handleError from 'utils/error';

interface FormInputs {
  triggerEvents: RunState[];
  url: string;
  webhookType: V1WebhookType;
}

interface Props {
  onClose?: () => void;
}

const useModalWebhookCreate = ({ onClose }: Props): ModalHooks => {
  const [form] = Form.useForm<FormInputs>();
  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal({ onClose });

  const modalContent = useMemo(() => {
    return (
      <Form autoComplete="off" form={form} layout="vertical">
        <Form.Item
          label="URL"
          name="url"
          rules={[
            { message: 'URL is required.', required: true },
            { message: 'URL must be valid.', type: 'url', whitespace: true },
          ]}>
          <Input />
        </Form.Item>
        <Form.Item
          label="Type"
          name="webhookType"
          rules={[{ message: 'Webhook type is required ', required: true }]}>
          <Select placeholder="Select type of Webhook">
            <Select.Option key={V1WebhookType.DEFAULT} value={V1WebhookType.DEFAULT}>
              Default
            </Select.Option>
            <Select.Option key={V1WebhookType.SLACK} value={V1WebhookType.SLACK}>
              Slack
            </Select.Option>
          </Select>
        </Form.Item>
        <Form.Item
          label="Trigger"
          name="triggerEvents"
          rules={[{ message: 'At least one trigger event is required', required: true }]}>
          <Select mode="multiple" placeholder="Select trigger event">
            <Select.Option key={RunState.Completed} value={RunState.Completed}>
              {RunState.Completed}
            </Select.Option>
            <Select.Option key={RunState.Error} value={RunState.Error}>
              {RunState.Error}
            </Select.Option>
          </Select>
        </Form.Item>
      </Form>
    );
  }, [form]);

  const handleOk = useCallback(async () => {
    const values = await form.validateFields();

    try {
      if (values) {
        await createWebhook({
          triggers: values.triggerEvents.map((state) => ({
            condition: { state },
            triggerType: V1TriggerType.EXPERIMENTSTATECHANGE,
          })),
          url: values.url,
          webhookType: values.webhookType,
        });
        routeToReactUrl(paths.webhooks());
        form.resetFields();
      }
    } catch (e) {
      if (e instanceof DetError) {
        handleError(e, {
          level: e.level,
          publicMessage: e.publicMessage,
          publicSubject: 'Unable to create webhook.',
          silent: false,
          type: e.type,
        });
      } else {
        handleError(e, {
          level: ErrorLevel.Error,
          publicMessage: 'Please try again later.',
          publicSubject: 'Unable to create webhook.',
          silent: false,
          type: ErrorType.Server,
        });
      }
    }
  }, [form]);

  const getModalProps = useCallback((): ModalFuncProps => {
    return {
      closable: true,
      content: modalContent,
      icon: null,
      okText: 'Create Webhook',
      onOk: handleOk,
      title: 'New Webhook',
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

export default useModalWebhookCreate;
