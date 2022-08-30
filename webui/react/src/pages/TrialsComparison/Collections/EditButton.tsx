import { Button, Form, Input, Modal, notification } from 'antd';
import yaml from 'js-yaml';
import React, { useCallback, useState } from 'react';

import MonacoEditor from 'components/MonacoEditor';

import { TrialFilters } from './filters';

interface Props {
  collectionName: string;
  filters: TrialFilters;
  saveCollection: (name: string) => void;
}

interface FormInputs {
  collectionName: string;
}

const EditButton: React.FC<Props> = ({ saveCollection, filters, collectionName }) => {
  const [ form ] = Form.useForm<FormInputs>();
  const [ isModalVisible, setIsModalVisible ] = useState<boolean>(false);

  const onShowModal = useCallback(() => setIsModalVisible(true), []);

  const onHideModal = useCallback(() => setIsModalVisible(false), []);

  const onSubmit = useCallback(async () => {
    const values = await form.validateFields();
    saveCollection(values.collectionName);
    notification.success({ message: 'Collection is sucessfully saved!' });
    onHideModal();
  }, [ form, onHideModal, saveCollection ]);

  return (
    <>
      <Button onClick={onShowModal}>Edit Collection</Button>
      <Modal
        title={`Edit Collection: ${collectionName}`}
        visible={isModalVisible}
        width={'clamp(280px, 416px, calc(100vw - 16px))'}
        onCancel={onHideModal}
        onOk={onSubmit}>
        <Form autoComplete="off" form={form} layout="vertical">
          <Form.Item
            initialValue={collectionName}
            name="collectionName"
            rules={[ { message: 'Collection name is required ', required: true } ]}>
            <Input
              allowClear
              bordered={true}
              placeholder="enter collection name"
              onPressEnter={onSubmit}
            />
          </Form.Item>
          <MonacoEditor
            height="40vh"
            language="yaml"
            options={{
              minimap: { enabled: false },
              occurrencesHighlight: false,
              readOnly: true,
            }}
            value={yaml.dump(filters)}
          />
        </Form>
      </Modal>
    </>
  );
};

export default EditButton;
