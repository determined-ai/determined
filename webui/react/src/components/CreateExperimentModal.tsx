import { Alert, Modal } from 'antd';
import yaml from 'js-yaml';
import React, { useCallback, useState } from 'react';
import MonacoEditor from 'react-monaco-editor';

import { routeAll } from 'routes';
import { forkExperiment } from 'services/api';

import css from './CreateExperimentModal.module.scss';

interface Props {
  title: string;
  okText: string;
  parentId: number;
  visible: boolean;
  config: string;
  onVisibleChange: (visible: boolean) => void;
  onConfigChange: (config: string) => void;
}

const CreateExperimentModal: React.FC<Props> = (
  { visible, config, onVisibleChange, onConfigChange, parentId, ...props }: Props,
) => {
  const [ configError, setConfigError ] = useState<string>();

  const editorOnChange = useCallback((newValue: string) => {
    onConfigChange(newValue);
    setConfigError(undefined);
  }, [ onConfigChange, setConfigError ]);

  const handleOk = async (): Promise<void> => {
    try {
      // Validate the yaml syntax by attempting to load it.
      yaml.safeLoad(config);
      const configId = await forkExperiment({ experimentConfig: config, parentId });
      onVisibleChange(false);
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
    onVisibleChange(false);
  };
  return <Modal
    bodyStyle={{
      padding: 0,
    }}
    className={css.configModal}
    okText={props.okText}
    style={{
      minWidth: '60rem',
    }}
    title={props.title}
    visible={visible}
    onCancel={handleCancel}
    onOk={handleOk}
  >
    <MonacoEditor
      height="40vh"
      language="yaml"
      options={{
        minimap: { enabled: false },
        scrollBeyondLastLine: false,
        selectOnLineNumbers: true,
      }}
      theme="vs-light"
      value={config}
      onChange={editorOnChange}
    />
    {configError &&
          <Alert className={css.error} message={configError} type="error" />
    }
  </Modal>;

};
export default CreateExperimentModal;
