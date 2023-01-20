import { Alert, Form, FormInstance, Input, ModalFuncProps } from 'antd';
import yaml from 'js-yaml';
import React, { useCallback, useEffect, useRef, useState } from 'react';

import { paths } from 'routes/utils';
import { createExperiment } from 'services/api';
import { V1LaunchWarning } from 'services/api-ts-sdk';
import Icon from 'shared/components/Icon/Icon';
import Spinner from 'shared/components/Spinner/Spinner';
import useModal, { ModalHooks as Hooks, ModalCloseReason } from 'shared/hooks/useModal/useModal';
import usePrevious from 'shared/hooks/usePrevious';
import { RawJson, ValueOf } from 'shared/types';
import { clone, isEqual } from 'shared/utils/data';
import { DetError, ErrorLevel, ErrorType, isDetError, isError } from 'shared/utils/error';
import { routeToReactUrl } from 'shared/utils/routes';
import { ExperimentBase, TrialHyperparameters, TrialItem } from 'types';
import handleError, { handleWarning } from 'utils/error';
import { trialHParamsToExperimentHParams } from 'utils/experiment';
import { upgradeConfig } from 'utils/experiment';

import css from './useModalExperimentCreate.module.scss';

export const CreateExperimentType = {
  ContinueTrial: 'Continue Trial',
  Fork: 'Fork',
} as const;

export type CreateExperimentType = ValueOf<typeof CreateExperimentType>;

interface Props {
  onClose?: () => void;
}

interface OpenProps {
  experiment: ExperimentBase;
  trial?: TrialItem;
  type: CreateExperimentType;
}

interface ModalState {
  config: RawJson;
  configError?: string;
  configString: string;
  error?: string;
  experiment?: ExperimentBase;
  isAdvancedMode: boolean;
  open: boolean;
  trial?: TrialItem;
  type: CreateExperimentType;
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
  workspaceName: string,
  projectName: string,
): RawJson => {
  const newConfig = clone(experimentConfig);
  return {
    ...newConfig,
    hyperparameters: trialHParamsToExperimentHParams(trialHparams),
    project: projectName,
    searcher: {
      max_length: experimentConfig.searcher.max_length,
      metric: experimentConfig.searcher.metric,
      name: 'single',
      smaller_is_better: experimentConfig.searcher.smaller_is_better,
      source_trial_id: trialId,
    },
    workspace: workspaceName,
  };
};

const MonacoEditor = React.lazy(() => import('components/MonacoEditor'));

const DEFAULT_MODAL_STATE = {
  config: {},
  configString: '',
  isAdvancedMode: false,
  open: false,
  type: CreateExperimentType.Fork,
};

const useModalExperimentCreate = ({ onClose }: Props = {}): ModalHooks => {
  const formRef = useRef<FormInstance>(null);
  const [registryCredentials, setRegistryCredentials] = useState<RawJson>();
  const [modalState, setModalState] = useState<ModalState>(DEFAULT_MODAL_STATE);
  const prevModalState = usePrevious(modalState, DEFAULT_MODAL_STATE);

  const handleModalClose = useCallback(() => {
    setModalState(DEFAULT_MODAL_STATE);
    onClose?.();
  }, [onClose]);

  const {
    modalClose,
    modalOpen: openOrUpdate,
    modalRef,
    ...modalHook
  } = useModal({
    onClose: handleModalClose,
    options: { rawCancel: true },
  });

  const handleFieldsChange = useCallback(() => {
    setModalState((prev) => {
      if (prev.error) return { ...prev, error: undefined };
      return prev;
    });
  }, []);

  const handleEditorChange = useCallback((newConfigString: string) => {
    // Update config string and config error upon each keystroke change.
    setModalState((prev) => {
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

  const handleCancel = useCallback(
    (close?: () => void) => {
      /**
       * 'close' is an indicator for if cancel button (show config) is clicked or not.
       * If cancel button (show config) is not clicked, 'close' is () => {}.
       */
      if (!close || close.toString() === 'function () {}') {
        modalClose(ModalCloseReason.Cancel);
      } else {
        setModalState((prev) => {
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
    },
    [modalClose],
  );

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

  const submitExperiment = useCallback(
    async (newConfig: string) => {
      const isFork = modalState.type === CreateExperimentType.Fork;
      if (!modalState.experiment || (!isFork && !modalState.trial)) return;

      try {
        const { experiment: newExperiment, warnings } = await createExperiment({
          activate: true,
          experimentConfig: newConfig,
          parentId: modalState.experiment.id,
          projectId: modalState.experiment.projectId,
        });
        const currentSlotsExceeded = warnings
          ? warnings.includes(V1LaunchWarning.CURRENTSLOTSEXCEEDED)
          : false;
        if (currentSlotsExceeded) {
          handleWarning({
            level: ErrorLevel.Warn,
            publicMessage:
              'The requested job requires more slots than currently available. You may need to increase cluster resources in order for the job to run.',
            publicSubject: 'Current Slots Exceeded',
            silent: false,
            type: ErrorType.Server,
          });
        }
        // Route to reload path to forcibly remount experiment page.
        const newPath = paths.experimentDetails(newExperiment.id);
        routeToReactUrl(paths.reload(newPath));
      } catch (e) {
        let errorMessage = `Unable to ${modalState.type.toLowerCase()} with the provided config.`;
        if (isError(e) && e.name === 'YAMLException') {
          errorMessage = e.message;
        } else if (isDetError(e)) {
          errorMessage = e.publicMessage || e.message;
        }

        setModalState((prev) => ({ ...prev, error: errorMessage }));

        // We throw an error to prevent the modal from closing.
        throw new DetError(errorMessage, { publicMessage: errorMessage, silent: true });
      }
    },
    [modalState],
  );

  const handleOk = useCallback(async () => {
    const error = modalState.error || modalState.configError;
    if (error) throw new Error(error);

    const { isAdvancedMode } = modalState;
    let userConfig, fullConfig;
    if (isAdvancedMode) {
      userConfig = (yaml.load(modalState.configString) || {}) as RawJson;
    } else {
      await formRef.current?.validateFields();
      userConfig = modalState.config;
    }

    // Add back `registry_auth` if it was stripped and no new `registry_auth` was provided.
    if (!userConfig?.environment?.registry_auth && registryCredentials) {
      const { environment, ...restConfig } = userConfig;
      fullConfig = {
        environment: { registry_auth: registryCredentials, ...environment },
        ...restConfig,
      };
    } else {
      fullConfig = userConfig;
    }

    const configString = isAdvancedMode ? yaml.dump(fullConfig) : getConfigFromForm(fullConfig);
    await submitExperiment(configString);
  }, [getConfigFromForm, modalState, submitExperiment, registryCredentials]);

  const getModalContent = useCallback(
    (state: ModalState) => {
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
              fallback={
                <div className={css.loading}>
                  <Spinner tip="Loading text editor..." />
                </div>
              }>
              <MonacoEditor height="40vh" value={configString} onChange={handleEditorChange} />
            </React.Suspense>
          )}
          <Form
            className={css.form}
            hidden={isAdvancedMode}
            initialValues={{
              maxLength: undefined,
              name: getExperimentName(config),
            }}
            labelCol={{ span: 8 }}
            name="basic"
            ref={formRef}
            onFieldsChange={handleFieldsChange}>
            <Form.Item
              label="Experiment name"
              name="name"
              rules={[{ message: 'Please provide a new experiment name.', required: true }]}>
              <Input />
            </Form.Item>
            {!isFork && (
              <Form.Item
                label={`Max ${getMaxLengthType(config) || 'length'}`}
                name="maxLength"
                rules={[
                  {
                    required: true,
                    validator: (rule, value) => {
                      let errorMessage = '';
                      if (!value) errorMessage = 'Please provide a max length.';
                      if (value < 1) errorMessage = 'Max length must be at least 1.';
                      return errorMessage ? Promise.reject(errorMessage) : Promise.resolve();
                    },
                  },
                ]}>
                <Input type="number" />
              </Form.Item>
            )}
          </Form>
        </>
      );
    },
    [handleEditorChange, handleFieldsChange],
  );

  const getModalProps = useCallback(
    (state: ModalState): ModalFuncProps | undefined => {
      const { experiment, isAdvancedMode, trial, type, open } = state;
      const isFork = type === CreateExperimentType.Fork;
      if (!experiment || (!isFork && !trial)) return;

      const titleLabel = isFork
        ? `Fork Experiment ${experiment.id}`
        : `Continue Trial ${trial?.id}`;
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
        open,
        title: (
          <div className={css.title}>
            <Icon name="fork" /> {titleLabel}
          </div>
        ),
        width: isAdvancedMode ? (isFork ? 760 : 1000) : undefined,
      };

      return props;
    },
    [getModalContent, handleCancel, handleOk],
  );

  const modalOpen = useCallback(({ experiment, trial, type }: OpenProps) => {
    const isFork = type === CreateExperimentType.Fork;
    let config = upgradeConfig(experiment.configRaw);

    if (!isFork && trial) {
      config = trialContinueConfig(
        config,
        trial.hyperparameters,
        trial.id,
        experiment.workspaceName,
        experiment.projectName,
      );
      config.description =
        `Continuation of trial ${trial.id}, experiment ${experiment.id}` +
        (config.description ? ` (${config.description})` : '');
    } else if (isFork) {
      if (config.description) config.description = `Fork of ${config.description}`;
    }

    const {
      environment: { registry_auth, ...restEnvironment },
      project: stripIt,
      workspace: stripItToo,
      ...restConfig
    } = config;
    setRegistryCredentials(registry_auth);

    const publicConfig = {
      environment: restEnvironment,
      project: experiment.projectName,
      workspace: experiment.workspaceName,
      ...restConfig,
    };
    setModalState((prev) => {
      const newModalState = {
        config: publicConfig,
        configString: yaml.dump(publicConfig),
        experiment,
        isAdvancedMode: false,
        open: true,
        trial,
        type,
      };
      return isEqual(prev, newModalState) ? prev : newModalState;
    });
  }, []);

  // Update the config from form when switching to advanced mode.
  useEffect(() => {
    if (modalState.isAdvancedMode !== prevModalState.isAdvancedMode && modalState.isAdvancedMode) {
      setModalState((prev) => ({ ...prev, configString: getConfigFromForm(prev.config) }));
    }
  }, [getConfigFromForm, modalState.isAdvancedMode, prevModalState]);

  /**
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal.
   */
  useEffect(() => {
    if (isEqual(modalState, prevModalState) || !modalState.open) return;
    openOrUpdate(getModalProps(modalState));
  }, [getModalProps, modalRef, modalState, openOrUpdate, prevModalState]);

  return { modalClose, modalOpen, modalRef, ...modalHook };
};

export default useModalExperimentCreate;
