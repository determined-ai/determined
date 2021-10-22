import { Alert, Form, FormInstance, Input, Modal, ModalFuncProps } from 'antd';
import { ModalFunc } from 'antd/es/modal/confirm';
import yaml from 'js-yaml';
import React, { useCallback, useEffect, useRef, useState } from 'react';

import Icon from 'components/Icon';
import Spinner from 'components/Spinner';
import { paths, routeToReactUrl } from 'routes/utils';
import { createExperiment } from 'services/api';
import { ExperimentBase, RawJson, TrialDetails, TrialHyperparameters } from 'types';
import { clone } from 'utils/data';
import { trialHParamsToExperimentHParams } from 'utils/experiment';
import { upgradeConfig } from 'utils/types';

import css from './useCreateExperimentModal.module.scss';
import usePrevious from './usePrevious';

export enum CreateExperimentType {
  Fork = 'Fork',
  ContinueTrial = 'Continue Trial',
}

interface ShowProps {
  experiment: ExperimentBase;
  trial?: TrialDetails;
  type: CreateExperimentType;
}

interface ModalState {
  config: RawJson;
  configError?: string;
  configString: string;
  error?: string;
  experiment?: ExperimentBase;
  isAdvancedMode: boolean;
  trial?: TrialDetails;
  type: CreateExperimentType;
  visible: boolean;
}

interface ModalHooks {
  showModal: (props: ShowProps) => void;
}

const getExperimentName = (config: RawJson) => {
  return config.name || '';
};

const getMaxLengthType = (config: RawJson) => {
  return (Object.keys(config.searcher?.max_length || {}) || [])[0];
};

const getMaxLengthValue = (config: RawJson) => {
  const value = (Object.keys(config.searcher?.max_length || {}) || [])[1];
  return value ? parseInt(value) : undefined;
};

const trialContinueConfig = (
  experimentConfig: RawJson,
  trialHparams: TrialHyperparameters,
  trialId: number,
): RawJson => {
  return {
    ...experimentConfig,
    hyperparameters: trialHParamsToExperimentHParams(trialHparams),
    searcher: {
      max_length: experimentConfig.searcher.max_length,
      metric: experimentConfig.searcher.metric,
      name: 'single',
      smaller_is_better: experimentConfig.searcher.smaller_is_better,
      source_trial_id: trialId,
    },
  };
};

const MonacoEditor = React.lazy(() => import('components/MonacoEditor'));

const useCreateExperimentModal = (): ModalHooks => {
  const modalRef = useRef<ReturnType<ModalFunc>>();
  const formRef = useRef<FormInstance>(null);
  const [ modalState, setModalState ] = useState<ModalState>({
    config: {},
    configString: '',
    isAdvancedMode: false,
    type: CreateExperimentType.Fork,
    visible: false,
  });
  const prevIsAdvancedMode = usePrevious(modalState.isAdvancedMode, false);

  const showModal = useCallback(({ experiment, trial, type }: ShowProps) => {
    const isFork = type === CreateExperimentType.Fork;

    let config = clone(experiment.configRaw);
    if (!isFork && trial) {
      config = trialContinueConfig(config, trial.hyperparameters, trial.id);
      config.description = `Continuation of trial ${trial.id}, experiment ${experiment.id}` +
        (config.description ? ` (${config.description})` : '');
    } else if (!isFork) {
      if (config.description) config.description = `Fork of ${config.description}`;
    }
    upgradeConfig(config);

    setModalState({
      config,
      configString: yaml.dump(config),
      experiment,
      isAdvancedMode: false,
      trial,
      type,
      visible: true,
    });
  }, []);

  const closeModal = useCallback(() => {
    if (!modalRef.current) return;
    modalRef.current.destroy();
    modalRef.current = undefined;
  }, []);

  const getConfigFromForm = useCallback((config: RawJson) => {
    if (!formRef.current) return yaml.dump(config);

    const formValues = formRef.current.getFieldsValue();
    const newConfig = clone(config);

    if (formValues.name) {
      newConfig.name = formValues.name;
    }
    if (formValues.maxLength) {
      const maxLengthType = getMaxLengthType(newConfig);
      newConfig.searcher.max_length = { [maxLengthType]: parseInt(formValues.maxLength) };
    }

    return yaml.dump(newConfig);
  }, []);

  const submitExperiment = useCallback(async (newConfig: string) => {
    const isFork = modalState.type === CreateExperimentType.Fork;
    if (!modalState.experiment || (!isFork && !modalState.trial)) return;

    try {
      const { id: newExperimentId } = await createExperiment({
        experimentConfig: newConfig,
        parentId: modalState.experiment.id,
      });

      closeModal();

      // Route to reload path to forcibly remount experiment page.
      const newPath = paths.experimentDetails(newExperimentId);
      routeToReactUrl(paths.reload(newPath));
    } catch (e) {
      let errorMessage = `Unable to ${modalState.type.toLowerCase()} with the provided config.`;
      if (e.name === 'YAMLException') {
        errorMessage = e.message;
      } else if (e.response?.data?.message) {
        errorMessage = e.response.data.message;
      } else if (e.json) {
        const errorJSON = await e.json();
        errorMessage = errorJSON.error?.error;
      }

      setModalState(prev => ({ ...prev, error: errorMessage }));
    }
  }, [ closeModal, modalState ]);

  const handleCancel = useCallback((close) => {
    if (!modalRef.current) return;

    if (close.triggerCancel) {
      closeModal();
    } else {
      setModalState(prev => {
        if (prev.isAdvancedMode && formRef.current) {
          try {
            const newConfig = (yaml.load(prev.configString) || {}) as RawJson;
            const isFork = prev.type === CreateExperimentType.Fork;

            formRef.current.setFields([
              { name: 'name', value: getExperimentName(newConfig) },
              {
                name: 'maxLength',
                value: !isFork ? getMaxLengthValue(newConfig) : undefined,
              },
            ]);

            formRef.current.validateFields();
          } catch (e) {}
        }

        return {
          ...prev,
          configError: !prev.isAdvancedMode ? undefined : prev.configError,
          isAdvancedMode: !prev.isAdvancedMode,
        };
      });
    }
  }, [ closeModal ]);

  const handleOk = useCallback(async () => {
    if (!formRef.current || !modalRef.current) return Promise.reject();
    if (!!modalState.error || !!modalState.configError) return Promise.reject();

    try {
      let configString;
      if (!modalState.isAdvancedMode) {
        await formRef.current.validateFields();
        configString = getConfigFromForm(modalState.config);
      } else {
        configString = modalState.configString;
      }
      await submitExperiment(configString);
    } catch (e) {}

    return Promise.reject();
  }, [ getConfigFromForm, modalState, submitExperiment ]);

  const handleEditorChange = useCallback((newConfigString: string) => {
    // Update config string and config error upon each keystroke change.
    setModalState(prev => {
      let configError = undefined;

      // Validate the yaml syntax by attempting to load it.
      try {
        yaml.load(newConfigString);
      } catch (e) {
        configError = e.message;
      }

      return { ...prev, configError, configString: newConfigString };
    });
  }, []);

  const generateModalContent = useCallback((state: ModalState): React.ReactNode => {
    const { config, configError, configString, error, type } = state;
    const isFork = type === CreateExperimentType.Fork;

    // We always render the form regardless of mode to provide a reference to it.
    return (
      <>
        {error && <Alert className={css.error} message={error} type="error" />}
        {configError && state.isAdvancedMode && (
          <Alert className={css.error} message={configError} type="error" />
        )}
        {state.isAdvancedMode && (
          <React.Suspense
            fallback={<div className={css.loading}><Spinner tip="Loading text editor..." /></div>}>
            <MonacoEditor
              height="40vh"
              value={configString}
              onChange={handleEditorChange}
            />
          </React.Suspense>
        )}
        <Form
          className={css.form}
          hidden={state.isAdvancedMode}
          initialValues={{
            maxLength: !isFork ? getMaxLengthValue(config) : undefined,
            name: getExperimentName(config),
          }}
          labelCol={{ span: 8 }}
          name="basic"
          ref={formRef}>
          <Form.Item
            label="Experiment name"
            name="name"
            rules={[ { message: 'Please provide a new experiment name.', required: true } ]}>
            <Input />
          </Form.Item>
          {!isFork && (
            <Form.Item
              label={`Max ${getMaxLengthType(config)}`}
              name="maxLength"
              rules={[ { message: 'Please provide a max length.', required: true } ]}>
              <Input type="number" />
            </Form.Item>
          )}
        </Form>
      </>
    );
  }, [ handleEditorChange ]);

  const generateModalProps = useCallback((state: ModalState): Partial<ModalFuncProps> => {
    const { experiment, trial, type } = state;
    const isFork = type === CreateExperimentType.Fork;
    if (!experiment || (!isFork && !trial)) return {};

    const titleLabel = (!isFork && trial)
      ? `Continue Trial ${trial.id}` : `Fork Experiment ${experiment.id}`;
    const modalProps = {
      bodyStyle: { padding: 0 },
      cancelText: state.isAdvancedMode ? 'Show Simple Config' : 'Show Full Config',
      className: css.base,
      closable: true,
      content: generateModalContent(state),
      icon: null,
      maskClosable: true,
      okText: type,
      onCancel: handleCancel,
      onOk: handleOk,
      style: { minWidth: 600 },
      title: (
        <div className={css.title}>
          <Icon name="fork" /> {titleLabel}
        </div>
      ),
    };

    return modalProps;
  }, [ generateModalContent, handleCancel, handleOk ]);

  // Detect modal state change and update.
  useEffect(() => {
    if (!modalState.visible) return;

    const modalProps = generateModalProps(modalState);
    if (modalRef.current) {
      modalRef.current.update(prev => ({ ...prev, ...modalProps }));
    } else {
      modalRef.current = Modal.confirm(modalProps);
    }
  }, [ generateModalProps, modalState ]);

  // Update the config from form when switching to advanced mode.
  useEffect(() => {
    if (modalState.isAdvancedMode !== prevIsAdvancedMode && modalState.isAdvancedMode) {
      setModalState(prev => ({ ...prev, configString: getConfigFromForm(prev.config) }));
    }
  }, [ getConfigFromForm, modalState.isAdvancedMode, prevIsAdvancedMode ]);

  // When the component using the hook unmounts, remove the modal automatically.
  useEffect(() => {
    return () => {
      if (!modalRef.current) return;
      modalRef.current.destroy();
      modalRef.current = undefined;
    };
  }, []);

  return { showModal };
};

export default useCreateExperimentModal;
