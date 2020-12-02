import { Alert, Modal } from 'antd';
import yaml from 'js-yaml';
import React, { useCallback, useState } from 'react';
import MonacoEditor from 'react-monaco-editor';

import { routeAll } from 'routes/utils';
import { createExperiment } from 'services/api';

import css from './CreateExperimentModal.module.scss';

interface Props {
  config: string;
  error?: string;
  okText: string;
  onCancel?: () => void;
  onConfigChange: (config: string) => void;
  onVisibleChange: (visible: boolean) => void;
  parentId: number; // parent experiment ID.
  title: string;
  visible: boolean;
}

const CreateExperimentModal: React.FC<Props> = (
  { visible, config, onVisibleChange, onConfigChange, parentId, error, ...props }: Props,
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
      const { id: configId } = await createExperiment({ experimentConfig: config, parentId });
      onVisibleChange(false);
      routeAll(`/experiments/${configId}`);
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
    props.onCancel && props.onCancel();
    onVisibleChange(false);
  };
  return <Modal
    bodyStyle={{ padding: 0 }}
    className={css.configModal}
    okText={props.okText}
    style={{ minWidth: '60rem' }}
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
    {error &&
          <Alert className={css.error} message={error} type="error" />
    }
  </Modal>;

};
export default CreateExperimentModal;
