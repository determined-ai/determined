import { Alert, Modal } from 'antd';
import yaml from 'js-yaml';
import React, { useCallback, useState } from 'react';
import MonacoEditor from 'react-monaco-editor';

import { routeAll } from 'routes';
import { forkExperiment } from 'services/api';

import css from './CreateExperimentModal.module.scss';

interface Props {
  visible: boolean;
  setVisible: (arg0: boolean) => void; // or on finish
  title: string;
  configValue: string;
  setConfigValue: (arg0: string) => void;
  parentId: number;
}

const CreateExperimentModal: React.FC<Props> = (
  { visible, configValue, setConfigValue, parentId, setVisible }: Props,
) => {
  const [ configError, setConfigError ] = useState<string>();

  const editorOnChange = useCallback((newValue) => {
    setConfigValue(newValue);
    setConfigError(undefined);
  }, [ setConfigError, setConfigValue ]);

  const monacoOpts = {
    minimap: { enabled: false },
    selectOnLineNumbers: true,
  };

  const handleOk = async (): Promise<void> => {
    try {
      // Validate the yaml syntax by attempting to load it.
      yaml.safeLoad(configValue);
      const configId = await forkExperiment({ experimentConfig: configValue, parentId });
      setVisible(false);
      routeAll(`/det/experiments/${configId}`);
    } catch (e) {
      let errorMessage = 'Failed to config using the provided config.';
      if (e.name === 'YAMLException') {
        errorMessage = e.message;
      } else if (e.response?.data?.message) {
        errorMessage = e.response.data.message;
      }
      setConfigError(errorMessage);
    }
  };

  const handleCancel = (): void => {
    setVisible(false);
  };
  return <Modal
    bodyStyle={{
      padding: 0,
    }}
    className={css.configModal}
    okText="Fork"
    style={{
      minWidth: '60rem',
    }}
    title={`Config Experiment ${parentId}`}
    visible={visible}
    onCancel={handleCancel}
    onOk={handleOk}
  >
    <MonacoEditor
      height="40vh"
      language="yaml"
      options={monacoOpts}
      theme="vs-light"
      value={configValue}
      onChange={editorOnChange}
    />
    {configError &&
          <Alert className={css.error} message={configError} type="error" />
    }
  </Modal>;

};
export default CreateExperimentModal;
