import { Alert, Checkbox, InputNumber, ModalFuncProps, Select } from 'antd';
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
import { ExperimentBase, Hyperparameter, ResourcePool } from 'types';

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
          id="search-method"
          value={searchMethod.name}
          onChange={handleSelectSearchMethod}>
          {Object.entries(SearchMethods).map(method =>
            <Select.Option key={method[0]} value={method[0]}>{method[1].name}</Select.Option>)}
        </SelectFilter>
        <p>{searchMethod.description}</p>
        <div>
          <h2>Hyperparameter</h2>
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
        <div><label htmlFor="resource-pool">Resource pool</label>
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
  }, [ experiment.id, newConfig ]);

  const handleOk = useCallback(() => {
    if (modalContent === page1) {
      setModalContent(page2);
    } else {
      submitExperiment();
    }
  }, [ submitExperiment, modalContent, page1, page2 ]);

  const okText = useMemo(() => {
    if (modalContent === page1) {
      return 'Select Resources';
    }
    return 'Run Experiment';
  }, [ modalContent, page1 ]);

  const modalProps: Partial<ModalFuncProps> = useMemo(() => {
    return {
      bodyStyle: { padding: 0 },
      className: css.base,
      closable: true,
      content: modalContent,
      icon: null,
      maskClosable: true,
      okText,
      onOk: handleOk,
      title: 'Hyperparameter Search',
    };
    //TODO: Back button in footer
  }, [ modalContent, okText, handleOk ]);

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

  const handleCheck = useCallback((e: CheckboxChangeEvent) => {
    setChecked(e.target.checked);
  }, []);

  const handleMinChange = useCallback((value: number) => {
    setMinVal(value);
  }, []);

  const handleMaxChange = useCallback((value: number) => {
    setMaxVal(value);
  }, []);

  return (
    <>
      <div>
        <Checkbox checked={checked} onChange={handleCheck} />
        {name}
      </div>
      <div>{hyperparameter.val}</div>
      <InputNumber disabled={!checked} value={minVal} onChange={handleMinChange} />
      <InputNumber disabled={!checked} value={maxVal} onChange={handleMaxChange} />
    </>
  );
};

export default useModalHyperparameterSearch;
