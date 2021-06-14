import { Tooltip } from 'antd';
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
  const pointerPosition = useMemo(() => {
    if (config.value.type === ExperimentHyperParamType.Constant) {
      return 50;
    } else if (config.value.type === ExperimentHyperParamType.Categorical
      && config.value.vals !== undefined) {
      return config.value.vals.length;
    } else if (config.value.type === ExperimentHyperParamType.Log) {
      return 50;
    } else {
      return 50;
    }
  }, [ config.value.type, config.value.vals ]);

  return (
    <div className={css.container}>
      {config.name}
      <div className={css.innerContainer}>
        <div className={css.valuesTrack}>
          {config.value.vals ?
            config.value.vals?.map(option =>
              <p className={css.text} key={option.toString()}>{option}</p>) :
            config.value.type === ExperimentHyperParamType.Log ?
              (new Array(
                Math.log10((config.value.maxval || 1)/(config.value.minval || 0)),
              )).fill(null)
                .map((_, idx) =>
                  <p className={css.text} key={idx}>
                    {String((config.value.maxval || 1)/(10**idx)).length > 4 ?
                      ((config.value.maxval || 1)/(10**idx)).toExponential() :
                      (config.value.maxval || 1)/(10**idx)}
                  </p>) :
              <>
                <p className={css.text}>{config.value.maxval}</p>
                <p className={css.text}>{config.value.minval}</p>
              </>
          }
        </div>
        <div
          className={
            (config.value.vals) ?
              css.grayTrack : config.value.type === ExperimentHyperParamType.Constant ?
                css.constantTrack : css.blueTrack
          }>
          {config.value.vals?.map(option =>
            <div
              className={css.trackOption}
              key={option.toString()}
            />)}
          {config.value.type === ExperimentHyperParamType.Constant &&
          <div
            className={css.trackOption}
          />}
        </div>
        <div className={css.pointerTrack}>
          <Pointer
            containerStyle={{ transform: `translateY(${270*pointerPosition/100}px)` }}
            content={<ParsedHumanReadableValue hp={value} type={config.value.type} />} />
        </div>
      </div>
    </div>
  );
};

interface PHRVProps {
  hp: TrialHyperParameters
  type: ExperimentHyperParamType
}

const ParsedHumanReadableValue: React.FC<PHRVProps> = ({ hp, type }: PHRVProps) => {
  switch (type) {
    case ExperimentHyperParamType.Categorical:
      return <p className={css.text}>{hp.value}</p>;
    case ExperimentHyperParamType.Constant:
      return <p className={css.text}>{hp.value}</p>;
    case ExperimentHyperParamType.Double:
      return (
        <p className={css.text}>
          <HumanReadableFloat num={parseFloat(hp.value as string)} precision={3} />
        </p>
      );
    case ExperimentHyperParamType.Int:
      return <p className={css.text}>{parseInt(hp.value as string)}</p>;
    case ExperimentHyperParamType.Log:
      return (
        <Tooltip title={hp.value}>
          <p className={css.text}>{parseFloat(hp.value as string).toExponential(2)}</p>
        </Tooltip>
      );
    default:
      return <p className={css.text}>{hp.value}</p>;
  }
};

interface PointerProps {
  containerStyle: React.CSSProperties;
  content: JSX.Element;
}

const Pointer: React.FC<PointerProps> = ({ containerStyle, content }: PointerProps) => {
  return (
    <div className={css.pointerContainer} style={containerStyle}>
      <div className={css.pointerArrow} />
      <div className={css.pointerText}>
        {content}
      </div>
    </div>
  );
};

export default TrialRangeHyperparameters;
