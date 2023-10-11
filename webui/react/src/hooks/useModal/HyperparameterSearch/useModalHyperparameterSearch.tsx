import { InfoCircleOutlined } from '@ant-design/icons';
import { Select as AntdSelect, ModalFuncProps, Radio, Space, Typography } from 'antd';
import { RefSelectProps } from 'antd/lib/select';
import yaml from 'js-yaml';
import React, { useCallback, useEffect, useId, useMemo, useRef, useState } from 'react';

import Button from 'components/kit/Button';
import Checkbox from 'components/kit/Checkbox';
import Form from 'components/kit/Form';
import Icon from 'components/kit/Icon';
import Input from 'components/kit/Input';
import InputNumber from 'components/kit/InputNumber';
import Message from 'components/kit/Message';
import Select, { Option, SelectValue } from 'components/kit/Select';
import Tooltip from 'components/kit/Tooltip';
import { Loadable } from 'components/kit/utils/loadable';
import Link from 'components/Link';
import useModal, { ModalHooks as Hooks, ModalCloseReason } from 'hooks/useModal/useModal';
import { paths } from 'routes/utils';
import { createExperiment } from 'services/api';
import { V1LaunchWarning } from 'services/api-ts-sdk';
import clusterStore, { maxPoolSlotCapacity } from 'stores/cluster';
import {
  ExperimentItem,
  ExperimentSearcherName,
  Hyperparameter,
  HyperparameterType,
  Primitive,
  ResourcePool,
  TrialDetails,
  TrialHyperparameters,
  TrialItem,
} from 'types';
import { flattenObject, isBoolean, unflattenObject } from 'utils/data';
import { DetError, ErrorLevel, ErrorType, handleWarning, isDetError } from 'utils/error';
import { roundToPrecision } from 'utils/number';
import { useObservable } from 'utils/observable';
import { routeToReactUrl } from 'utils/routes';
import { validateLength } from 'utils/string';

import css from './useModalHyperparameterSearch.module.scss';

const FORM_ID = 'create-hp-search-form';

interface Props {
  experiment: ExperimentItem;
  onClose?: () => void;
  trial?: TrialDetails | TrialItem;
}

export interface ShowModalProps {
  initialModalProps?: ModalFuncProps;
  trial?: TrialDetails | TrialItem;
}

interface ModalHooks extends Omit<Hooks, 'modalOpen'> {
  modalOpen: (props?: ShowModalProps) => void;
}

interface SearchMethod {
  displayName: string;
  icon: React.ReactNode;
  name: `${ExperimentSearcherName}`;
}

const SEARCH_METHODS: Record<string, SearchMethod> = {
  ASHA: {
    displayName: 'Adaptive',
    icon: <Icon name="searcher-adaptive" title="Adaptive" />,
    name: 'adaptive_asha',
  },
  Grid: {
    displayName: 'Grid',
    icon: <Icon name="searcher-grid" title="Grid" />,
    name: 'grid',
  },
  Random: {
    displayName: 'Random',
    icon: <Icon name="searcher-random" title="Random" />,
    name: 'random',
  },
} as const;

const DEFAULT_LOG_BASE = 10;

interface HyperparameterRowValues {
  count?: number;
  max?: number;
  min?: number;
  type: HyperparameterType;
  value?: number | string;
}

const useModalHyperparameterSearch = ({
  experiment,
  onClose,
  trial: trialIn,
}: Props): ModalHooks => {
  const idPrefix = useId();
  const { modalClose, modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal({ onClose });
  const [trial, setTrial] = useState(trialIn);
  const [modalError, setModalError] = useState<string>();
  const [searcher, setSearcher] = useState(
    Object.values(SEARCH_METHODS).find((searcher) => searcher.name === experiment.searcherType) ??
      SEARCH_METHODS.ASHA,
  );
  const canceler = useRef<AbortController>(new AbortController());
  const resourcePools = Loadable.getOrElse([], useObservable(clusterStore.resourcePools));
  const [resourcePool, setResourcePool] = useState<ResourcePool>(
    resourcePools.find((pool) => pool.name === experiment.resourcePool) ?? resourcePools[0],
  );
  const [form] = Form.useForm();
  const [currentPage, setCurrentPage] = useState(0);
  const [validationError, setValidationError] = useState(false);
  const formValues = Form.useWatch([], form);

  const trialHyperparameters = useMemo(() => {
    if (!trial) return;
    const continueFn = (value: unknown) => value === 'object';
    return flattenObject<TrialHyperparameters>(trial.hyperparameters, {
      continueFn,
    }) as unknown as Record<string, Primitive>;
  }, [trial]);

  const hyperparameters = useMemo(() => {
    return Object.entries(experiment.hyperparameters).map((hp) => {
      const hpObject = { hyperparameter: hp[1], name: hp[0] };
      if (trialHyperparameters?.[hp[0]]) {
        hpObject.hyperparameter.val = trialHyperparameters[hp[0]];
      }
      return hpObject;
    });
  }, [experiment.hyperparameters, trialHyperparameters]);

  const submitExperiment = useCallback(async () => {
    const fields: Record<string, Primitive | HyperparameterRowValues> = form.getFieldsValue(true);

    // Deep cloning parent experiment's config
    const baseConfig = structuredClone(experiment.configRaw);

    // Replacing fields from orginial config with user-selected values
    baseConfig.name = (fields.name as string).trim();
    baseConfig.searcher.name = fields.searcher;
    baseConfig.searcher.max_trials =
      fields.searcher === SEARCH_METHODS.Grid.name ? undefined : fields.max_trials;
    baseConfig.searcher.max_length = {};
    baseConfig.searcher.max_length[fields.length_units as string] = fields.max_length;
    baseConfig.searcher.max_concurrent_trials = fields.max_concurrent_trials ?? 16;
    baseConfig.resources.resource_pool = fields.pool;
    baseConfig.resources.slots_per_trial = fields.slots_per_trial;

    // Dealing with ASHA-specific settings
    if (fields.searcher === SEARCH_METHODS.ASHA.name) {
      baseConfig.searcher.bracket_rungs = baseConfig.searcher.bracket_rungs ?? [];
      baseConfig.searcher.stop_once = fields.stop_once ?? baseConfig.searcher.stop_once ?? false;
      baseConfig.searcher.max_rungs = baseConfig.searcher.max_rungs ?? 5;
      baseConfig.searcher.divisor = baseConfig.searcher.divisor ?? 4;
      baseConfig.searcher.mode = fields.mode ?? baseConfig.searcher.mode ?? 'standard';
    } else {
      baseConfig.searcher.bracket_rungs = undefined;
      baseConfig.searcher.stop_once = undefined;
      baseConfig.searcher.max_rungs = undefined;
      baseConfig.searcher.divisor = undefined;
      baseConfig.searcher.mode = undefined;
    }

    // Parsing hyperparameters
    Object.entries(fields)
      .filter((field) => typeof field[1] === 'object')
      .forEach((hp) => {
        const hpName = hp[0];
        const hpInfo = hp[1] as HyperparameterRowValues;
        if (hpInfo.type === HyperparameterType.Categorical) return;
        else if (hpInfo.type === HyperparameterType.Constant) {
          if (!hpInfo.value) return;
          let parsedVal;
          try {
            if (typeof hpInfo.value === 'string') {
              // Parse hyperparameter value in case it's not a string or number
              parsedVal = JSON.parse(hpInfo.value);
            } else {
              parsedVal = hpInfo.value;
            }
          } catch (e) {
            parsedVal = hpInfo.value;
          }
          baseConfig.hyperparameters[hpName] = {
            type: hpInfo.type,
            val: parsedVal,
          };
        } else {
          const prevBase: number | undefined = baseConfig.hyperparameters[hpName]?.base;
          baseConfig.hyperparameters[hpName] = {
            base: hpInfo.type === HyperparameterType.Log ? prevBase ?? DEFAULT_LOG_BASE : undefined,
            count: fields.searcher === SEARCH_METHODS.Grid.name ? hpInfo.count : undefined,
            maxval:
              hpInfo.type === HyperparameterType.Int
                ? roundToPrecision(hpInfo.max ?? 0, 0)
                : hpInfo.max,
            minval:
              hpInfo.type === HyperparameterType.Int
                ? roundToPrecision(hpInfo.min ?? 0, 0)
                : hpInfo.min,
            type: hpInfo.type,
          };
        }
      });

    // Unflatten hyperparameters to deal with nesting
    baseConfig.hyperparameters = unflattenObject(baseConfig.hyperparameters);

    const newConfig = yaml.dump(baseConfig);

    try {
      const { experiment: newExperiment, warnings } = await createExperiment(
        {
          activate: true,
          experimentConfig: newConfig,
          parentId: experiment.id,
          projectId: experiment.projectId,
        },
        { signal: canceler.current?.signal },
      );
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
      let errorMessage = 'Unable to create experiment.';
      if (isDetError(e)) {
        errorMessage = e.publicMessage || e.message;
      }

      setModalError(errorMessage);

      // We throw an error to prevent the modal from closing.
      throw new DetError(errorMessage, { publicMessage: errorMessage, silent: true });
    }
  }, [experiment.configRaw, experiment.id, experiment.projectId, form]);

  const handleOk = useCallback(() => {
    if (currentPage === 0) {
      setCurrentPage(1);
    } else {
      submitExperiment();
    }
  }, [currentPage, submitExperiment]);

  const handleBack = useCallback(() => {
    setCurrentPage((prev) => prev - 1);
  }, []);

  const handleCancel = useCallback(() => {
    modalClose(ModalCloseReason.Cancel);
  }, [modalClose]);

  const handleSelectPool = useCallback(
    (value: SelectValue) => {
      setResourcePool(resourcePools.find((pool) => pool.name === value) ?? resourcePools[0]);
    },
    [resourcePools],
  );

  const maxSlots = useMemo(
    () => (resourcePool ? maxPoolSlotCapacity(resourcePool) : 0),
    [resourcePool],
  );

  const [maxLengthUnit, maxLength] = useMemo(() => {
    return (Object.entries(experiment.config.searcher.max_length ?? { batches: 1 })[0] ?? [
      'batches',
      1,
    ]) as ['batches' | 'records' | 'epochs', number];
  }, [experiment.config.searcher.max_length]);

  useEffect(() => {
    if (resourcePool || resourcePools.length === 0) return;
    setResourcePool(resourcePools[0]);
  }, [resourcePool, resourcePools]);

  const validateForm = useCallback(() => {
    if (!formValues) return;
    if (currentPage === 1) {
      // Validating hyperparameters page
      const hyperparameters = formValues as Record<string, HyperparameterRowValues>;
      setValidationError(
        !Object.values(hyperparameters).every((hp) => {
          switch (hp.type) {
            case HyperparameterType.Categorical:
              return true;
            case HyperparameterType.Constant:
              return hp.value != null;
            default:
              return (
                hp.min != null &&
                hp.max != null &&
                hp.max >= hp.min &&
                (searcher !== SEARCH_METHODS.Grid || (hp.count != null && hp.count > 0))
              );
          }
        }),
      );
    } else if (currentPage === 0) {
      // Validating searcher page
      const {
        searcher,
        name,
        pool,
        slots_per_trial,
        max_trials,
        max_length,
        length_units,
        mode,
        stop_once,
        max_concurrent_trials,
      } = formValues;

      const validName = validateLength(name ?? '');
      const validSlotsPerTrial =
        slots_per_trial != null && slots_per_trial >= 0 && slots_per_trial <= maxSlots;
      const validMaxLength = max_length != null && max_length > 0;
      const validMaxConcurrentTrials = max_concurrent_trials != null && max_concurrent_trials >= 0;
      const validMaxTrials =
        searcher === SEARCH_METHODS.Grid.name || (max_trials != null && max_trials > 0);

      setValidationError(
        !(
          validName &&
          validSlotsPerTrial &&
          validMaxLength &&
          validMaxConcurrentTrials &&
          validMaxTrials &&
          pool != null &&
          length_units != null &&
          (searcher !== SEARCH_METHODS.ASHA.name || (mode != null && isBoolean(stop_once)))
        ),
      );
    }
  }, [currentPage, formValues, maxSlots, searcher]);

  useEffect(() => {
    validateForm();
  }, [validateForm]);

  const handleSelectSearcher = useCallback(
    (searcherName: string) => {
      const searcher =
        Object.values(SEARCH_METHODS).find((searcher) => searcher.name === searcherName) ??
        SEARCH_METHODS.ASHA;
      setSearcher(searcher);
      form.setFieldValue('searcher', searcher.name);
    },
    [form],
  );

  const hyperparameterPage = useMemo((): React.ReactNode => {
    // We always render the form regardless of mode to provide a reference to it.
    return (
      <div className={css.base}>
        {modalError && <Message icon="error" title={modalError} />}
        <div className={css.labelWithLink}>
          <p>Select hyperparameters and define the search space.</p>
          <Link
            external
            path={paths.docs('/training/hyperparameter/configure-hp-ranges.html')}
            popout>
            Learn more
          </Link>
        </div>
        <div
          className={css.hyperparameterContainer}
          style={{
            gridTemplateColumns: `180px minmax(100px, 1.4fr)
              repeat(${searcher === SEARCH_METHODS.Grid ? 4 : 3}, minmax(60px, 1fr))`,
          }}>
          <label id="hyperparameter">
            <h2>Hyperparameter</h2>
          </label>
          <label id="type">
            <h2>Type</h2>
          </label>
          <label id="current-value">
            <h2>Current</h2>
          </label>
          <label id="min-value">
            <h2>Min value</h2>
          </label>
          <label id="max-value">
            <h2>Max value</h2>
          </label>
          {searcher === SEARCH_METHODS.Grid && (
            <label id="count">
              <h2>Grid Count</h2>
            </label>
          )}
          {hyperparameters.map((hp) => (
            <HyperparameterRow key={hp.name} searcher={searcher} {...hp} />
          ))}
        </div>
      </div>
    );
  }, [hyperparameters, modalError, searcher]);

  const searcherPage = useMemo((): React.ReactNode => {
    // We always render the form regardless of mode to provide a reference to it.
    return (
      <div className={css.base}>
        {modalError && <Message icon="error" title={modalError} />}
        <Form.Item
          initialValue={searcher.name}
          label={
            <div className={css.labelWithLink}>
              <p>Select search method</p>
              <Link
                external
                path={paths.docs(
                  '/training-hyperparameter/index.html#specifying-the-search-algorithm',
                )}
                popout>
                Learn more
              </Link>
            </div>
          }
          name="searcher">
          <Radio.Group className={css.searcherGroup} optionType="button">
            {Object.values(SEARCH_METHODS).map((searcherOption) => (
              <Button
                column
                icon={searcherOption.icon}
                key={searcherOption.name}
                selected={searcher.name === searcherOption.name}
                onClick={() => handleSelectSearcher(searcherOption.name)}>
                {searcherOption.displayName}
              </Button>
            ))}
          </Radio.Group>
        </Form.Item>
        <Form.Item
          initialValue={experiment.name}
          label="New experiment name"
          name="name"
          rules={[{ required: true }]}>
          <Input maxLength={80} />
        </Form.Item>
        <div className={css.poolContainer}>
          <Form.Item
            initialValue={resourcePool?.name}
            label="Resource pool"
            name="pool"
            rules={[{ required: true }]}>
            <Select onChange={handleSelectPool}>
              {resourcePools.map((pool) => (
                <Option key={pool.name} value={pool.name}>
                  {pool.name}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <p>{maxSlots} max slots</p>
        </div>
        <h2 className={css.sectionTitle}>Configure Trials</h2>
        <div className={css.inputRow}>
          <Form.Item
            initialValue={maxLength}
            label="Max length"
            name="max_length"
            rules={[{ min: 1, required: true, type: 'number' }]}>
            <InputNumber min={1} precision={0} />
          </Form.Item>
          <Form.Item
            initialValue={maxLengthUnit}
            label="Units"
            name="length_units"
            rules={[{ required: true }]}>
            <Select>
              <Option value="records">records</Option>
              <Option value="batches">batches</Option>
              {(experiment.configRaw?.records_per_epoch ?? 0) > 0 && (
                <Option value="epochs">epochs</Option>
              )}
            </Select>
          </Form.Item>
          <Form.Item
            initialValue={experiment.configRaw?.resources?.slots_per_trial || 1}
            label="Slots per trial"
            name="slots_per_trial"
            rules={[{ max: maxSlots, min: 1, required: true, type: 'number' }]}
            validateStatus={
              formValues?.slots_per_trial > maxSlots || formValues?.slots_per_trial < 1
                ? 'error'
                : 'success'
            }>
            <InputNumber max={maxSlots} min={0} precision={0} />
          </Form.Item>
        </div>
        {searcher.name === 'adaptive_asha' && (
          <Form.Item
            initialValue={experiment.configRaw.searcher?.mode ?? 'standard'}
            label={
              <div className={css.labelWithTooltip}>
                Early stopping mode
                <Tooltip content="How aggressively to perform early stopping of underperforming trials">
                  <InfoCircleOutlined />
                </Tooltip>
              </div>
            }
            name="mode"
            rules={[{ required: true }]}>
            <Select>
              <Option value="aggressive">Aggressive</Option>
              <Option value="standard">Standard</Option>
              <Option value="conservative">Conservative</Option>
            </Select>
          </Form.Item>
        )}
        {searcher.name === 'adaptive_asha' && (
          <Form.Item
            initialValue={experiment.configRaw.searcher?.stop_once ?? true}
            name="stop_once"
            rules={[{ required: true }]}
            valuePropName="checked">
            <Checkbox>
              Stop once - Only stop trials one time when there is enough evidence to terminate
              training (recommended for faster search)
            </Checkbox>
          </Form.Item>
        )}
        <h2 className={css.sectionTitle}>Set Training Limits</h2>
        <div className={css.inputRow}>
          <Form.Item
            hidden={searcher === SEARCH_METHODS.Grid}
            initialValue={experiment.config.searcher.max_trials ?? 1}
            label="Max trials"
            name="max_trials"
            rules={[{ min: 1, required: true, type: 'number' }]}>
            <InputNumber min={1} precision={0} />
          </Form.Item>
          <Form.Item
            initialValue={experiment.configRaw.searcher.max_concurrent_trials ?? 16}
            label={
              <div className={css.labelWithTooltip}>
                Max concurrent trials
                <Tooltip content="Use 0 for max possible parallelism">
                  <InfoCircleOutlined style={{ color: 'var(--theme-colors-monochrome-8)' }} />
                </Tooltip>
              </div>
            }
            name="max_concurrent_trials"
            rules={[{ min: 0, required: true, type: 'number' }]}>
            <InputNumber min={0} precision={0} />
          </Form.Item>
        </div>
      </div>
    );
  }, [
    experiment.config.searcher.max_trials,
    experiment.configRaw?.records_per_epoch,
    experiment.configRaw?.resources?.slots_per_trial,
    experiment.configRaw.searcher.max_concurrent_trials,
    experiment.configRaw.searcher?.mode,
    experiment.configRaw.searcher?.stop_once,
    experiment.name,
    formValues?.slots_per_trial,
    handleSelectPool,
    handleSelectSearcher,
    maxLength,
    maxLengthUnit,
    maxSlots,
    modalError,
    resourcePool?.name,
    resourcePools,
    searcher,
  ]);

  const pages = useMemo(
    () => [searcherPage, hyperparameterPage],
    [hyperparameterPage, searcherPage],
  );

  const footer = useMemo(() => {
    return (
      <div className={css.footer}>
        {currentPage > 0 && <Button onClick={handleBack}>Back</Button>}
        <div className={css.spacer} />
        <Space>
          <Button onClick={handleCancel}>Cancel</Button>
          <Button disabled={validationError} type="primary" onClick={handleOk}>
            {currentPage === 0 ? 'Select Hyperparameters' : 'Run Experiment'}
          </Button>
        </Space>
      </div>
    );
  }, [currentPage, handleBack, handleCancel, handleOk, validationError]);

  const modalProps: Partial<ModalFuncProps> = useMemo(() => {
    return {
      className: css.modal,
      closable: true,
      content: (
        <Form form={form} id={idPrefix + FORM_ID} layout="vertical">
          {pages[currentPage]}
          {footer}
        </Form>
      ),
      icon: null,
      maskClosable: true,
      title: 'Hyperparameter Search',
      width: 700,
    };
  }, [form, idPrefix, pages, currentPage, footer]);

  const modalOpen = useCallback(
    (props?: ShowModalProps) => {
      setCurrentPage(0);
      form.resetFields();
      if (props?.trial) setTrial(props?.trial);
      openOrUpdate({ ...modalProps, ...props?.initialModalProps });
    },
    [form, modalProps, openOrUpdate],
  );

  /*
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal
   */
  useEffect(() => {
    if (modalRef.current) openOrUpdate(modalProps);
  }, [modalProps, modalRef, openOrUpdate]);

  return { modalClose, modalOpen, modalRef, ...modalHook };
};

interface RowProps {
  hyperparameter: Hyperparameter;
  name: string;
  searcher: SearchMethod;
}

const HyperparameterRow: React.FC<RowProps> = ({ hyperparameter, name, searcher }: RowProps) => {
  const type: HyperparameterType | undefined = Form.useWatch([name, 'type']);
  const typeRef = useRef<RefSelectProps>(null);
  const [active, setActive] = useState(hyperparameter.type !== HyperparameterType.Constant);
  const min: number | undefined = Form.useWatch([name, 'min']);
  const max: number | undefined = Form.useWatch([name, 'max']);
  const [valError, setValError] = useState<string>();
  const [minError, setMinError] = useState<string>();
  const [maxError, setMaxError] = useState<string>();
  const [rangeError, setRangeError] = useState<string>();
  const [countError, setCountError] = useState<string>();

  const handleTypeChange = useCallback((value: HyperparameterType) => {
    setActive(value !== HyperparameterType.Constant);
  }, []);

  const validateValue = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const { value } = e.target;
    setValError(value === '' ? 'Current value is required.' : undefined);
  }, []);

  const validateMin = useCallback(
    (value: number | string | null) => {
      if (value == null) {
        setMinError('Minimum value is required.');
      } else if (typeof value === 'string') {
        setMinError('Minimum value must be a number.');
      } else if (max != null && value > max) {
        setRangeError('Maximum value must be greater or equal to than minimum value.');
        setMinError(undefined);
      } else {
        setMinError(undefined);
        setRangeError(undefined);
      }
    },
    [max],
  );

  const validateMax = useCallback(
    (value: number | string | null) => {
      if (value == null) {
        setMaxError('Maximum value is required.');
      } else if (typeof value === 'string') {
        setMaxError('Maximum value must be a number.');
      } else if (min != null && value < min) {
        setRangeError('Maximum value must be greater or equal to than minimum value.');
      } else {
        setMaxError(undefined);
        setRangeError(undefined);
      }
    },
    [min],
  );

  const validateCount = useCallback((value: number | string | null) => {
    if (value == null) {
      setCountError('Grid count is required.');
    } else if (typeof value === 'string') {
      setCountError('Grid count must be a number.');
    } else if (value < 1) {
      setCountError('Grid count must be greater than or equal to 1.');
    } else {
      setCountError(undefined);
    }
  }, []);

  return (
    <>
      <div className={css.hyperparameterName}>
        <Typography.Title ellipsis={{ rows: 1, tooltip: true }} level={3}>
          {name}
        </Typography.Title>
      </div>
      <Form.Item initialValue={hyperparameter.type} name={[name, 'type']} noStyle>
        <AntdSelect
          aria-labelledby="type"
          getPopupContainer={(triggerNode) => triggerNode}
          ref={typeRef}
          onChange={handleTypeChange}>
          {(Object.keys(HyperparameterType) as Array<keyof typeof HyperparameterType>).map(
            (type) => (
              <Option
                disabled={HyperparameterType[type] === HyperparameterType.Categorical}
                key={HyperparameterType[type]}
                value={HyperparameterType[type]}>
                {type}
                {type === 'Log' ? ` (base ${hyperparameter.base ?? DEFAULT_LOG_BASE})` : ''}
              </Option>
            ),
          )}
        </AntdSelect>
      </Form.Item>
      <Form.Item
        initialValue={hyperparameter.val}
        name={[name, 'value']}
        rules={[{ required: !active }]}
        validateStatus={valError ? 'error' : 'success'}>
        <Input
          aria-labelledby="current-value"
          disabled={active}
          placeholder={active ? 'n/a' : ''}
          onChange={validateValue}
        />
      </Form.Item>
      {type === HyperparameterType.Categorical ? (
        <>
          <Form.Item initialValue={hyperparameter.minval} name={[name, 'min']} noStyle>
            <Input aria-labelledby="min-value" disabled />
          </Form.Item>
          <Form.Item initialValue={hyperparameter.maxval} name={[name, 'max']} noStyle>
            <Input aria-labelledby="max-value" disabled />
          </Form.Item>
          <Form.Item hidden={searcher !== SEARCH_METHODS.Grid} name={[name, 'count']}>
            <InputNumber aria-labelledby="count" disabled />
          </Form.Item>
        </>
      ) : (
        <>
          <Form.Item
            initialValue={hyperparameter.minval}
            name={[name, 'min']}
            rules={[
              {
                max: max,
                required: active,
                type: 'number',
              },
            ]}
            validateStatus={(minError || rangeError) && active ? 'error' : undefined}>
            <InputNumber
              aria-labelledby="min-value"
              className={type === HyperparameterType.Int ? undefined : css.disableArrows}
              disabled={!active}
              placeholder={!active ? 'n/a' : ''}
              precision={type === HyperparameterType.Int ? 0 : undefined}
              onChange={validateMin}
            />
          </Form.Item>
          <Form.Item
            initialValue={hyperparameter.maxval}
            name={[name, 'max']}
            rules={[
              {
                min: min,
                required: active,
                type: 'number',
              },
            ]}
            validateStatus={(maxError || rangeError) && active ? 'error' : undefined}>
            <InputNumber
              aria-labelledby="max-value"
              className={type === HyperparameterType.Int ? undefined : css.disableArrows}
              disabled={!active}
              placeholder={!active ? 'n/a' : ''}
              precision={type === HyperparameterType.Int ? 0 : undefined}
              onChange={validateMax}
            />
          </Form.Item>
          <Form.Item
            hidden={searcher !== SEARCH_METHODS.Grid}
            initialValue={hyperparameter.count}
            name={[name, 'count']}
            rules={[
              {
                min: 0,
                required: active && searcher === SEARCH_METHODS.Grid,
                type: 'number',
              },
            ]}
            validateStatus={
              countError && searcher === SEARCH_METHODS.Grid && active ? 'error' : undefined
            }>
            <InputNumber
              aria-labelledby="count"
              disabled={!active}
              placeholder={!active ? 'n/a' : ''}
              precision={0}
              onChange={validateCount}
            />
          </Form.Item>
        </>
      )}
      {type === HyperparameterType.Categorical && (
        <p className={css.warning}>Categorical hyperparameters are not currently supported.</p>
      )}
      {!active && valError && <p className={css.error}>{valError}</p>}
      {active && minError && <p className={css.error}>{minError}</p>}
      {active && maxError && <p className={css.error}>{maxError}</p>}
      {active && rangeError && <p className={css.error}>{rangeError}</p>}
      {active && countError && searcher === SEARCH_METHODS.Grid && (
        <p className={css.error}>{countError}</p>
      )}
    </>
  );
};

export default useModalHyperparameterSearch;
