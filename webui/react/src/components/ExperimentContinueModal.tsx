import Alert from 'hew/Alert';
import Button from 'hew/Button';
import Form, { hasErrors } from 'hew/Form';
import Icon from 'hew/Icon';
import Input from 'hew/Input';
import InputNumber from 'hew/InputNumber';
import { Modal } from 'hew/Modal';
import Row from 'hew/Row';
import Spinner from 'hew/Spinner';
import { Body } from 'hew/Typography';
import { Loaded } from 'hew/utils/loadable';
import yaml from 'js-yaml';
import _ from 'lodash';
import React, { useCallback, useEffect, useId, useMemo, useState } from 'react';

import useFeature from 'hooks/useFeature';
import { paths } from 'routes/utils';
import { continueExperiment, createExperiment } from 'services/api';
import { V1LaunchWarning } from 'services/api-ts-sdk';
import { ExperimentBase, RawJson, RunState, TrialItem, ValueOf } from 'types';
import handleError, {
  DetError,
  ErrorLevel,
  ErrorType,
  handleWarning,
  isDetError,
  isError,
} from 'utils/error';
import {
  FULL_CONFIG_BUTTON_TEXT,
  getExperimentName,
  getMaxLengthType,
  getMaxLengthValue,
  SIMPLE_CONFIG_BUTTON_TEXT,
  trialContinueConfig,
  upgradeConfig,
} from 'utils/experiment';
import { routeToReactUrl } from 'utils/routes';
import { capitalize, capitalizeWord } from 'utils/string';

import css from './ExperimentContinueModal.module.scss';
const FORM_ID = 'continue-experiment-form';

export const ContinueExperimentType = {
  Continue: 'CONTINUE',
  Reactivate: 'REACTIVATE',
} as const;

export type ContinueExperimentType = ValueOf<typeof ContinueExperimentType>;

const ExperimentCopyMapping: Record<ContinueExperimentType, string> = {
  [ContinueExperimentType.Continue]: 'Continue Trial in New Experiment',
  [ContinueExperimentType.Reactivate]: 'Reactivate Current Trial',
} satisfies Record<ContinueExperimentType, string>;

const SearchCopyMapping: Record<ContinueExperimentType, string> = {
  [ContinueExperimentType.Continue]: 'Continue as New Run',
  [ContinueExperimentType.Reactivate]: 'Reactivate Current Run',
};

type EntityCopyMap = {
  experiment: string;
  trial: string;
};

const experimentEntityCopyMap: EntityCopyMap = {
  experiment: 'experiment',
  trial: 'trial',
};

const flatRunsEntityCopyMap: EntityCopyMap = {
  experiment: 'search',
  trial: 'run',
};

const EXPERIMENT_NAME = 'name';
const MAX_LENGTH = 'maxLength';
const ADDITIONAL_LENGTH = 'additionalLength';

export interface Props {
  onClose?: () => void;
  experiment: ExperimentBase;
  trial?: TrialItem;
  type: ContinueExperimentType;
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
  type: ContinueExperimentType;
}

const CodeEditor = React.lazy(() => import('hew/CodeEditor'));

const DEFAULT_MODAL_STATE = {
  config: {},
  configString: '',
  isAdvancedMode: false,
  open: false,
  type: ContinueExperimentType.Continue,
};

const ExperimentContinueModalComponent = ({
  onClose,
  experiment,
  trial,
  type,
}: Props): JSX.Element => {
  const idPrefix = useId();
  const [registryCredentials, setRegistryCredentials] = useState<RawJson>();
  const [modalState, setModalState] = useState<ModalState>(DEFAULT_MODAL_STATE);
  const [disabled, setDisabled] = useState<boolean>(true);
  const [originalConfig, setOriginalConfig] = useState(experiment.configRaw);
  const f_flat_runs = useFeature().isOn('flat_runs');

  const isReactivate = type === ContinueExperimentType.Reactivate;
  const [actionCopyMap, entityCopyMap] = f_flat_runs
    ? [SearchCopyMapping, flatRunsEntityCopyMap]
    : [ExperimentCopyMapping, experimentEntityCopyMap];
  const actionCopy = actionCopyMap[modalState.type];

  useEffect(() => setOriginalConfig(experiment.configRaw), [experiment]);

  const requiredFields = useMemo(() => [EXPERIMENT_NAME, MAX_LENGTH], []);

  const handleModalClose = () => {
    setModalState(DEFAULT_MODAL_STATE);
    onClose?.();
  };

  const [form] = Form.useForm();

  const handleFieldsChange = () => {
    setModalState((prev) => {
      if (prev.error) return { ...prev, error: undefined };
      const values = form.getFieldsValue();
      if (!prev.isAdvancedMode) {
        prev.config.name = values[EXPERIMENT_NAME];
      }
      if (!isReactivate && values[MAX_LENGTH]) {
        const maxLengthType = getMaxLengthType(prev.config);
        if (maxLengthType) {
          prev.config.searcher.max_length[maxLengthType] = parseInt(values[MAX_LENGTH]);
        } else {
          prev.config.searcher.max_length = parseInt(values[MAX_LENGTH]);
        }
      }
      if (isReactivate && values[ADDITIONAL_LENGTH] && parseInt(values[ADDITIONAL_LENGTH]) >= 0) {
        const maxLengthType = getMaxLengthType(prev.config);
        if (maxLengthType) {
          prev.config.searcher.max_length[maxLengthType] =
            originalConfig.searcher.max_length[maxLengthType] + parseInt(values[ADDITIONAL_LENGTH]);
        } else {
          prev.config.searcher.max_length =
            originalConfig.searcher.max_length + parseInt(values[ADDITIONAL_LENGTH]);
        }
      }
      prev.configString = yaml.dump(prev.config);
      return prev;
    });

    const hasError = hasErrors(form);
    const values = form.getFieldsValue();
    const missingRequiredFields = Object.entries(values).some(([key, value]) => {
      return requiredFields.includes(key) && !value;
    });
    setDisabled(hasError || missingRequiredFields);
  };

  const handleEditorChange = useCallback((newConfigString: string) => {
    // Update config string and config error upon each keystroke change.
    setModalState((prev) => {
      if (!prev) return prev;

      const newModalState = { ...prev, configString: newConfigString };

      // Validate the yaml syntax by attempting to load it.
      try {
        newModalState.config = (yaml.load(newConfigString) || {}) as RawJson;
        newModalState.configError = undefined;
        newModalState.error = undefined;
      } catch (e) {
        if (isError(e)) newModalState.configError = e.message;
      }
      setDisabled(newModalState.configError !== undefined);
      return newModalState;
    });
  }, []);

  const toggleMode = useCallback(async () => {
    setModalState((prev) => {
      if (!prev) return prev;

      return {
        ...prev,
        configError: undefined,
        error: undefined,
        isAdvancedMode: !prev.isAdvancedMode,
      };
    });
    // avoid calling form.setFields inside setModalState:
    if (modalState.isAdvancedMode && form) {
      try {
        const newConfig = (yaml.load(modalState.configString) || {}) as RawJson;
        const maxLengthType = getMaxLengthType(newConfig);
        const isReactivate = modalState.type === ContinueExperimentType.Reactivate;
        const originalLength = maxLengthType
          ? originalConfig.searcher.max_length[maxLengthType]
          : originalConfig.searcher.max_length;
        let additionalLength;
        try {
          const newLength = maxLengthType
            ? newConfig.searcher.max_length[maxLengthType]
            : newConfig.searcher.max_length;
          const lengthDifference = newLength - originalLength;
          if (
            originalLength &&
            lengthDifference &&
            Number.isInteger(originalLength) &&
            Number.isInteger(lengthDifference) &&
            lengthDifference > 0
          ) {
            additionalLength = lengthDifference;
          }
        } catch {
          additionalLength = undefined;
        }

        form.setFields([
          { name: EXPERIMENT_NAME, value: getExperimentName(newConfig) },
          {
            name: MAX_LENGTH,
            value: !isReactivate ? getMaxLengthValue(newConfig) : undefined,
          },
          { name: ADDITIONAL_LENGTH, value: additionalLength },
        ]);
        await form.validateFields();
      } catch (e) {
        handleError(e, { publicMessage: 'failed to load previous yaml config' });
      }
    } else {
      setDisabled(false);
    }
  }, [form, modalState, originalConfig.searcher.max_length]);

  const getConfigFromForm = useCallback(
    (config: RawJson) => {
      if (!form) return yaml.dump(config);

      const formValues = form.getFieldsValue();
      const newConfig = structuredClone(config);

      if (formValues[MAX_LENGTH]) {
        const maxLengthType = getMaxLengthType(newConfig);
        if (maxLengthType === undefined) {
          // Unitless searcher config.
          newConfig.searcher.max_length = parseInt(formValues[MAX_LENGTH]);
        } else {
          newConfig.searcher.max_length = { [maxLengthType]: parseInt(formValues[MAX_LENGTH]) };
        }
      }
      return yaml.dump(newConfig);
    },
    [form],
  );

  const submitExperiment = useCallback(
    async (newConfig: string) => {
      const isReactivate = modalState.type === ContinueExperimentType.Reactivate;
      if (!modalState.experiment || (!isReactivate && !modalState.trial)) return;
      try {
        if (!isReactivate) {
          const { experiment: newExperiment, warnings } = await createExperiment({
            activate: true,
            experimentConfig: newConfig,
            parentId: modalState.experiment.id,
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
        } else {
          await continueExperiment({
            id: modalState.experiment.id,
            overrideConfig: newConfig,
          });
          const newPath = paths.experimentDetails(experiment.id);
          routeToReactUrl(paths.reload(newPath));
        }
      } catch (e) {
        let errorMessage = `Unable to ${actionCopy.toLowerCase()} with the provided config.`;
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
    [actionCopy, experiment.id, modalState.experiment, modalState.trial, modalState.type],
  );

  const handleSubmit = async () => {
    const error = modalState.error || modalState.configError;
    if (error) throw new Error(error);

    const { isAdvancedMode } = modalState;
    let userConfig, fullConfig;
    if (isAdvancedMode) {
      userConfig = (yaml.load(modalState.configString) || {}) as RawJson;
    } else {
      await form.validateFields();
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

    if (isReactivate) {
      const { workspace, project, ...restConfig } = fullConfig;
      fullConfig = {
        project: undefined,
        workspace: undefined,
        ...restConfig,
      };
    }
    const configString = isAdvancedMode ? yaml.dump(fullConfig) : getConfigFromForm(fullConfig);
    await submitExperiment(configString);
  };

  useEffect(() => {
    let config = upgradeConfig(experiment.configRaw);

    if (!isReactivate && trial) {
      config = trialContinueConfig(
        config,
        trial.hyperparameters,
        trial.id,
        experiment.workspaceName,
        experiment.projectName,
      );
      config.description =
        `Continuation of ${entityCopyMap.trial} ${trial.id}, ${entityCopyMap.experiment} ${experiment.id}` +
        (config.description ? ` (${config.description})` : '');
    } else if (isReactivate) {
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
        ...prev,
        config: publicConfig,
        configString: prev.configString || yaml.dump(publicConfig),
        experiment,
        open: true,
        trial,
        type,
      };
      return _.isEqual(prev, newModalState) ? prev : newModalState;
    });
    form.validateFields(requiredFields); // initial disabled state set here, gets updated later in handleFieldsChange
  }, [entityCopyMap, experiment, trial, type, isReactivate, form, requiredFields]);

  if (!experiment || (!isReactivate && !trial)) return <></>;

  const hideSimpleConfig = isReactivate && experiment.state !== RunState.Completed;

  const maxLengthType = capitalizeWord(getMaxLengthType(modalState.config) || 'batches');
  const modalIsInAdvancedMode = modalState.isAdvancedMode || hideSimpleConfig;
  return (
    <Modal
      cancel
      size={
        !hideSimpleConfig
          ? modalState.isAdvancedMode
            ? isReactivate
              ? 'medium'
              : 'large'
            : 'small'
          : 'large'
      }
      submit={{
        disabled,
        form: idPrefix + FORM_ID,
        handleError,
        handler: handleSubmit,
        text: isReactivate
          ? `Reactivate ${capitalize(entityCopyMap.trial)}`
          : `Launch ${capitalize(entityCopyMap.experiment)}`,
      }}
      title={actionCopy}
      onClose={handleModalClose}>
      <>
        {modalState.error && <Alert message={modalState.error} showIcon type="error" />}
        {modalState.configError && modalIsInAdvancedMode && (
          <Alert message={modalState.configError} showIcon type="error" />
        )}
        {modalIsInAdvancedMode && (
          <React.Suspense fallback={<Spinner spinning tip="Loading text editor..." />}>
            <CodeEditor
              file={Loaded(modalState.configString)}
              files={[{ key: 'config.yaml' }]}
              height="40vh"
              onChange={handleEditorChange}
              onError={handleError}
            />
          </React.Suspense>
        )}
        {!modalIsInAdvancedMode && (
          <Body>
            {isReactivate
              ? `Reactivate and continue the current ${entityCopyMap.trial} from the latest checkpoint`
              : f_flat_runs
                ? "Start a new run from the current run's latest checkpoint"
                : "Start a new experiment from the current trial's latest checkpoint"}
          </Body>
        )}
        <Form
          form={form}
          hidden={modalState.isAdvancedMode}
          id={idPrefix + FORM_ID}
          labelCol={{ span: 8 }}
          name="basic"
          onFieldsChange={handleFieldsChange}>
          {!isReactivate && (
            <Form.Item
              initialValue={experiment.name}
              label={`${capitalize(entityCopyMap.experiment)} name`}
              name={EXPERIMENT_NAME}
              rules={[
                {
                  message: `Please provide a new ${entityCopyMap.experiment} name.`,
                  required: true,
                },
              ]}>
              <Input />
            </Form.Item>
          )}
          {!isReactivate && (
            <Form.Item
              label={`Max ${maxLengthType}`}
              name={MAX_LENGTH}
              rules={[
                {
                  required: true,
                  validator: (_rule, value) => {
                    let errorMessage = '';
                    if (!value) errorMessage = `Please provide a max ${maxLengthType}.`;
                    if (value < 1) errorMessage = `Max ${maxLengthType} must be at least 1.`;
                    return errorMessage ? Promise.reject(errorMessage) : Promise.resolve();
                  },
                },
              ]}>
              <Input type="number" />
            </Form.Item>
          )}
          {isReactivate && !hideSimpleConfig && (
            <Form.Item
              label={
                <Row>
                  {`Additional ${maxLengthType}`}
                  <Icon
                    name="info"
                    showTooltip
                    title={`Add additional training to the current ${entityCopyMap.trial}.`}
                  />
                </Row>
              }
              name={ADDITIONAL_LENGTH}
              rules={[
                {
                  required: false,
                  validator: (_rule, value) => {
                    let errorMessage = '';
                    if (value < 0) errorMessage = `Additional ${maxLengthType} must be at least 0.`;
                    if (value && !Number.isInteger(value))
                      errorMessage = `Additional ${maxLengthType} must be an integer.`;
                    return errorMessage ? Promise.reject(errorMessage) : Promise.resolve();
                  },
                },
              ]}>
              <InputNumber className={css.fullWidth} />
            </Form.Item>
          )}
        </Form>
        <div>
          {!hideSimpleConfig && (
            <Button onClick={toggleMode}>
              {modalState.isAdvancedMode ? SIMPLE_CONFIG_BUTTON_TEXT : FULL_CONFIG_BUTTON_TEXT}
            </Button>
          )}
        </div>
      </>
    </Modal>
  );
};

export default ExperimentContinueModalComponent;
