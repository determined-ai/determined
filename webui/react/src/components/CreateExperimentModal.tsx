import { Alert, Modal } from 'antd';
import yaml from 'js-yaml';
import React, { SetStateAction, useCallback, useState } from 'react';
import MonacoEditor from 'react-monaco-editor';

import { routeAll } from 'routes';
import { forkExperiment } from 'services/api';

import css from './CreateExperimentModal.module.scss';

interface RWState {
  visible: boolean;
  config: string;
}

interface Props {
  title: string;
  parentId: number;
  state: RWState;
  setState: (arg0: SetStateAction<RWState>) => void;
}

const CreateExperimentModal: React.FC<Props> = (
  { state, setState, parentId }: Props,
) => {
  const [ configError, setConfigError ] = useState<string>();

  const editorOnChange = useCallback((newValue: string) => {
    setState((existingState: RWState) => ({ ...existingState, config: newValue }));
    setConfigError(undefined);
  }, [ setState, setConfigError ]);

  const monacoOpts = {
    minimap: { enabled: false },
    selectOnLineNumbers: true,
  };

  const handleOk = async (): Promise<void> => {
    try {
      // Validate the yaml syntax by attempting to load it.
      yaml.safeLoad(state.config);
      const configId = await forkExperiment({ experimentConfig: state.config, parentId });
      setState(existingState => ({ ...existingState, visible: false }));
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
    setState(existingState => ({ ...existingState, visible: false }));
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
    visible={state.visible}
    onCancel={handleCancel}
    onOk={handleOk}
  >
    <MonacoEditor
      height="40vh"
      language="yaml"
      options={monacoOpts}
      theme="vs-light"
      value={state.config}
      onChange={editorOnChange}
    />
    {configError &&
          <Alert className={css.error} message={configError} type="error" />
    }
  </Modal>;

};
export default CreateExperimentModal;
