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
import { routeToReactUrl } from 'shared/utils/routes';
import { ExperimentBase, ExperimentSearcherName, Hyperparameter, HyperparameterType, ResourcePool } from 'types';

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

  const handleSelectSearchMethod = useCallback((value: SelectValue) => {
    setSearchMethod(
      Object.values(SearchMethods).find(searcher => searcher.name === value) ?? SearchMethods.ASHA,
    );
  }, []);

  const hyperparameters = useMemo(() => {
    return Object.entries(experiment.hyperparameters)
      .map(hp => ({ hyperparameter: hp[1], name: hp[0] }));
  }, [ experiment.hyperparameters ]);

  const page1 = useMemo((): React.ReactNode => {
    // We always render the form regardless of mode to provide a reference to it.
    return (
      <Form form={form} layout="vertical" name="hyperparameters">
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
            {hyperparameters.map(hp => <HyperparameterRow form={form} key={hp.name} {...hp} />)}
          </div>
        </div>
      </Form>
    );
  }, [ form,
    handleSelectSearchMethod,
    hyperparameters,
    modalError,
    searchMethod.description ]);

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

  const page2 = useMemo((): React.ReactNode => {
    // We always render the form regardless of mode to provide a reference to it.
    return (
      <Form form={form} layout="vertical" requiredMark={false}>
        <div className={css.base}>
          {modalError && <Alert className={css.error} message={modalError} type="error" />}
          <p>Select the resources to allocate to this search and the trial iteration limit.</p>
          <div className={css.poolRow}>
            <Form.Item
              initialValue={resourcePools?.[0].name}
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
              } ]}>
              <InputNumber precision={0} />
            </Form.Item>
          </div>
          <p>{maxSlots} max slots</p>
          <Form.Item
            initialValue={1}
            label="Max Trials"
            name="max_trials"
            rules={[ { min: 0, required: true, type: 'number' } ]}>
            <InputNumber className={css.fullWidth} precision={0} />
          </Form.Item>
          <p>
            Note: HP search jobs (while more efficient than manual searching) can take longer
            depending on the size of the HP search space and the resources you allocate to it.
          </p>
        </div>
      </Form>
    );
  }, [ form, handleSelectPool, maxSlots, modalError, resourcePools ]);

  const pages = useMemo(() => [ page1, page2 ], [ page1, page2 ]);

  const submitExperiment = useCallback(async () => {
    const fields: Record<string, Primitive | HyperparameterRowValues> = form.getFieldsValue(true);
    console.log(fields);
    const baseConfig = clone(experiment.configRaw, true);
    baseConfig.searcher.name = fields.searcher;
    baseConfig.searcher.max_trials = fields.max_trials;
    baseConfig.resources.resource_pool = fields.pool;

    Object.entries(fields)
      .filter(field => typeof field[1] === 'object' && field[1].active)
      .forEach(hp => {
        const hpName = hp[0];
        const hpInfo = hp[1] as HyperparameterRowValues;
        baseConfig.hyperparameters[hpName] = {
          maxval: hpInfo.max_val,
          minval: hpInfo.min_val,
          type: hpInfo.type,
        };
        if (hpInfo.type === HyperparameterType.Log) baseConfig.hyperparameters[hpName].base = 10.0;
      });
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

  const footer = useMemo(() => {
    return (
      <div className={css.footer}>
        {currentPage === 1 && <Button onClick={handleBack}>Back</Button>}
        <div className={css.spacer} />
        <Space>
          <Button onClick={handleCancel}>Cancel</Button>
          <Button type="primary" onClick={handleOk}>
            {currentPage === 0 ? 'Select Resources' : 'Run Experiment'}
          </Button>
        </Space>
      </div>
    );
  }, [ currentPage, handleBack, handleCancel, handleOk ]);

  const modalProps: Partial<ModalFuncProps> = useMemo(() => {
    return {
      className: css.modal,
      closable: true,
      content: <>{pages[currentPage]}{footer}</>,
      icon: null,
      title: 'Hyperparameter Search',
      width: 600,
    };
  }, [ pages, currentPage, footer ]);

  const modalOpen = useCallback(({ initialModalProps }: ShowModalProps) => {
    setCurrentPage(0);
    openOrUpdate({ ...modalProps, ...initialModalProps });
  }, [ modalProps, openOrUpdate ]);

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

const HyperparameterRow: React.FC<RowProps> = ({ form, hyperparameter, name }: RowProps) => {
  const [ checked, setChecked ] = useState(false);
  const [ type, setType ] = useState<HyperparameterType>(hyperparameter.type);

  const handleCheck = useCallback((e: CheckboxChangeEvent) => {
    setChecked(e.target.checked);
    if (e.target.checked) {
      setType(HyperparameterType.Double);
    } else {
      setType(HyperparameterType.Constant);
    }
  }, []);

  const handleTypeChange = useCallback((value: HyperparameterType) => {
    setType(value);
    if (value === HyperparameterType.Constant) setChecked(false);
    else setChecked(true);
  }, []);

  useEffect(() => {
    form.setFields([ { name: [ name, 'type' ], value: type } ]);
  }, [ form, name, type ]);

  useEffect(() => {
    form.setFields([ { name: [ name, 'active' ], value: checked } ]);
  }, [ checked, form, name ]);

  const inputs = useMemo(() => {
    switch (type) {
      case HyperparameterType.Constant:
      case HyperparameterType.Double:
        return (
          <>
            <Form.Item
              initialValue={hyperparameter.val as number}
              name={[ name, 'value' ]}
              noStyle
              rules={[ {
                required: !checked,
                type: 'number',
              } ]}>
              <InputNumber disabled={checked} />
            </Form.Item>
            <Form.Item
              initialValue={hyperparameter.minval}
              name={[ name, 'min' ]}
              noStyle
              rules={[ {
                max: form.getFieldValue([ name, 'max' ]),
                required: checked,
                type: 'number',
              } ]}>
              <InputNumber disabled={!checked} />
            </Form.Item>
            <Form.Item
              initialValue={hyperparameter.maxval}
              name={[ name, 'max' ]}
              noStyle
              rules={[ {
                min: form.getFieldValue([ name, 'min' ]),
                required: checked,
                type: 'number',
              } ]}>
              <InputNumber disabled={!checked} />
            </Form.Item>
          </>
        );
      case HyperparameterType.Int:
        return (
          <>
            <Form.Item
              initialValue={hyperparameter.val as number}
              name={[ name, 'value' ]}
              noStyle
              rules={[ {
                required: !checked,
                type: 'number',
              } ]}>
              <InputNumber disabled={checked} precision={0} />
            </Form.Item>
            <Form.Item
              initialValue={hyperparameter.minval}
              name={[ name, 'min' ]}
              noStyle
              rules={[ {
                max: form.getFieldValue([ name, 'max' ]),
                required: checked,
                type: 'number',
              } ]}>
              <InputNumber
                disabled={!checked}
                precision={0}
              />
            </Form.Item>
            <Form.Item
              initialValue={hyperparameter.maxval}
              name={[ name, 'max' ]}
              noStyle
              rules={[ {
                min: form.getFieldValue([ name, 'min' ]),
                required: checked,
                type: 'number',
              } ]}>
              <InputNumber
                disabled={!checked}
                precision={0}
              />
            </Form.Item>
          </>
        );
      case HyperparameterType.Log:
        return (
          <>
            <Form.Item
              initialValue={hyperparameter.val as number}
              name={[ name, 'value' ]}
              noStyle
              rules={[ { required: !checked, type: 'number' } ]}>
              <InputNumber disabled={checked} />
            </Form.Item>
            <Form.Item
              initialValue={hyperparameter.minval}
              name={[ name, 'min' ]}
              noStyle
              rules={[ {
                max: form.getFieldValue([ name, 'max' ]),
                required: checked,
                type: 'number',
              } ]}>
              <InputNumber disabled={!checked} />
            </Form.Item>
            <Form.Item
              initialValue={hyperparameter.maxval}
              name={[ name, 'max' ]}
              noStyle
              rules={[ {
                min: form.getFieldValue([ name, 'min' ]),
                required: checked,
                type: 'number',
              } ]}>
              <InputNumber disabled={!checked} />
            </Form.Item>
          </>
        );
      case HyperparameterType.Categorical:
        return (
          <>
            <Form.Item
              initialValue={hyperparameter.val as string}
              name={[ name, 'value' ]}
              noStyle
              rules={[ { enum: hyperparameter.vals, required: true, type: 'enum' } ]}>
              <Input />
            </Form.Item>
            <Form.Item
              initialValue={hyperparameter.minval}
              name={[ name, 'min' ]}
              noStyle
              rules={[ { required: checked } ]}>
              <InputNumber disabled={!checked} />
            </Form.Item>
            <Form.Item
              initialValue={hyperparameter.maxval}
              name={[ name, 'max' ]}
              noStyle
              rules={[ { required: checked } ]}>
              <InputNumber disabled={!checked} />
            </Form.Item>
          </>
        );
    }
  }, [ checked,
    form,
    hyperparameter.maxval,
    hyperparameter.minval,
    hyperparameter.val,
    hyperparameter.vals,
    name,
    type ]);

  return (
    <>
      <Space className={css.hyperparameterName}>
        <Form.Item initialValue={false} name={[ name, 'active' ]} noStyle valuePropName="checked">
          <Checkbox onChange={handleCheck} />
        </Form.Item>
        <Typography.Title ellipsis={{ rows: 1, tooltip: true }} level={3}>{name}</Typography.Title>
      </Space>
      <Form.Item initialValue={hyperparameter.type} name={[ name, 'type' ]} noStyle>
        <Select value={type} onChange={handleTypeChange}>
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
      {inputs}
    </>
  );
};

export default useModalHyperparameterSearch;
