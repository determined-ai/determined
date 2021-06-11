import React, { useMemo } from 'react';

import HumanReadableFloat from 'components/HumanReadableFloat';
import {
  ExperimentBase, ExperimentHyperParam, ExperimentHyperParamType,
  RawJson, TrialDetails, TrialHyperParameters,
} from 'types';

import css from './TrialRangeHyperparameters.module.scss';

export interface Props {
  experiment: ExperimentBase;
  trial: TrialDetails;
}

interface HyperParameter {
  name: string,
  value: ExperimentHyperParam,
}

const TrialRangeHyperparameters: React.FC<Props> = ({ experiment, trial }: Props) => {
  const configSource: HyperParameter[] = useMemo(() => {
    return Object.entries(experiment.config.hyperparameters).map(([ name, value ]) => {
      return {
        name,
        value,
      };
    });
  }, [ experiment.config.hyperparameters ]);

  const valueSource: {name: string; value: number | string | boolean | RawJson}[] = useMemo(() => {
    return Object.entries(trial.hparams).map(([ name, value ]) => {
      return {
        name,
        value,
      };
    });
  }, [ trial.hparams ]);

  return (
    <div style={{ display: 'flex', gap: 20 }}>
      {configSource.map(hp => <div key={hp.name}>
        <HyperparameterRange
          config={hp}
          value={valueSource.find(hparam => hparam.name === hp.name) || { name: '', value: '' }} />
      </div>)}
    </div>
  );
};

interface RangeProps {
  config: HyperParameter
  value: TrialHyperParameters
}

const HyperparameterRange:React.FC<RangeProps> = ({ config, value }: RangeProps) => {
  return (
    <div className={css.container}>
      {config.name}
      <div className={css.innerContainer}>
        <div className={css.valuesTrack}>
          <p className={css.text}>{config.value.maxval}</p>
          <p className={css.text}>{config.value.minval}</p>
        </div>
        <div className={css.track}>
          {config.value.vals?.map(option =>
            <div
              className={css.trackOption}
              key={option.toString()}
            />)}
        </div>
        <div className={css.pointerTrack} style={{ height: `${100}%` }}>
          <div style={{ display: 'flex' }}>
            <div className={css.pointer} />
            <div className={css.valueBox}>
              <ParsedHumanReadableValue hp={value} type={config.value.type} />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

const ParsedHumanReadableValue = (hp: TrialHyperParameters, type: ExperimentHyperParamType) => {
  switch (type) {
    case ExperimentHyperParamType.Categorical:
      return <p className={css.text}>{hp.value}</p>;
    case ExperimentHyperParamType.Constant:
      return <p className={css.text}>{hp.value}</p>;
    case ExperimentHyperParamType.Double:
      return <HumanReadableFloat num={parseFloat(hp.value as string)} />;
    case ExperimentHyperParamType.Int:
      return <p className={css.text}>{parseInt(hp.value as string)}</p>;
    case ExperimentHyperParamType.Log:
      return <HumanReadableFloat num={parseFloat(hp.value as string)} />;
    default:
      return <p className={css.text}>Err</p>;
  }
};

export default TrialRangeHyperparameters;
