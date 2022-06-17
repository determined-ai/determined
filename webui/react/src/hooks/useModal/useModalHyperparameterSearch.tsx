import { Alert, Button, Checkbox, Input, InputNumber,
  ModalFuncProps, Select, Space, Typography } from 'antd';
import { CheckboxChangeEvent } from 'antd/lib/checkbox';
import { SelectValue } from 'antd/lib/select';
import yaml from 'js-yaml';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import SelectFilter from 'components/SelectFilter';
import { useStore } from 'contexts/Store';
import { paths } from 'routes/utils';
import { createExperiment } from 'services/api';
import { DetError, isDetError } from 'shared/utils/error';
import { routeToReactUrl } from 'shared/utils/routes';
import { ExperimentBase, Hyperparameter, HyperparameterType, ResourcePool } from 'types';

import useModal, { ModalHooks as Hooks } from './useModal';
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
  const { modalClose, modalOpen: openOrUpdate, modalRef } = useModal({});
  const [ modalError, setModalError ] = useState<string>();
  const [ searchMethod, setSearchMethod ] = useState(SearchMethods.ASHA);
  const { resourcePools } = useStore();
  const [ resourcePool, setResourcePool ] = useState<ResourcePool>();
  const [ slots, setSlots ] = useState<number>();
  const [ maxTrials, setMaxTrials ] = useState<number>();

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
        <div>
          <label htmlFor="search-method">Search method</label>
          <a>Learn more</a>
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
        <div className={css.hyperparameterContainer}>
          <h2 className={css.hyperparameterName}>Hyperparameter</h2>
          <h2>Type</h2>
          <h2>Current</h2>
          <h2>Min value</h2>
          <h2>Max value</h2>
          {hyperparameters.map(hp => <HyperparameterRow key={hp.name} {...hp} />)}
        </div>
      </div>
    );
  }, [ handleSelectSearchMethod,
    hyperparameters,
    modalError,
    searchMethod.description,
    searchMethod.name ]);

  const handleSelectPool = useCallback((value: SelectValue) => {
    setResourcePool(resourcePools.find(pool => pool.imageId === value));
  }, [ resourcePools ]);

  const handleChangeSlots = useCallback((value: number) => {
    setSlots(value);
  }, []);

  const handleChangeMaxTrials = useCallback((value: number) => {
    setMaxTrials(value);
  }, []);

  const page2 = useMemo((): React.ReactNode => {
    // We always render the form regardless of mode to provide a reference to it.
    return (
      <div className={css.base}>
        {modalError && <Alert className={css.error} message={modalError} type="error" />}
        <p>Select the resources to allocate to this search and the trial iteration limit.</p>
        <div>
          <label htmlFor="resource-pool">Resource pool</label>
          <SelectFilter
            id="resource-pool"
            value={resourcePool?.imageId}
            onChange={handleSelectPool}>
            {resourcePools.map(pool =>
              <Select.Option key={pool.imageId} value={pool.imageId}>{pool.name}</Select.Option>)}
          </SelectFilter>
          <label htmlFor="slots">Slots</label>
          <InputNumber id="slots" value={slots} onChange={handleChangeSlots} />
        </div>
        <p>TODO max slots</p>
        <label htmlFor="max-trials">Slots</label>
        <InputNumber id="max-trials" value={maxTrials} onChange={handleChangeMaxTrials} />
        <p>
          Note: HP search jobs (while more efficient than manual searching) can take longer
          depending on the size of the HP search space and the resources you allocate to it.
        </p>
      </div>
    );
  }, [ handleChangeMaxTrials,
    handleChangeSlots,
    handleSelectPool,
    maxTrials,
    modalError,
    resourcePool?.imageId,
    resourcePools,
    slots ]);

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
    //modalClose();
  }, [ ]);

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
      maskClosable: true,
      title: 'Hyperparameter Search',
      width: 600,
    };
    //TODO: Back button in footer
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
  hyperparameter: Hyperparameter;
  name: string;
}

const HyperparameterRow: React.FC<RowProps> = ({ hyperparameter, name }: RowProps) => {
  const [ checked, setChecked ] = useState(false);
  const [ minVal, setMinVal ] = useState<number>();
  const [ maxVal, setMaxVal ] = useState<number>();
  const [ type, setType ] = useState<HyperparameterType>(hyperparameter.type);

  const handleCheck = useCallback((e: CheckboxChangeEvent) => {
    setChecked(e.target.checked);
    if (e.target.checked) {
      setType(prev => prev === HyperparameterType.Constant ? HyperparameterType.Double : prev);
    } else {
      setType(HyperparameterType.Constant);
    }
  }, []);

  const handleMinChange = useCallback((value: number) => {
    setMinVal(value);
  }, []);

  const handleMaxChange = useCallback((value: number) => {
    setMaxVal(value);
  }, []);

  const handleTypeChange = useCallback((value: HyperparameterType) => {
    setType(value);
    if (value === HyperparameterType.Constant) setChecked(false);
    else setChecked(true);
  }, []);

  const typeSelect = useMemo(() => {
    return (
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
    );
  }, [ handleTypeChange, type ]);

  const inputs = useMemo(() => {
    switch (type) {
      case HyperparameterType.Constant:
        return (
          <>
            <InputNumber value={hyperparameter.val as number} />
            <InputNumber disabled={!checked} value={minVal} onChange={handleMinChange} />
            <InputNumber disabled={!checked} value={maxVal} onChange={handleMaxChange} />
          </>
        );
      case HyperparameterType.Double:
        return (
          <>
            <InputNumber value={hyperparameter.val as number} />
            <InputNumber disabled={!checked} value={minVal} onChange={handleMinChange} />
            <InputNumber disabled={!checked} value={maxVal} onChange={handleMaxChange} />
          </>
        );
      case HyperparameterType.Int:
        return (
          <>
            <InputNumber precision={0} value={hyperparameter.val as number} />
            <InputNumber disabled={!checked} value={minVal} onChange={handleMinChange} />
            <InputNumber disabled={!checked} value={maxVal} onChange={handleMaxChange} />
          </>
        );
      case HyperparameterType.Log:
        return (
          <>
            <InputNumber value={hyperparameter.val as number} />
            <InputNumber disabled={!checked} value={minVal} onChange={handleMinChange} />
            <InputNumber disabled={!checked} value={maxVal} onChange={handleMaxChange} />
          </>
        );
      case HyperparameterType.Categorical:
        return (
          <>
            <Input value={hyperparameter.val as string} />
            <InputNumber disabled={!checked} value={minVal} onChange={handleMinChange} />
            <InputNumber disabled={!checked} value={maxVal} onChange={handleMaxChange} />
          </>
        );
    }
  }, [ checked, handleMaxChange, handleMinChange, hyperparameter.val, maxVal, minVal, type ]);

  return (
    <>
      <Space className={css.hyperparameterName}>
        <Checkbox checked={checked} onChange={handleCheck} />
        <Typography.Title ellipsis={{ rows: 1, tooltip: true }} level={3}>{name}</Typography.Title>
      </Space>
      {typeSelect}
      {inputs}
    </>
  );
};

export default useModalHyperparameterSearch;
