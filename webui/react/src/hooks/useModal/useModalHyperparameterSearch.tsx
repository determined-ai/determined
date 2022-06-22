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
import { DetError, isDetError } from 'shared/utils/error';
import { routeToReactUrl } from 'shared/utils/routes';
import { ExperimentBase, Hyperparameter, HyperparameterType, ResourcePool } from 'types';

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
  name: string;
}

const SearchMethods: Record<string, SearchMethod> = {
  ASHA: {
    description: `Automated HP search multi-trial experiment that will stop poor 
  performing trials early as it searches the HP space.`,
    name: 'Adaptive ASHA',
  },
  Grid: {
    description: `Brute force evaluates all possible hyperparameter configurations 
  and returns the best.`,
    name: 'Grid',
  },
  Random: {
    description: `Evaluates a set of hyperparameter configurations chosen at 
  random and returns the best.`,
    name: 'Random',
  },
} as const;

const useModalHyperparameterSearch = ({ experiment }: Props): ModalHooks => {
  const { modalClose, modalOpen: openOrUpdate, modalRef } = useModal();
  const [ modalError, setModalError ] = useState<string>();
  const [ searchMethod, setSearchMethod ] = useState(SearchMethods.ASHA);
  const { resourcePools } = useStore();
  const [ resourcePool, setResourcePool ] = useState<ResourcePool>();
  const [ form ] = Form.useForm();

  const handleSelectSearchMethod = useCallback((value: SelectValue) => {
    setSearchMethod(SearchMethods[value as string]);
  }, []);

  const hyperparameters = useMemo(() => {
    return Object.entries(experiment.hyperparameters)
      .map(hp => ({ hyperparameter: hp[1], name: hp[0] }));
  }, [ experiment.hyperparameters ]);

  const page1 = useMemo((): React.ReactNode => {
    // We always render the form regardless of mode to provide a reference to it.
    return (
      <div className={css.base}>
        {modalError && <Alert className={css.error} message={modalError} type="error" />}
        <p>
          Select the hyperparameter space and search method to
          optimize your model hyperparameters.
        </p>
        <div className={css.searchTitle}>
          <label htmlFor="search-method">Search method</label>
          <Link
            external
            path={paths.
              docs('/training-hyperparameter/index.html#specifying-the-search-algorithm')}>
            Learn more
          </Link>
        </div>
        <SelectFilter
          className={css.fullWidth}
          id="search-method"
          value={searchMethod.name}
          onChange={handleSelectSearchMethod}>
          {Object.entries(SearchMethods).map(method =>
            <Select.Option key={method[0]} value={method[0]}>{method[1].name}</Select.Option>)}
        </SelectFilter>
        <p>{searchMethod.description}</p>
        <Form form={form} name="hyperparameters">
          <div className={css.hyperparameterContainer}>
            <h2 className={css.hyperparameterName}>Hyperparameter</h2>
            <h2>Type</h2>
            <h2>Current</h2>
            <h2>Min value</h2>
            <h2>Max value</h2>
            {hyperparameters.map(hp => <HyperparameterRow form={form} key={hp.name} {...hp} />)}
          </div>
        </Form>
      </div>
    );
  }, [ form,
    handleSelectSearchMethod,
    hyperparameters,
    modalError,
    searchMethod.description,
    searchMethod.name ]);

  const handleSelectPool = useCallback((value: SelectValue) => {
    setResourcePool(resourcePools.find(pool => pool.imageId === value));
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
      <div className={css.base}>
        {modalError && <Alert className={css.error} message={modalError} type="error" />}
        <p>Select the resources to allocate to this search and the trial iteration limit.</p>
        <Form form={form} layout="vertical" requiredMark={false}>
          <div className={css.poolRow}>
            <Form.Item
              initialValue={resourcePools?.[0].imageId}
              label="Resource Pool"
              name="pool"
              rules={[ { required: true } ]}>
              <SelectFilter
                onChange={handleSelectPool}>
                {resourcePools.map(pool => (
                  <Select.Option key={pool.imageId} value={pool.imageId}>
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
              <InputNumber />
            </Form.Item>
          </div>
          <p>{maxSlots} max slots</p>
          <Form.Item
            initialValue={1}
            label="Max Trials"
            name="max-trials"
            rules={[ { min: 0, required: true, type: 'number' } ]}>
            <InputNumber className={css.fullWidth} />
          </Form.Item>
        </Form>
        <p>
          Note: HP search jobs (while more efficient than manual searching) can take longer
          depending on the size of the HP search space and the resources you allocate to it.
        </p>
      </div>
    );
  }, [ form,
    handleSelectPool,
    maxSlots,
    modalError,
    resourcePools ]);

  const [ modalContent, setModalContent ] = useState(page1);

  const newConfig = useMemo(() => {
    return yaml.dump(experiment.configRaw);
  }, [ experiment.configRaw ]);

  const submitExperiment = useCallback(async () => {
    try {
      const { id: newExperimentId } = await createExperiment({
        activate: true,
        experimentConfig: newConfig,
        parentId: experiment.id,
        projectId: experiment.projectId,
      });

      // Route to reload path to forcibly remount experiment page.
      const newPath = paths.experimentDetails(newExperimentId);
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
  }, [ experiment.id, experiment.projectId, newConfig ]);

  const handleOk = useCallback(() => {
    if (modalContent === page1) {
      setModalContent(page2);
    } else {
      submitExperiment();
    }
  }, [ submitExperiment, modalContent, page1, page2 ]);

  const handleBack = useCallback(() => {
    setModalContent(page1);
  }, [ page1 ]);

  const handleCancel = useCallback(() => {
    modalClose(ModalCloseReason.Cancel);
  }, [ modalClose ]);

  const footer = useMemo(() => {
    if (modalContent === page1) {
      return (
        <div className={css.footer}>
          <div className={css.spacer} />
          <Space>
            <Button onClick={handleCancel}>Cancel</Button>
            <Button type="primary" onClick={handleOk}>Select Resources</Button>
          </Space>
        </div>
      );
    }
    return (
      <div className={css.footer}>
        <Button onClick={handleBack}>Back</Button>
        <div className={css.spacer} />
        <Space>
          <Button onClick={handleCancel}>Cancel</Button>
          <Button type="primary" onClick={handleOk}>Run Experiment</Button>
        </Space>
      </div>
    );
  }, [ handleBack, handleCancel, handleOk, modalContent, page1 ]);

  const modalProps: Partial<ModalFuncProps> = useMemo(() => {
    return {
      bodyStyle: { padding: 0 },
      className: css.base,
      closable: true,
      content: <>{modalContent}{footer}</>,
      icon: null,
      title: 'Hyperparameter Search',
      width: 600,
    };
  }, [ modalContent, footer ]);

  const modalOpen = useCallback(({ initialModalProps }: ShowModalProps) => {
    setModalContent(page1);
    openOrUpdate({ ...modalProps, ...initialModalProps });
  }, [ modalProps, openOrUpdate, page1 ]);

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
