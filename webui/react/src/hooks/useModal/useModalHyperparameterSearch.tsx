import { Alert, Button, Checkbox, Form, Input, InputNumber,
  ModalFuncProps, Select, Space, Typography } from 'antd';
import { FormInstance } from 'antd/es/form/Form';
import { CheckboxChangeEvent } from 'antd/lib/checkbox';
import { SelectValue } from 'antd/lib/select';
import yaml from 'js-yaml';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Link from 'components/Link';
import SelectFilter from 'components/SelectFilter';
import { useStore } from 'contexts/Store';
import { maxPoolSlotCapacity } from 'pages/Cluster/ClusterOverview';
import { paths } from 'routes/utils';
import { createExperiment } from 'services/api';
import { Primitive } from 'shared/types';
import { clone } from 'shared/utils/data';
import { DetError, isDetError } from 'shared/utils/error';
import { roundToPrecision } from 'shared/utils/number';
import { routeToReactUrl } from 'shared/utils/routes';
import { ExperimentBase, ExperimentSearcherName, Hyperparameter,
  HyperparameterType, ResourcePool } from 'types';

import useModal, { ModalHooks as Hooks, ModalCloseReason } from './useModal';
import css from './useModalHyperparameterSearch.module.scss';

interface Props {
  experiment: ExperimentBase;
}

export interface ShowModalProps {
  initialModalProps?: ModalFuncProps;
}

interface ModalHooks extends Omit<Hooks, 'modalOpen'> {
  modalOpen: (props: ShowModalProps) => void;
}

interface SearchMethod {
  description: string;
  displayName: string;
  name: `${ExperimentSearcherName}`;
}

const SearchMethods: Record<string, SearchMethod> = {
  ASHA: {
    description: `Automated HP search multi-trial experiment that will stop poor 
  performing trials early as it searches the HP space.`,
    displayName: 'Adaptive ASHA',
    name: 'adaptive_asha',
  },
  AsyncHalving: {
    description: `Automated HP search multi-trial experiment that will stop poor
  performing trials early as it searches the HP space.`,
    displayName: 'Async Halving',
    name: 'async_halving',
  },
  Grid: {
    description: `Brute force evaluates all possible hyperparameter configurations 
  and returns the best.`,
    displayName: 'Grid',
    name: 'grid',
  },
  Random: {
    description: `Evaluates a set of hyperparameter configurations chosen at 
  random and returns the best.`,
    displayName: 'Random',
    name: 'random',
  },
} as const;

interface HyperparameterRowValues {
  active: boolean,
  max_val: number,
  min_val:number
  type: HyperparameterType,
  value: number | string,
}

const useModalHyperparameterSearch = ({ experiment }: Props): ModalHooks => {
  const { modalClose, modalOpen: openOrUpdate, modalRef } = useModal();
  const [ modalError, setModalError ] = useState<string>();
  const [ searchMethod, setSearchMethod ] = useState(SearchMethods.ASHA);
  const { resourcePools } = useStore();
  const [ resourcePool, setResourcePool ] = useState<ResourcePool>();
  const [ form ] = Form.useForm();
  const [ currentPage, setCurrentPage ] = useState(0);
  const [ canceler ] = useState(new AbortController());
  const [ slotsError, setSlotsError ] = useState(false);
  const [ validationError, setValidationError ] = useState(false);

  const handleSelectSearchMethod = useCallback((value: SelectValue) => {
    setSearchMethod(
      Object.values(SearchMethods).find(searcher => searcher.name === value) ?? SearchMethods.ASHA,
    );
  }, []);

  const hyperparameters = useMemo(() => {
    return Object.entries(experiment.hyperparameters)
      .map(hp => ({ hyperparameter: hp[1], name: hp[0] }));
  }, [ experiment.hyperparameters ]);

  const submitExperiment = useCallback(async () => {
    const fields: Record<string, Primitive | HyperparameterRowValues> = form.getFieldsValue(true);

    const baseConfig = clone(experiment.configRaw, true);
    baseConfig.searcher.name = fields.searcher;
    baseConfig.searcher.max_trials = fields.max_trials;
    baseConfig.resources.resource_pool = fields.pool;
    baseConfig.resources.max_slots = fields.slots;

    Object.entries(fields)
      .filter(field => typeof field[1] === 'object')
      .forEach(hp => {
        const hpName = hp[0];
        const hpInfo = hp[1] as HyperparameterRowValues;
        if (hpInfo.type === HyperparameterType.Categorical) return;
        if (hpInfo.active) {
          baseConfig.hyperparameters[hpName] = {
            maxval: hpInfo.type === HyperparameterType.Int ?
              roundToPrecision(hpInfo.max_val, 0) :
              hpInfo.max_val,
            minval: hpInfo.type === HyperparameterType.Int ?
              roundToPrecision(hpInfo.min_val, 0) :
              hpInfo.min_val,
            type: hpInfo.type,
          };
        } else {
          baseConfig.hyperparameters[hpName] = {
            type: hpInfo.type,
            val: hpInfo.value,
          };
        }
        if (hpInfo.type === HyperparameterType.Log) baseConfig.hyperparameters[hpName].base = 10.0;
      });

    console.log(baseConfig);

    const newConfig = yaml.dump(baseConfig);

    try {
      // const { id: newExperimentId } = await createExperiment({
      //   activate: true,
      //   experimentConfig: newConfig,
      //   parentId: experiment.id,
      //   projectId: experiment.projectId,
      // }, { signal: canceler.signal });

      // // Route to reload path to forcibly remount experiment page.
      // const newPath = paths.experimentDetails(newExperimentId);
      // routeToReactUrl(paths.reload(newPath));
    } catch (e) {
      let errorMessage = 'Unable to create experiment.';
      if (isDetError(e)) {
        errorMessage = e.publicMessage || e.message;
      }

      setModalError(errorMessage);

      // We throw an error to prevent the modal from closing.
      throw new DetError(errorMessage, { publicMessage: errorMessage, silent: true });
    }
  }, [ canceler.signal, experiment.configRaw, experiment.id, experiment.projectId, form ]);

  const handleOk = useCallback(() => {
    if (currentPage === 0) {
      setCurrentPage(1);
    } else {
      submitExperiment();
    }
  }, [ currentPage, submitExperiment ]);

  const handleBack = useCallback(() => {
    setCurrentPage(0);
  }, []);

  const handleCancel = useCallback(() => {
    modalClose(ModalCloseReason.Cancel);
  }, [ modalClose ]);

  const handleSelectPool = useCallback((value: SelectValue) => {
    setResourcePool(resourcePools.find(pool => pool.name === value));
  }, [ resourcePools ]);

  const maxSlots = useMemo(
    () => resourcePool ? maxPoolSlotCapacity(resourcePool) : 0,
    [ resourcePool ],
  );

  useEffect(() => {
    if (resourcePool || resourcePools.length === 0) return;
    setResourcePool(resourcePools[0]);
  }, [ resourcePool, resourcePools ]);

  const validateSlots = useCallback((slots: number) => {
    setSlotsError(!(Number.isInteger(slots) && slots >= 1 && slots <= maxSlots));
  }, [ maxSlots ]);

  const validateForm = useCallback(() => {
    form.validateFields()
      .catch((errorInfo) => setValidationError(errorInfo.errorFields.length !== 0));
  }, [ form ]);

  const page1 = useMemo((): React.ReactNode => {
    // We always render the form regardless of mode to provide a reference to it.
    return (
      <div className={css.base}>
        {modalError && <Alert className={css.error} message={modalError} type="error" />}
        <p>
          Select the hyperparameter space and search method to
          optimize your model hyperparameters.
        </p>
        <Form.Item
          initialValue={SearchMethods.ASHA.name}
          label={(
            <div className={css.searchTitle}>
              <p>Search method</p>
              <Link
                external
                path={paths.
                  docs('/training-hyperparameter/index.html#specifying-the-search-algorithm')}>
                Learn more
              </Link>
            </div>
          )}
          name="searcher">
          <SelectFilter
            onChange={handleSelectSearchMethod}>
            {Object.values(SearchMethods).map(searcher => (
              <Select.Option key={searcher.name} value={searcher.name}>
                {searcher.displayName}
              </Select.Option>
            ))}
          </SelectFilter>
        </Form.Item>
        <p>{searchMethod.description}</p>
        <div className={css.hyperparameterContainer}>
          <h2 className={css.hyperparameterName}>Hyperparameter</h2>
          <h2>Type</h2>
          <h2>Current</h2>
          <h2>Min value</h2>
          <h2>Max value</h2>
          {hyperparameters.map(hp => (
            <HyperparameterRow
              form={form}
              key={hp.name}
              {...hp}
            />
          ))}
        </div>
      </div>
    );
  }, [ form,
    handleSelectSearchMethod,
    hyperparameters,
    modalError,
    searchMethod.description ]);

  const page2 = useMemo((): React.ReactNode => {
    // We always render the form regardless of mode to provide a reference to it.
    return (
      <div className={css.base}>
        {modalError && <Alert className={css.error} message={modalError} type="error" />}
        <p>Select the resources to allocate to this search and the trial iteration limit.</p>
        <div className={css.poolRow}>
          <Form.Item
            initialValue={resourcePools?.[0]?.name}
            label="Resource Pool"
            name="pool"
            rules={[ { required: true } ]}>
            <SelectFilter
              onChange={handleSelectPool}>
              {resourcePools.map(pool => (
                <Select.Option key={pool.name} value={pool.name}>
                  {pool.name}
                </Select.Option>
              ))}
            </SelectFilter>
          </Form.Item>
          <Form.Item
            initialValue={1}
            label="Slots"
            name="slots"
            rules={[ {
              max: maxSlots,
              min: 1,
              required: true,
              type: 'number',
            } ]}
            validateStatus={slotsError ? 'error' : 'success'}>
            <InputNumber precision={0} onChange={validateSlots} />
          </Form.Item>
        </div>
        {slotsError && (
          <p className={css.error}>
            Slots must be an integer between 1 and {maxSlots} (max slots).
          </p>
        )}
        <p>{maxSlots} max slots</p>
        <Form.Item
          initialValue={1}
          label="Max Trials"
          name="max_trials"
          rules={[ { min: 1, required: true, type: 'number' } ]}>
          <InputNumber className={css.fullWidth} precision={0} />
        </Form.Item>
        <p>
          Note: HP search jobs (while more efficient than manual searching) can take longer
          depending on the size of the HP search space and the resources you allocate to it.
        </p>
      </div>
    );
  }, [ handleSelectPool, maxSlots, modalError, resourcePools, slotsError, validateSlots ]);

  const pages = useMemo(() => [ page1, page2 ], [ page1, page2 ]);

  const footer = useMemo(() => {
    return (
      <div className={css.footer}>
        {currentPage === 1 && <Button onClick={handleBack}>Back</Button>}
        <div className={css.spacer} />
        <Space>
          <Button onClick={handleCancel}>Cancel</Button>
          <Button disabled={validationError} type="primary" onClick={handleOk}>
            {currentPage === 0 ? 'Select Resources' : 'Run Experiment'}
          </Button>
        </Space>
      </div>
    );
  }, [ currentPage, handleBack, handleCancel, handleOk, validationError ]);

  const modalProps: Partial<ModalFuncProps> = useMemo(() => {
    return {
      className: css.modal,
      closable: true,
      content: (
        <Form form={form} layout="vertical" requiredMark={false} onValuesChange={validateForm}>
          {pages[currentPage]}
          {footer}
        </Form>),
      icon: null,
      title: 'Hyperparameter Search',
      width: 600,
    };
  }, [ form, validateForm, pages, currentPage, footer ]);

  const modalOpen = useCallback(({ initialModalProps }: ShowModalProps) => {
    setCurrentPage(0);
    form.resetFields();
    openOrUpdate({ ...modalProps, ...initialModalProps });
  }, [ form, modalProps, openOrUpdate ]);

  /*
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal
   */
  useEffect(() => {
    if (modalRef.current) openOrUpdate(modalProps);
  }, [ modalProps, modalRef, openOrUpdate ]);

  return { modalClose, modalOpen, modalRef };
};

interface RowProps {
  form: FormInstance;
  hyperparameter: Hyperparameter;
  name: string;
}

const HyperparameterRow: React.FC<RowProps> = (
  { form, hyperparameter, name }: RowProps,
) => {
  const [ checked, setChecked ] = useState(form.getFieldValue([ name, 'active' ]) ?? false);
  const [ type, setType ] = useState<HyperparameterType>(hyperparameter.type);
  const [ valError, setValError ] = useState<string>();
  const [ minError, setMinError ] = useState<string>();
  const [ maxError, setMaxError ] = useState<string>();
  const [ rangeError, setRangeError ] = useState<string>();

  const handleCheck = useCallback((e: CheckboxChangeEvent) => {
    const { checked } = e.target;
    setChecked(e.target.checked);
    setType(checked ? HyperparameterType.Double : HyperparameterType.Constant);
    form.setFields([ {
      name: [ name, 'type' ],
      value: checked ? HyperparameterType.Double : HyperparameterType.Constant,
    } ]);
    form.validateFields();
  }, [ form, name ]);

  const handleTypeChange = useCallback((value: HyperparameterType) => {
    setType(value);
    setChecked(value !== HyperparameterType.Constant);
    form.setFields([ { name: [ name, 'active' ], value: value !== HyperparameterType.Constant } ]);
    form.validateFields();
  }, [ form, name ]);

  const validateValue = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const { value } = e.target;
    setValError(value === '' ? 'Current value is required.' : undefined);
  }, []);

  const validateMin = useCallback((value: number | string | null) => {
    if (value == null) {
      setMinError('Minimum value is required.');
    } else if (typeof value === 'string') {
      setMinError('Minimum value must be a number.');
    } else if (value > (form.getFieldValue([ name, 'max' ]) as number)) {
      setRangeError('Maximum value must be greater or equal to than minimum value.');
      setMinError(undefined);
    } else {
      setMinError(undefined);
      setRangeError(undefined);
    }
  }, [ form, name ]);

  const validateMax = useCallback((value: number | string | null) => {
    if (value == null) {
      setMaxError('Maximum value is required.');
    } else if (typeof value === 'string') {
      setMaxError('Maximum value must be a number.');
    } else if (value < (form.getFieldValue([ name, 'min' ]) as number)) {
      setRangeError('Maximum value must be greater or equal to than minimum value.');
    } else {
      setMaxError(undefined);
      setRangeError(undefined);
    }
  }, [ form, name ]);

  return (
    <>
      <Space className={css.hyperparameterName}>
        <Form.Item
          initialValue={hyperparameter.type !== HyperparameterType.Constant}
          name={[ name, 'active' ]}
          noStyle
          valuePropName="checked">
          <Checkbox onChange={handleCheck} />
        </Form.Item>
        <Typography.Title ellipsis={{ rows: 1, tooltip: true }} level={3}>{name}</Typography.Title>
      </Space>
      <Form.Item
        dependencies={[ [ name, 'max' ], [ name, 'min' ] ]}
        initialValue={hyperparameter.type}
        name={[ name, 'type' ]}
        noStyle>
        <Select onChange={handleTypeChange}>
          {(Object.keys(HyperparameterType) as Array<keyof typeof HyperparameterType>)
            .map((type) => (
              <Select.Option
                disabled={HyperparameterType[type] === HyperparameterType.Categorical}
                key={HyperparameterType[type]}
                value={HyperparameterType[type]}>
                {type}
              </Select.Option>
            ))}
        </Select>
      </Form.Item>
      <Form.Item
        initialValue={hyperparameter.val}
        name={[ name, 'value' ]}
        rules={[ { required: !form.getFieldValue([ name, 'active' ]) } ]}
        validateStatus={valError ? 'error' : 'success'}>
        <Input disabled={form.getFieldValue([ name, 'active' ])} onChange={validateValue} />
      </Form.Item>
      {type === HyperparameterType.Categorical ? (
        <>
          <Form.Item
            initialValue={hyperparameter.minval}
            name={[ name, 'min' ]}
            noStyle>
            <Input disabled />
          </Form.Item>
          <Form.Item
            initialValue={hyperparameter.maxval}
            name={[ name, 'max' ]}
            noStyle>
            <Input disabled />
          </Form.Item>
        </>
      ) : (
        <>
          <Form.Item
            dependencies={[ [ name, 'active' ], [ name, 'type' ] ]}
            initialValue={hyperparameter.minval}
            name={[ name, 'min' ]}
            rules={[ {
              max: form.getFieldValue([ name, 'max' ]),
              required: form.getFieldValue([ name, 'active' ]),
              type: 'number',
            } ]}
            validateStatus={((typeof minError === 'string' || typeof rangeError === 'string')
        && form.getFieldValue([ name, 'active' ])) ? 'error' : undefined}>
            <InputNumber
              disabled={!form.getFieldValue([ name, 'active' ])}
              precision={form.getFieldValue([ name, 'type' ]) === HyperparameterType.Int ?
                0 : undefined}
              onChange={validateMin}
            />
          </Form.Item>
          <Form.Item
            initialValue={hyperparameter.maxval}
            name={[ name, 'max' ]}
            rules={[ {
              min: form.getFieldValue([ name, 'min' ]),
              required: form.getFieldValue([ name, 'active' ]),
              type: 'number',
            } ]}
            validateStatus={((typeof maxError === 'string' || typeof rangeError === 'string')
        && form.getFieldValue([ name, 'active' ])) ? 'error' : undefined}>
            <InputNumber
              disabled={!form.getFieldValue([ name, 'active' ])}
              precision={form.getFieldValue([ name, 'type' ]) === HyperparameterType.Int ?
                0 : undefined}
              onChange={validateMax}
            />
          </Form.Item>
        </>
      )}
      {form.getFieldValue([ name, 'type' ]) === HyperparameterType.Categorical &&
        <p className={css.warning}>Categorical hyperparameters are not currently supported.</p>}
      {form.getFieldValue([ name, 'type' ]) === HyperparameterType.Log &&
        <p className={css.warning}>Logs are base 10.</p>}
      {(!form.getFieldValue([ name, 'active' ]) && valError) &&
        <p className={css.error}>{valError}</p>}
      {(form.getFieldValue([ name, 'active' ]) && minError) &&
        <p className={css.error}>{minError}</p>}
      {(form.getFieldValue([ name, 'active' ]) && maxError) &&
        <p className={css.error}>{maxError}</p>}
      {(form.getFieldValue([ name, 'active' ]) && rangeError) &&
        <p className={css.error}>{rangeError}</p>}
    </>
  );
};

export default useModalHyperparameterSearch;
