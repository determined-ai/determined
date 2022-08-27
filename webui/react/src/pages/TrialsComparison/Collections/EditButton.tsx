import { Button, Modal, notification } from 'antd';
import yaml from 'js-yaml';
import React, { useCallback, useState } from 'react';
import MonacoEditor from 'react-monaco-editor';

import { useStore } from 'contexts/Store';
import Spinner from 'shared/components/Spinner';
import { DarkLight } from 'shared/themes';

import { TrialFilters } from './filters';

interface Props {
  collectionName: string;
  filters: TrialFilters;
  saveCollection: () => void;
}

const EditButton: React.FC<Props> = ({ saveCollection, filters, collectionName }) => {
  // theme doesnt apply to MonacoEditor somehow, so its the workaround
  const { ui } = useStore();
  const [ isModalVisible, setIsModalVisible ] = useState<boolean>(false);

  const onShowModal = useCallback(() => setIsModalVisible(true), []);

  const onHideModal = useCallback(() => setIsModalVisible(false), []);

  const onSubmit = useCallback(() => {
    saveCollection();
    notification.success({ message: 'Collection is sucessfully saved!' });
    onHideModal();
  }, [ onHideModal, saveCollection ]);

  return (
    <>
      <Button onClick={onShowModal}>Edit Collection</Button>
      <Modal
        title={`Edit Collection: ${collectionName}`}
        visible={isModalVisible}
        onCancel={onHideModal}
        onOk={onSubmit}>
        <React.Suspense
          fallback={<div><Spinner tip="Loading text editor..." /></div>}>
          <MonacoEditor
            height="40vh"
            language="yaml"
            options={{
              minimap: { enabled: false },
              occurrencesHighlight: false,
              readOnly: true,
            }}
            theme={ui.darkLight === DarkLight.Dark ? 'vs-dark' : 'vs-light'}
            value={yaml.dump(filters)}
          />
        </React.Suspense>
      </Modal>
    </>
  );
};

export default EditButton;
