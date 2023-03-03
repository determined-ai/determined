import { MinusCircleOutlined, PlusOutlined } from '@ant-design/icons';
import { ModalFuncProps } from 'antd';
import React from 'react';
import { useCallback } from 'react';

import Button from 'components/kit/Button';
import Form from 'components/kit/Form';
import Input from 'components/kit/Input';
import { patchExperiment } from 'services/api';
import useModal, { ModalHooks as Hooks, ModalCloseReason } from 'shared/hooks/useModal/useModal';
import handleError from 'utils/error';

import css from './useModalExperimentEdit.module.scss';

type FormInputs = {
  description: string;
  experimentName: string;
  tags: string[];
};
interface Props {
  description: string;
  experimentId: number;
  experimentName: string;
  fetchExperimentDetails: () => void;
  onClose?: (reason?: ModalCloseReason) => void;
  tags: string[];
}

interface ModalHooks extends Omit<Hooks, 'modalOpen'> {
  modalOpen: () => void;
}

const FORM_ID = 'edit-experiment-form';

const useModalExperimentEdit = ({
  onClose,
  experimentName,
  experimentId,
  description,
  tags,
  fetchExperimentDetails,
}: Props): ModalHooks => {
  const [form] = Form.useForm<FormInputs>();
  const { modalOpen: openOrUpdate, ...modalHook } = useModal({ onClose });

  const getModalProps = useCallback((): ModalFuncProps => {
    const handleOk = async () => {
      const values = await form.validateFields();
      try {
        await patchExperiment({
          body: {
            description: values.description,
            labels: values.tags,
            name: values.experimentName,
          },
          experimentId,
        });
        fetchExperimentDetails();
      } catch (e) {
        handleError(e, {
          publicMessage: 'Unable to update name',
          silent: false,
        });
      }
    };

    const handleClose = () => {
      form.resetFields();
      onClose?.();
    };

    return {
      closable: true,
      content: (
        <Form autoComplete="off" form={form} id={FORM_ID} layout="vertical">
          <Form.Item
            initialValue={experimentName}
            label="Name"
            name="experimentName"
            rules={[{ max: 128, message: 'Name must be 1 ~ 128 characters', required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item initialValue={description} label="Description" name="description">
            <Input.TextArea />
          </Form.Item>
          <label>Tags</label>
          <Form.List initialValue={tags} name="tags">
            {(fields, { add, remove }) => (
              <>
                {fields.map(({ key, name, ...restField }) => (
                  <div className={css.tagItem} key={key}>
                    <Form.Item
                      {...restField}
                      name={name}
                      rules={[{ message: 'Tag is missing', required: true }]}>
                      <Input placeholder="Tag name" />
                    </Form.Item>
                    <MinusCircleOutlined
                      className={css.removeTagButton}
                      onClick={() => remove(name)}
                    />
                  </div>
                ))}
                <Form.Item>
                  <Button block icon={<PlusOutlined />} type="dashed" onClick={() => add()}>
                    Add Tag
                  </Button>
                </Form.Item>
              </>
            )}
          </Form.List>
        </Form>
      ),
      icon: null,
      okButtonProps: { form: FORM_ID, htmlType: 'submit', type: 'primary' },
      okText: 'Save',
      onCancel: handleClose,
      onOk: handleOk,
      title: 'Edit Experiment',
    };
  }, [description, experimentId, experimentName, fetchExperimentDetails, form, onClose, tags]);

  const modalOpen = useCallback(() => {
    openOrUpdate(getModalProps());
  }, [getModalProps, openOrUpdate]);

  return { modalOpen, ...modalHook };
};

export default useModalExperimentEdit;
