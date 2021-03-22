import { Alert, Modal } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import yaml from 'js-yaml';
import React, { useCallback, useEffect, useState } from 'react';
import MonacoEditor from 'react-monaco-editor';

import css from './CreateExperimentModal.module.scss';

interface Props extends ModalFuncProps {
  config: string;
  error?: string;
  onCancel?: () => void;
  onOk?: (config: string) => void;
}

const CreateExperimentModalX: React.FC<Props> = ({
  config,
  error,
  onCancel,
  onOk,
  ...props
}: Props) => {
  const [ localConfig, setLocalConfig ] = useState<string>(config);
  const [ configError, setConfigError ] = useState<string>();

  const handleEditorChange = useCallback((newConfig: string) => {
    setLocalConfig(newConfig);

    // Validate the yaml syntax by attempting to load it.
    try {
      yaml.safeLoad(newConfig);
      setConfigError(undefined);
    } catch (e) {
      setConfigError(e.message);
    }
  }, []);

  const handleOk = useCallback(() => {
    if (onOk) onOk(localConfig);
  }, [ localConfig, onOk ]);

  const handleCancel = useCallback(() => {
    if (onCancel) onCancel();
  }, [ onCancel ]);

  useEffect(() => {
    setLocalConfig(config);
  }, [ config ]);

  return (
    <Modal
      bodyStyle={{ padding: 0 }}
      okButtonProps={{ disabled: !!configError }}
      okText={props.okText}
      style={{ minWidth: '60rem' }}
      title={props.title}
      visible={props.visible}
      onCancel={handleCancel}
      onOk={handleOk}>
      <MonacoEditor
        height="40vh"
        language="yaml"
        options={{
          minimap: { enabled: false },
          scrollBeyondLastLine: false,
          selectOnLineNumbers: true,
        }}
        theme="vs-light"
        value={localConfig}
        onChange={handleEditorChange}
      />
      {configError && <Alert className={css.error} message={configError} type="error" />}
      {error && <Alert className={css.error} message={error} type="error" />}
    </Modal>
  );
};

export default CreateExperimentModalX;
