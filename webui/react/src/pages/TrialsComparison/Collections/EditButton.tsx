import { Button, Modal, notification } from 'antd';
import yaml from 'js-yaml';
import React, { useCallback, useState } from 'react';

import MonacoEditor from 'components/MonacoEditor';
import Spinner from 'shared/components/Spinner';

import { TrialFilters } from './filters';

interface Props {
  collectionName: string;
  filters: TrialFilters;
  saveCollection: () => void;
}

const EditButton: React.FC<Props> = ({ saveCollection, filters, collectionName }) => {
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
        width={'clamp(280px, 416px, calc(100vw - 16px))'}
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
            value={yaml.dump(filters)}
          />
        </React.Suspense>
      </Modal>
    </>
  );
};

export default EditButton;
