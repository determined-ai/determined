import { Button, Form, Input, Modal } from 'antd';
import React, { useCallback, useMemo, useState } from 'react';

import css from './TextEditorModal.module.scss';

interface Props {
  disabled: boolean;
  onSave: (newValue: string) => Promise<Error | void>;
  placeholder: string;
  title: string;
  value: string;
}

interface FormInputs {
  text: string;
}

const TextEditorModal: React.FC<Props> = ({ disabled, onSave, title, placeholder, value }) => {
  const [isModalOpen, setIsModalOpen] = useState<boolean>(false);
  const [isConfirmLoading, setIsConfirmLoading] = useState<boolean>(false);
  const [form] = Form.useForm<FormInputs>();
  const classes = useMemo(() => {
    const classList = [];
    if (!value) classList.push(css.buttonBlur);
    return classList.join(' ');
  }, [value]);

  const onShowModal = () => {
    setIsModalOpen(true);
  };
  const onHideModal = () => setIsModalOpen(false);

  const onSubmit = useCallback(async () => {
    const value = await form.validateFields();
    setIsConfirmLoading(true);
    onSave(value.text).then(() => {
      onHideModal();
      setIsConfirmLoading(false);
    });
  }, [form, onSave]);

  return (
    <>
      <Button disabled={disabled} type="text" onClick={onShowModal}>
        <span className={classes}>{value ? value : placeholder}</span>
      </Button>
      <Modal
        confirmLoading={isConfirmLoading}
        title={title}
        visible={isModalOpen}
        onCancel={onHideModal}
        onOk={onSubmit}>
        <Form form={form} layout="vertical">
          <Form.Item initialValue={value} name="text">
            <Input.TextArea placeholder={placeholder} rows={8} />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};

export default TextEditorModal;
