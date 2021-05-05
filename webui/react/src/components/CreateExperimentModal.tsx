import { Alert, Button, Form, Input, Modal } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import yaml from 'js-yaml';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import { RawJson } from 'types';
import { clone } from 'utils/data';

import css from './CreateExperimentModal.module.scss';
import Spinner from './Spinner';

export enum CreateExperimentType {
  Fork = 'Fork',
  ContinueTrial = 'Continue Trial',
}

interface Props extends Omit<ModalFuncProps, 'type'> {
  config?: RawJson;
  error?: string;
  onCancel?: () => void;
  onOk?: (config: string) => void;
  type: CreateExperimentType;
}

const getExperimentName = (config: RawJson) => {
  return config.description || '';
};

const getMaxLengthType = (config: RawJson) => {
  return (Object.keys(config.searcher?.max_length || {}) || [])[0];
};

const getMaxLengthValue = (config: RawJson) => {
  const value = (Object.keys(config.searcher?.max_length || {}) || [])[1];
  return value ? parseInt(value) : undefined;
};

const MonacoEditor = React.lazy(() => import('react-monaco-editor'));

const CreateExperimentModal: React.FC<Props> = ({
  config = {},
  error,
  onCancel,
  onOk,
  type,
  ...props
}: Props) => {
  const [ form ] = Form.useForm();
  const [ isAdvancedMode, setIsAdvancedMode ] = useState(false);
  const [ configError, setConfigError ] = useState<string>();
  const [ localConfig, setLocalConfig ] = useState<string>('');

  const [ isFork, experimentName, maxLengthType, maxLengthValue ] = useMemo(() => {
    return [
      type === CreateExperimentType.Fork,
      getExperimentName(config),
      getMaxLengthType(config),
      getMaxLengthValue(config),
    ];
  }, [ config, type ]);

  const getConfigFromForm = useCallback(() => {
    const formValues = form.getFieldsValue();
    const newConfig = clone(config);

    if (formValues.description) {
      newConfig.description = formValues.description;
    }
    if (formValues.maxLength) {
      newConfig.searcher.max_length = { [maxLengthType]: parseInt(formValues.maxLength) };
    }

    return yaml.dump(newConfig);
  }, [ config, form, maxLengthType ]);

  const handleShowForm = useCallback(() => {
    setIsAdvancedMode(false);
  }, []);

  const handleShowEditor = useCallback(() => {
    setLocalConfig(getConfigFromForm());
    setIsAdvancedMode(true);
  }, [ getConfigFromForm ]);

  const handleEditorChange = useCallback((newConfigString: string) => {
    // Update config string upon each keystroke change.
    setLocalConfig(newConfigString);

    // Validate the yaml syntax by attempting to load it.
    try {
      const newConfig = (yaml.load(newConfigString) || {}) as RawJson;

      form.setFields([
        { name: 'description', value: getExperimentName(newConfig) },
        {
          name: 'maxLength',
          value: !isFork ? getMaxLengthValue(newConfig) : undefined,
        },
      ]);

      setConfigError(undefined);
    } catch (e) {
      setConfigError(e.message);
    }
  }, [ form, isFork ]);

  const handleOk = useCallback(async () => {
    if (!isAdvancedMode) {
      try {
        await form.validateFields();
        if (onOk) onOk(getConfigFromForm());
      } catch (e) {}
    } else {
      if (onOk) onOk(localConfig);
    }
  }, [ getConfigFromForm, form, isAdvancedMode, localConfig, onOk ]);

  const handleCancel = useCallback(() => {
    form.resetFields();
    if (onCancel) onCancel();
  }, [ form, onCancel ]);

  useEffect(() => {
    setLocalConfig(yaml.dump(config));
  }, [ config ]);

  return isAdvancedMode ? (
    <Modal
      bodyStyle={{ padding: 0 }}
      footer={(
        <>
          <Button onClick={handleShowForm}>Edit Form</Button>
          <Button disabled={!!configError} type="primary" onClick={handleOk}>{type}</Button>
        </>
      )}
      style={{ minWidth: '60rem' }}
      title={props.title}
      visible={props.visible}
      onCancel={handleCancel}>
      <React.Suspense fallback={<div className={css.loading}><Spinner /></div>}>
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
      </React.Suspense>
      {configError && <Alert className={css.error} message={configError} type="error" />}
      {error && <Alert className={css.error} message={error} type="error" />}
    </Modal>
  ) : (
    <Modal
      footer={<>
        <Button onClick={handleShowEditor}>Edit Full Config</Button>
        <Button type="primary" onClick={handleOk}>{type}</Button>
      </>}
      style={{ minWidth: '60rem' }}
      title={props.title}
      visible={props.visible}
      onCancel={handleCancel}>
      <Form
        form={form}
        initialValues={{
          description: experimentName,
          maxLength: !isFork ? maxLengthValue : undefined,
        }}
        labelCol={{ span: 8 }}
        name="basic">
        <Form.Item
          label="Experiment description"
          name="description"
          rules={[
            { message: 'Please provide a new experiment name.', required: true },
          ]}>
          <Input />
        </Form.Item>
        {!isFork && (
          <Form.Item
            label={`Max ${maxLengthType}`}
            name="maxLength"
            rules={[
              { message: 'Please provide a max length.', required: true },
            ]}>
            <Input type="number" />
          </Form.Item>
        )}
      </Form>
    </Modal>
  );
};

export default CreateExperimentModal;
