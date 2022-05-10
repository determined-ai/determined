import { Alert, Form, FormInstance, Input, ModalFuncProps } from 'antd';
import yaml from 'js-yaml';
import React, { useCallback, useEffect, useRef, useState } from 'react';

import Icon from 'components/Icon';
import Spinner from 'components/Spinner';
import usePrevious from 'hooks/usePrevious';
import { paths } from 'routes/utils';
import { createExperiment } from 'services/api';
import { clone, isEqual } from 'shared/utils/data';
import {
  ExperimentBase,
  TrialDetails,
  TrialHyperparameters,
} from 'types';
import handleError from 'utils/error';
import { trialHParamsToExperimentHParams } from 'utils/experiment';
import { upgradeConfig } from 'utils/experiment';

import { RawJson } from '../../shared/types';
import { DetError, isDetError, isError } from '../../shared/utils/error';
import { routeToReactUrl } from '../../shared/utils/routes';

import useModal, { ModalHooks as Hooks, ModalCloseReason } from './useModal';
import css from './useModalExperimentCreate.module.scss';

export enum CreateExperimentType {
  Fork = 'Fork',
  ContinueTrial = 'Continue Trial',
}

interface Props {
  onClose?: () => void;
}

interface OpenProps {
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

interface ModalHooks extends Omit<Hooks, 'modalOpen'> {
  modalOpen: (openProps: OpenProps) => void;
}

const getExperimentName = (config: RawJson) => {
  return config.name || '';
};

// For unitless searchers, this will return undefined.
const getMaxLengthType = (config: RawJson) => {
  return (Object.keys(config.searcher?.max_length || {}) || [])[0];
};

const getMaxLengthValue = (config: RawJson) => {
  const value = (Object.keys(config.searcher?.max_length || {}) || [])[1];
  return value ? parseInt(value) : parseInt(config.searcher?.max_length);
};

const trialContinueConfig = (
  experimentConfig: RawJson,
  trialHparams: TrialHyperparameters,
  trialId: number,
): RawJson => {
  const newConfig = clone(experimentConfig);
  return {
    ...newConfig,
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

const DEFAULT_MODAL_STATE = {
  config: {},
  configString: '',
  isAdvancedMode: false,
  type: CreateExperimentType.Fork,
  visible: false,
};

const useModalExperimentCreate = (props?: Props): ModalHooks => {
  const formRef = useRef<FormInstance>(null);
  const [ registryCredentials, setRegistryCredentials ] = useState<RawJson>();
  const [ modalState, setModalState ] = useState<ModalState>(DEFAULT_MODAL_STATE);
  const prevModalState = usePrevious(modalState, DEFAULT_MODAL_STATE);

  const handleModalClose = useCallback(() => {
    setModalState(DEFAULT_MODAL_STATE);
    props?.onClose?.();
  }, [ props ]);

  const { modalClose, modalOpen: openOrUpdate, modalRef } = useModal({
    onClose: handleModalClose,
    options: { rawCancel: true },
  });

  const handleEditorChange = useCallback((newConfigString: string) => {
    // Update config string and config error upon each keystroke change.
    setModalState(prev => {
      if (!prev) return prev;

      const newModalState = { ...prev, configString: newConfigString };

      // Validate the yaml syntax by attempting to load it.
      try {
        yaml.load(newConfigString);
        newModalState.configError = undefined;
      } catch (e) {
        if (isError(e)) newModalState.configError = e.message;
      }

      return newModalState;
    });
  }, []);

  const handleCancel = useCallback((close) => {
    if (close?.triggerCancel) {
      modalClose(ModalCloseReason.Cancel);
    } else {
      setModalState(prev => {
        if (!prev) return prev;

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
          } catch (e) {
            handleError(e, { publicMessage: 'failed to load previous yaml config' });
          }
        }

        return {
          ...prev,
          configError: undefined,
          error: undefined,
          isAdvancedMode: !prev.isAdvancedMode,
        };
      });
    }
  }, [ modalClose ]);

  const getConfigFromForm = useCallback((config: RawJson) => {
    if (!formRef.current) return yaml.dump(config);

    const formValues = formRef.current.getFieldsValue();
    const newConfig = clone(config);

    if (formValues.name) {
      newConfig.name = formValues.name;
    }
    if (formValues.maxLength) {
      const maxLengthType = getMaxLengthType(newConfig);
      if (maxLengthType === undefined) {
        // Unitless searcher config.
        newConfig.searcher.max_length = parseInt(formValues.maxLength);
      } else {
        newConfig.searcher.max_length = { [maxLengthType]: parseInt(formValues.maxLength) };
      }
    }

    return yaml.dump(newConfig);
  }, []);

  const submitExperiment = useCallback(async (newConfig: string) => {
    const isFork = modalState.type === CreateExperimentType.Fork;
    if (!modalState.experiment || (!isFork && !modalState.trial)) return;

    try {
      const { id: newExperimentId } = await createExperiment({
        activate: true,
        experimentConfig: newConfig,
        parentId: modalState.experiment.id,
      });

      // Route to reload path to forcibly remount experiment page.
      const newPath = paths.experimentDetails(newExperimentId);
      routeToReactUrl(paths.reload(newPath));
    } catch (e) {
      let errorMessage = `Unable to ${modalState.type.toLowerCase()} with the provided config.`;
      if (isError(e) && e.name === 'YAMLException') {
        errorMessage = e.message;
      } else if (isDetError(e)) {
        errorMessage = e.publicMessage || e.message;
      }

      setModalState(prev => ({ ...prev, error: errorMessage }));

      // We throw an error to prevent the modal from closing.
      throw new DetError(errorMessage, { publicMessage: errorMessage, silent: true });
    }
  }, [ modalState ]);

  const handleOk = useCallback(async () => {
    const error = modalState.error || modalState.configError;
    if (error) throw new Error(error);

    /**
     * add back registry_auth if it was stripped
     * and no new registry_auth was provided
     */
    let userConfig, fullConfig;
    if (!modalState.isAdvancedMode) {
      await formRef.current?.validateFields();
      userConfig = modalState.config;
    } else {
      userConfig = (yaml.load(modalState.configString) || {}) as RawJson;
    }
    if(!userConfig?.environment?.registry_auth && registryCredentials) {
      const { environment, ...restConfig } = userConfig;
      fullConfig = {
        environment: { registry_auth: registryCredentials, ...environment },
        ...restConfig,
      };
    } else {
      fullConfig = userConfig;
    }

    const configString = getConfigFromForm(fullConfig);
    await submitExperiment(configString);
  }, [ getConfigFromForm, modalState, submitExperiment, registryCredentials ]);

  const getModalContent = useCallback((state: ModalState) => {
    const { config, configError, configString, error, isAdvancedMode, type } = state;
    const isFork = type === CreateExperimentType.Fork;

    // We always render the form regardless of mode to provide a reference to it.
    return (
      <>
        {error && <Alert className={css.error} message={error} type="error" />}
        {configError && isAdvancedMode && (
          <Alert className={css.error} message={configError} type="error" />
        )}
        {isAdvancedMode && (
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
          hidden={isAdvancedMode}
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
              label={`Max ${getMaxLengthType(config) || 'length'}`}
              name="maxLength"
              rules={[ { message: 'Please provide a max length.', required: true } ]}>
              <Input type="number" />
            </Form.Item>
          )}
        </Form>
      </>
    );
  }, [ handleEditorChange ]);

  const getModalProps = useCallback((state: ModalState): ModalFuncProps | undefined => {
    const { experiment, isAdvancedMode, trial, type, visible } = state;
    const isFork = type === CreateExperimentType.Fork;
    if (!experiment || (!isFork && !trial)) return;

    const titleLabel = isFork ? `Fork Experiment ${experiment.id}` : `Continue Trial ${trial?.id}`;
    const props = {
      bodyStyle: { padding: 0 },
      cancelText: isAdvancedMode ? 'Show Simple Config' : 'Show Full Config',
      className: css.base,
      closable: true,
      content: getModalContent(state),
      icon: null,
      maskClosable: true,
      okText: type,
      onCancel: handleCancel,
      onOk: handleOk,
      title: (
        <div className={css.title}>
          <Icon name="fork" /> {titleLabel}
        </div>
      ),
      visible,
      width: isAdvancedMode ? (isFork ? 760 : 1000) : undefined,
    };

    return props;
  }, [ getModalContent, handleCancel, handleOk ]);

  const modalOpen = useCallback(({ experiment, trial, type }: OpenProps) => {
    const isFork = type === CreateExperimentType.Fork;
    let config = upgradeConfig(experiment.configRaw);

    if (!isFork && trial) {
      config = trialContinueConfig(config, trial.hyperparameters, trial.id);
      config.description = `Continuation of trial ${trial.id}, experiment ${experiment.id}` +
        (config.description ? ` (${config.description})` : '');
    } else if (isFork) {
      if (config.description) config.description = `Fork of ${config.description}`;
    }

    const { environment: { registry_auth, ...restEnvironment }, ...restConfig } = config;
    setRegistryCredentials(registry_auth);
    const publicConfig = { environment: restEnvironment, ...restConfig };

    setModalState(prev => {
      const newModalState = {
        config: publicConfig,
        configString: yaml.dump(publicConfig),
        experiment,
        isAdvancedMode: false,
        trial,
        type,
        visible: true,
      };
      return isEqual(prev, newModalState) ? prev : newModalState;
    });
  }, []);

  // Update the config from form when switching to advanced mode.
  useEffect(() => {
    if (modalState.isAdvancedMode !== prevModalState.isAdvancedMode && modalState.isAdvancedMode) {
      setModalState(prev => ({ ...prev, configString: getConfigFromForm(prev.config) }));
    }
  }, [ getConfigFromForm, modalState.isAdvancedMode, prevModalState ]);

  /*
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal
   */
  useEffect(() => {
    if (modalState === prevModalState || !modalState.visible) return;

    const modalProps = getModalProps(modalState);
    openOrUpdate(modalProps);
  }, [ getModalProps, modalRef, modalState, openOrUpdate, prevModalState ]);

  return { modalClose, modalOpen, modalRef };
};

export default useModalExperimentCreate;
